package access

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestResolveKBOwnershipMatrix(t *testing.T) {
	kb := &types.KnowledgeBase{ID: "kb", TenantID: 1}
	request := KBRequest{Caller: types.Caller{TenantID: 1, Role: types.TenantRoleViewer}}

	// Ownership resolves to admin regardless of the caller's workspace role
	// or the operation asked for; the requested permission is only recorded
	// as the grant's operation permission.
	grant, err := ResolveKB(context.Background(), request, kb, types.OrgRoleViewer)
	require.NoError(t, err)
	require.Equal(t, types.OrgRoleAdmin, grant.Permission)
	require.Equal(t, uint64(1), grant.EffectiveTenantID)
	require.Equal(t, uint64(1), grant.Caller.TenantID)

	grant, err = ResolveKB(context.Background(), request, kb, types.OrgRoleEditor)
	require.NoError(t, err)
	require.Equal(t, types.OrgRoleAdmin, grant.Permission)

	// Another workspace's KB is denied whatever role is asked for.
	foreign := &types.KnowledgeBase{ID: "kb", TenantID: 2}
	_, err = ResolveKB(context.Background(), request, foreign, types.OrgRoleViewer)
	require.ErrorIs(t, err, ErrForbidden)
	_, err = ResolveKB(context.Background(), request, foreign, types.OrgRoleEditor)
	require.ErrorIs(t, err, ErrForbidden)

	// The same KB resolves for its own owner under that owner's tenant.
	ownerRequest := KBRequest{Caller: types.Caller{TenantID: 2, Role: types.TenantRoleViewer}}
	grant, err = ResolveKB(context.Background(), ownerRequest, foreign, types.OrgRoleViewer)
	require.NoError(t, err)
	require.Equal(t, uint64(2), grant.EffectiveTenantID)

	// A request minted for one caller cannot be replayed under another
	// caller's context.
	mismatched := types.WithCaller(context.Background(), types.Caller{TenantID: 2, Role: types.TenantRoleViewer})
	_, err = ResolveKB(mismatched, request, kb, types.OrgRoleViewer)
	require.ErrorIs(t, err, ErrForbidden, "the request caller must match the caller in the context")
}

func TestResolveKBMissingIdentityResourceAndAPIKeyScope(t *testing.T) {
	kb := &types.KnowledgeBase{ID: "kb", TenantID: 1}
	_, err := ResolveKB(context.Background(), KBRequest{}, kb, types.OrgRoleViewer)
	require.ErrorIs(t, err, ErrUnauthorized)
	_, err = ResolveKB(context.Background(), KBRequest{Caller: types.Caller{TenantID: 1}}, nil, types.OrgRoleViewer)
	require.ErrorIs(t, err, ErrNotFound)
	ctx := types.WithTenantAPIKeyScope(
		context.Background(),
		types.TenantAPIKeyScope{KnowledgeBaseIDs: types.StringArray{"other"}},
	)
	_, err = ResolveKB(ctx, KBRequest{Caller: types.Caller{TenantID: 1}}, kb, types.OrgRoleViewer)
	require.Error(t, err, "API key scope must apply even to owned KBs")
}
