package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type suggestionTagRepo struct {
	interfaces.KnowledgeTagRepository
	tagsByTenant map[uint64][]*types.KnowledgeTag
}

func (r *suggestionTagRepo) GetByIDs(_ context.Context, tenantID uint64, ids []string) ([]*types.KnowledgeTag, error) {
	wanted := make(map[string]bool, len(ids))
	for _, id := range ids {
		wanted[id] = true
	}
	var result []*types.KnowledgeTag
	for _, tag := range r.tagsByTenant[tenantID] {
		if tag != nil && wanted[tag.ID] {
			result = append(result, tag)
		}
	}
	return result, nil
}

type suggestionKnowledgeRepo struct {
	interfaces.KnowledgeRepository
	idsByTenantAndKB map[uint64]map[string][]string
}

func (r *suggestionKnowledgeRepo) ListIDsByTagIDs(
	_ context.Context,
	tenantID uint64,
	kbID string,
	_ []string,
) ([]string, error) {
	return append([]string(nil), r.idsByTenantAndKB[tenantID][kbID]...), nil
}

type suggestionKBService struct {
	interfaces.KnowledgeBaseService
	kbs map[string]*types.KnowledgeBase
}

func (s *suggestionKBService) GetKnowledgeBasesByIDsOnly(
	_ context.Context,
	ids []string,
) ([]*types.KnowledgeBase, error) {
	result := make([]*types.KnowledgeBase, 0, len(ids))
	for _, id := range ids {
		if kb := s.kbs[id]; kb != nil {
			result = append(result, kb)
		}
	}
	return result, nil
}

func TestResolveSuggestionTagScopes_DropsKBsTheCallerCannotRead(t *testing.T) {
	const (
		callerTenant = uint64(1)
		kbID         = "own-kb"
		foreignKB    = "foreign-kb"
		tagID        = "own-tag"
	)
	svc := &customAgentService{
		tagRepo: &suggestionTagRepo{tagsByTenant: map[uint64][]*types.KnowledgeTag{
			callerTenant: {{ID: tagID, TenantID: callerTenant, KnowledgeBaseID: kbID}},
		}},
		knowledgeRepo: &suggestionKnowledgeRepo{idsByTenantAndKB: map[uint64]map[string][]string{
			callerTenant: {kbID: {"doc-in-tag"}},
		}},
		kbService: &suggestionKBService{kbs: map[string]*types.KnowledgeBase{
			kbID:      {ID: kbID, TenantID: callerTenant},
			foreignKB: {ID: foreignKB, TenantID: 2},
		}},
	}

	resolved, err := svc.resolveSuggestionTagScopes(
		context.WithValue(context.Background(), types.TenantIDContextKey, callerTenant),
		[]types.TagScope{
			{KnowledgeBaseID: kbID, TagIDs: []string{tagID}},
			{KnowledgeBaseID: foreignKB, TagIDs: []string{"foreign-tag"}},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, []string{kbID}, resolved.KnowledgeBaseIDs)
	assert.Equal(t, []string{"doc-in-tag"}, resolved.KnowledgeIDs)
	assert.Equal(t, []string{tagID}, resolved.TagIDsByTenant[callerTenant])
	assert.Empty(t, resolved.TagIDsByTenant[2], "a foreign KB must be dropped, not grouped under its own tenant")
}

func TestMergeHybridStarterSuggestions_ReservesKnowledgeSlots(t *testing.T) {
	curated := []types.SuggestedQuestion{
		{Question: "curated 1", Source: "agent_config"},
		{Question: "curated 2", Source: "agent_config"},
		{Question: "curated 3", Source: "agent_config"},
		{Question: "curated 4", Source: "agent_config"},
		{Question: "curated 5", Source: "agent_config"},
		{Question: "curated 6", Source: "agent_config"},
	}
	knowledge := []types.SuggestedQuestion{
		{Question: "knowledge 1", Source: "document"},
		{Question: "knowledge 2", Source: "faq"},
		{Question: "knowledge 3", Source: "document"},
	}

	got := mergeHybridStarterSuggestions(curated, knowledge, 6)
	require.Len(t, got, 6)
	assert.Equal(t, []string{
		"curated 1", "curated 2", "curated 3", "curated 4", "knowledge 1", "knowledge 2",
	}, []string{got[0].Question, got[1].Question, got[2].Question, got[3].Question, got[4].Question, got[5].Question})
}

func TestMergeHybridStarterSuggestions_BackfillsWhenKnowledgeIsEmpty(t *testing.T) {
	curated := []types.SuggestedQuestion{
		{Question: "curated 1"}, {Question: "curated 2"}, {Question: "curated 3"},
	}
	got := mergeHybridStarterSuggestions(curated, nil, 3)
	require.Len(t, got, 3)
	assert.Equal(t, []string{"curated 1", "curated 2", "curated 3"}, []string{
		got[0].Question, got[1].Question, got[2].Question,
	})
}

func TestExcludeSuggestionStrings_TagScopeOverridesSameKnowledgeBase(t *testing.T) {
	got := excludeSuggestionStrings([]string{"kb-with-tag", "kb-explicit"}, []string{"kb-with-tag"})
	assert.Equal(t, []string{"kb-explicit"}, got)
}

type suggestionDocRepo struct {
	interfaces.KnowledgeRepository
	docs map[string]*types.Knowledge
}

func (r *suggestionDocRepo) GetKnowledgeByIDOnly(_ context.Context, id string) (*types.Knowledge, error) {
	if doc, ok := r.docs[id]; ok {
		return doc, nil
	}
	return nil, errors.New("not found")
}

// Document IDs in a suggestion scope reach a tenant's chunks directly, so each
// must be readable by the caller, as KB IDs already are: a suggestion
// snapshot may record IDs from a session that ran under another tenant.
func TestReadableSuggestionKnowledgeIDs(t *testing.T) {
	svc := &customAgentService{knowledgeRepo: &suggestionDocRepo{docs: map[string]*types.Knowledge{
		"own-doc":     {ID: "own-doc", TenantID: 7, KnowledgeBaseID: "own-kb"},
		"foreign-doc": {ID: "foreign-doc", TenantID: 84, KnowledgeBaseID: "agent-kb"},
	}}}
	caller := types.WithCaller(context.Background(),
		types.Caller{TenantID: 7, UserID: "u", Role: types.TenantRoleAdmin})
	ids := []string{"own-doc", "foreign-doc", "missing-doc"}

	assert.Equal(t, []string{"own-doc"}, svc.readableSuggestionKnowledgeIDs(caller, ids))

	// Follow-up generation moves execution into another workspace; that does
	// not make a foreign workspace's documents readable.
	assert.Equal(t, []string{"own-doc"},
		svc.readableSuggestionKnowledgeIDs(types.WithExecutionTenant(caller, 84), ids),
		"execution in a foreign workspace does not make its documents the caller's")
}
