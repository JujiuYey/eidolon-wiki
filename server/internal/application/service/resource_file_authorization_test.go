package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/access"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

func TestCatalogFileAuthorizationRejectsPrivateHandlesInSharedText(t *testing.T) {
	catalog, db := newResourceCatalogForTest(t)
	require.NoError(t, db.AutoMigrate(&types.KnowledgeBase{}, &types.Knowledge{}, &types.Chunk{}, &types.WikiPage{}))
	ctx := newSharedAccessContext()
	const physical = "local://7/exports/private.png"
	ref, err := catalog.Register(ctx, 7, physical, interfaces.ResourceRegistration{})
	require.NoError(t, err)
	for _, kb := range []string{"private", "shared"} {
		require.NoError(t, db.Create(&types.KnowledgeBase{ID: kb, TenantID: 7}).Error)
		require.NoError(
			t,
			db.Create(&types.Knowledge{ID: kb + "-doc", TenantID: 7, KnowledgeBaseID: kb, Type: "file"}).Error,
		)
	}
	require.NoError(
		t,
		catalog.Bind(ctx, ref, types.ResourceOwnerKnowledge, "private-doc", types.ResourceRelationSourceFile),
	)
	text := "![private](" + ref + ") ![legacy](" + physical + ")"
	require.NoError(
		t,
		db.Create(
			&types.Chunk{ID: "chunk", TenantID: 7, KnowledgeBaseID: "shared", KnowledgeID: "shared-doc", Content: text},
		).Error,
	)
	require.NoError(
		t,
		db.Create(&types.WikiPage{ID: "wiki", TenantID: 7, KnowledgeBaseID: "shared", Content: text}).Error,
	)
	grant := &access.KBAccess{
		KnowledgeBase:     &types.KnowledgeBase{ID: "shared", TenantID: 7},
		Caller:            types.CallerFromContext(ctx),
		EffectiveTenantID: 7,
		Permission:        types.OrgRoleViewer,
	}
	lookup := catalog.(interfaces.KBResourceLookup)

	// A manually minted cross-workspace grant still requires an exact live
	// binding between the resource and the granted KB: retrieval text naming
	// the handle must not authorize a file bound to another KB.
	checkKB := func(allowed bool) {
		t.Helper()
		for _, path := range []string{ref, physical} {
			_, err := access.ResolveKBFile(ctx, grant, "shared", path, catalog, lookup)
			if allowed {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, access.ErrForbidden)
			}
		}
	}
	checkKB(false)
	require.NoError(
		t,
		catalog.Bind(ctx, ref, types.ResourceOwnerKnowledge, "shared-doc", types.ResourceRelationSourceFile),
	)
	checkKB(true)
	require.NoError(t, db.Where("id = ?", "shared").Delete(&types.KnowledgeBase{}).Error)
	checkKB(false)

	// Message-file authorization stays ownership-bound: the tenant that owns
	// the resource serves it on evidence, another workspace's caller never
	// does, and without a reference there is nothing to serve.
	message := &types.Message{
		ID:                  "message",
		AgentTenantID:       7,
		Content:             text,
		KnowledgeReferences: []*types.SearchResult{{KnowledgeBaseID: "shared", Content: text}},
	}
	owner := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(7))
	_, err = access.AuthorizeMessageFile(owner, message, ref, catalog)
	require.NoError(t, err)
	_, err = access.AuthorizeMessageFile(ctx, message, ref, catalog)
	require.ErrorIs(t, err, access.ErrForbidden, "a caller from another workspace is never the owner")
	message.KnowledgeReferences = nil
	message.Content = ""
	_, err = access.AuthorizeMessageFile(owner, message, ref, catalog)
	require.ErrorIs(t, err, access.ErrForbidden, "no reference, no file")
}
