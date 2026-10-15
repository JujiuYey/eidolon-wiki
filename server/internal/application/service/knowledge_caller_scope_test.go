package service

import (
	"context"
	"testing"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

// callerScopeKnowledgeRepo answers batch reads from memory with the same
// tenant scoping the SQL repository enforces.
type callerScopeKnowledgeRepo struct {
	interfaces.KnowledgeRepository
	rows []*types.Knowledge
}

func (r *callerScopeKnowledgeRepo) GetKnowledgeBatch(
	_ context.Context, tenantID uint64, ids []string,
) ([]*types.Knowledge, error) {
	want := make(map[string]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	var out []*types.Knowledge
	for _, row := range r.rows {
		if want[row.ID] && row.TenantID == tenantID {
			out = append(out, row)
		}
	}
	return out, nil
}

// TestCallerScopeFiltersDocumentsAndSearchScope pins the ownership-only
// access model end to end at the service layer: documents and search scope
// resolve under the caller's tenant, foreign-tenant KBs are denied or
// dropped, and an execution-tenant switch never widens any of it.
func TestCallerScopeFiltersDocumentsAndSearchScope(t *testing.T) {
	repo := &callerScopeKnowledgeRepo{rows: []*types.Knowledge{
		{ID: "own", TenantID: 1, KnowledgeBaseID: "own", Type: "file"},
		{ID: "foreign", TenantID: 2, KnowledgeBaseID: "foreign", Type: "file"},
		{ID: "other-foreign", TenantID: 3, KnowledgeBaseID: "other-foreign", Type: "file"},
	}}
	svc := &knowledgeService{repo: repo}
	kbs := []*types.KnowledgeBase{
		{ID: "own", TenantID: 1},
		{ID: "foreign", TenantID: 2},
		{ID: "other-foreign", TenantID: 3},
	}
	ids := []string{"own", "foreign", "other-foreign"}
	search := &knowledgeBaseService{kgRepo: repo}
	kbService := &suggestionKBService{kbs: make(map[string]*types.KnowledgeBase)}
	for _, kb := range kbs {
		kbService.kbs[kb.ID] = kb
	}
	session := &sessionService{knowledgeBaseService: kbService, knowledgeService: svc}
	base := newSharedAccessContext()

	rows, err := svc.GetKnowledgeBatch(base, 1, ids)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "own", rows[0].ID)

	knowledgeMap, err := search.fetchKnowledgeData(base, 1, ids)
	require.NoError(t, err)
	require.Len(t, knowledgeMap, 1)
	require.NotNil(t, knowledgeMap["own"])

	require.NoError(t, search.authorizeKBAccess(base, kbs[:1]))
	require.Error(t, search.authorizeKBAccess(base, kbs[1:2]), "a foreign KB is denied for the caller")
	execution := types.WithExecutionTenant(base, 2)
	require.Error(t, search.authorizeKBAccess(execution, kbs[1:2]), "an execution tenant switch is not ownership")

	targets, err := session.buildSearchTargets(
		base,
		1,
		ids,
		nil,
		[]types.TagScope{{KnowledgeBaseID: "foreign", TagIDs: []string{"tag"}}},
	)
	require.NoError(t, err)
	require.Len(t, targets, 1, "denied KBs must not become search or tag targets")
	require.Equal(t, "own", targets[0].KnowledgeBaseID)

	tenant, err := resolveKBReadTenant(base, kbs[0])
	require.NoError(t, err)
	require.Equal(t, uint64(1), tenant)
	_, err = resolveKBReadTenant(base, kbs[1])
	require.Error(t, err, "FAQ/tag reads cannot reach a foreign KB")
}

type callerScopeKBList struct {
	interfaces.KnowledgeBaseService
	tenant uint64
}

func (s *callerScopeKBList) ListKnowledgeBases(ctx context.Context) ([]*types.KnowledgeBase, error) {
	s.tenant = types.MustTenantIDFromContext(ctx)
	return []*types.KnowledgeBase{{ID: "own", TenantID: s.tenant, Type: types.KnowledgeBaseTypeDocument}}, nil
}

type callerScopeSearchRepo struct {
	interfaces.KnowledgeRepository
	scopes []types.KnowledgeSearchScope
}

func (r *callerScopeSearchRepo) SearchKnowledgeInScopes(
	_ context.Context,
	scopes []types.KnowledgeSearchScope,
	_ string,
	_, _ int,
	_ []string,
) ([]*types.Knowledge, bool, int64, error) {
	r.scopes = scopes
	return nil, false, 0, nil
}

func TestDocumentSearchListsCallerTenantAfterExecutionSwitch(t *testing.T) {
	kbs := &callerScopeKBList{}
	repo := &callerScopeSearchRepo{}
	svc := &knowledgeService{kbService: kbs, repo: repo}
	ctx := types.WithExecutionTenant(newSharedAccessContext(), 2)
	_, _, _, err := svc.SearchKnowledge(ctx, "query", 0, 10, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(1), kbs.tenant)
	require.Equal(t, []types.KnowledgeSearchScope{{TenantID: 1, KBID: "own"}}, repo.scopes)
	require.Equal(t, uint64(2), types.MustTenantIDFromContext(ctx), "search must not mutate the parent context")
}

// TestKnowledgeBatchIsTenantScopedAndAPIScopeStillNarrows pins that batch
// reads only ever see the caller's tenant, and that an API-key scope keeps
// narrowing access to owned KBs.
func TestKnowledgeBatchIsTenantScopedAndAPIScopeStillNarrows(t *testing.T) {
	svc := &knowledgeService{repo: &callerScopeKnowledgeRepo{rows: []*types.Knowledge{
		{ID: "own", TenantID: 1, KnowledgeBaseID: "own", Type: "file"},
		{ID: "also-own", TenantID: 1, KnowledgeBaseID: "also-own", Type: "file"},
		{ID: "foreign", TenantID: 2, KnowledgeBaseID: "foreign", Type: "file"},
	}}}
	ctx := newSharedAccessContext()
	rows, err := svc.GetKnowledgeBatch(ctx, 1, []string{"own", "also-own", "foreign"})
	require.NoError(t, err)
	require.Len(t, rows, 2)

	search := &knowledgeBaseService{}
	scoped := types.WithTenantAPIKeyScope(ctx, types.TenantAPIKeyScope{KnowledgeBaseIDs: types.StringArray{"other"}})
	err = search.authorizeKBAccess(scoped, []*types.KnowledgeBase{{ID: "own", TenantID: 1}})
	require.Error(t, err)
	app, ok := apperrors.IsAppError(err)
	require.True(t, ok)
	require.Equal(t, apperrors.ErrForbidden, app.Code, "API-key scope denial must not become a lookup failure")
}
