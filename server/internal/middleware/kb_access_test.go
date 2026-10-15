package middleware

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	apprepo "github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// stubKBLookup is a tiny KBLookup stand-in for tests; satisfies the
// KBLookup interface (a single method) without dragging in the full
// KnowledgeBaseService surface.
type stubKBLookup struct {
	kbs    map[string]*types.KnowledgeBase
	getErr error
}

func (s *stubKBLookup) GetKnowledgeBaseByID(_ context.Context, id string) (*types.KnowledgeBase, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if kb, ok := s.kbs[id]; ok {
		return kb, nil
	}
	return nil, apprepo.ErrKnowledgeBaseNotFound
}

// runGuard fires a single request through the guard and returns the
// gin recorder + the kb access (if any) the guard stashed.
func runGuard(
	t *testing.T,
	tenantID uint64,
	kbID string,
	requiredPerm types.OrgMemberRole,
	kb *types.KnowledgeBase,
) (*httptest.ResponseRecorder, *gin.Context) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "id", Value: kbID}}

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), types.TenantIDContextKey, tenantID)
	c.Request = req.WithContext(ctx)

	kbsvc := &stubKBLookup{kbs: map[string]*types.KnowledgeBase{}}
	if kb != nil {
		kbsvc.kbs[kbID] = kb
	}

	guard := RequireKBAccess(
		KBIDFromParam("id"),
		requiredPerm,
		kbsvc,
	)
	guard(c)
	return rec, c
}

func TestRequireKBAccess_OwnKB(t *testing.T) {
	rec, c := runGuard(t, 100, "kb-1",
		types.OrgRoleViewer,
		&types.KnowledgeBase{ID: "kb-1", TenantID: 100},
	)
	require.False(t, c.IsAborted(), "should pass through")
	require.Equal(t, 200, rec.Code) // gin's default; nothing wrote a status
	access, ok := KBAccessFromContext(c)
	require.True(t, ok)
	require.Equal(t, uint64(100), access.EffectiveTenantID)
	require.Equal(t, types.OrgRoleAdmin, access.Permission, "own KB grants admin")
	// The request context's tenant should still be the caller's own.
	got, ok := types.TenantIDFromContext(c.Request.Context())
	require.True(t, ok)
	require.Equal(t, uint64(100), got)
}

// TestIsResourceNotFound_RecognisesKnowledgeSentinel pins that a missing
// *document* (knowledge) is treated as not-found, not a transient error.
// Regression: ErrKnowledgeNotFound was absent from the predicate, so
// GET/DELETE /knowledge/:id and chunk list resolved a missing doc into a
// raw 500 instead of a 404 — which the CLI then surfaced as a retryable
// server.error (exit 7), looping agents on a permanently-absent doc.
func TestIsResourceNotFound_RecognisesKnowledgeSentinel(t *testing.T) {
	require.True(t, isResourceNotFound(apprepo.ErrKnowledgeNotFound),
		"missing document (ErrKnowledgeNotFound) must classify as not-found")
	require.True(t, isResourceNotFound(apprepo.ErrKnowledgeBaseNotFound),
		"missing KB must still classify as not-found")
	require.True(t, isResourceNotFound(apprepo.ErrChunkNotFound),
		"missing chunk (ErrChunkNotFound) must classify as not-found — chunk view/by-id resolved a missing chunk into a raw 500 (exit 7) otherwise")
	require.True(t, isResourceNotFound(ErrResourceNotFound),
		"generic resource-not-found sentinel must still classify as not-found")
	require.False(t, isResourceNotFound(errors.New("connection refused")),
		"a genuine transient error must NOT be classified as not-found")
}

func TestRequireKBAccess_NotFound_Aborts(t *testing.T) {
	_, c := runGuard(t, 100, "kb-missing", types.OrgRoleViewer, nil)
	require.True(t, c.IsAborted(), "missing KB must abort")
	require.NotEmpty(t, c.Errors)
	_, ok := KBAccessFromContext(c)
	require.False(t, ok, "no access should be stashed on failure")
}

func TestRequireKBAccess_ForeignKB_Aborts(t *testing.T) {
	// Ownership-only resolution: another workspace's KB is denied before
	// any grant can be minted, so a stale or foreign ID on a gated route
	// never widens into cross-workspace reads.
	_, c := runGuard(t, 100, "kb-foreign",
		types.OrgRoleViewer,
		&types.KnowledgeBase{ID: "kb-foreign", TenantID: 200},
	)
	require.True(t, c.IsAborted(), "another workspace's KB must abort")
	_, ok := KBAccessFromContext(c)
	require.False(t, ok, "no access should be stashed on denial")
}

func TestRequireKBAccess_NoTenant_Aborts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "id", Value: "kb-x"}}
	c.Request = httptest.NewRequest("GET", "/", nil) // no tenant in context
	guard := RequireKBAccess(
		KBIDFromParam("id"),
		types.OrgRoleViewer,
		&stubKBLookup{},
	)
	guard(c)
	require.True(t, c.IsAborted())
}
