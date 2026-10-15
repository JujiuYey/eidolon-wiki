package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestKBWritableIDs(t *testing.T) {
	targets := types.SearchTargets{
		{KnowledgeBaseID: "own", TenantID: 42},
		{KnowledgeBaseID: "foreign-editor", TenantID: 7},
		{KnowledgeBaseID: "foreign-viewer", TenantID: 7},
		{KnowledgeBaseID: "agent-scope", TenantID: 7},
	}
	caller := func(role types.TenantRole, userID string) context.Context {
		return types.WithCaller(context.Background(), types.Caller{TenantID: 42, UserID: userID, Role: role})
	}
	apiKey := func(scope types.TenantAPIKeyScope) context.Context {
		return types.WithTenantAPIKeyScope(caller(types.TenantRoleViewer, "system-42"), scope)
	}

	// Only the caller's own workspace is writable, whatever role it holds
	// there; a foreign workspace's KB is never writable.
	require.Equal(t, []string{"own"},
		kbWritableIDs(caller(types.TenantRoleContributor, "u"), targets))
	require.Equal(t, []string{"own"}, kbWritableIDs(caller(types.TenantRoleAdmin, "u"), targets))
	require.Equal(t, []string{"own"}, kbWritableIDs(caller(types.TenantRoleContributor, ""), targets),
		"callers without a user stay bounded to their own workspace")

	// A tenant Viewer (also the IM / embed / MCP endpoint principals) is
	// read-only, like on the HTTP write routes.
	require.Empty(t, kbWritableIDs(caller(types.TenantRoleViewer, "u"), targets))

	// Scoped API keys write only with the ingest capability.
	chatOnly := types.TenantAPIKeyScope{Capabilities: types.StringArray{string(types.APIKeyCapabilityChat)}}
	require.Empty(t, kbWritableIDs(apiKey(chatOnly), targets))
	ingest := types.TenantAPIKeyScope{Capabilities: types.StringArray{string(types.APIKeyCapabilityIngest)}}
	require.Equal(t, []string{"own"}, kbWritableIDs(apiKey(ingest), targets))
}
