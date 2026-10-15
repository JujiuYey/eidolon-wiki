package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type handlerKnowledgeAccessStub struct {
	interfaces.KnowledgeService
	knowledge *types.Knowledge
}

func (s *handlerKnowledgeAccessStub) GetKnowledgeByIDOnly(context.Context, string) (*types.Knowledge, error) {
	return s.knowledge, nil
}

func TestKBGuardAndHandlersShareResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, ginCallerKey := range []bool{true, false} {
		name := "request context only"
		if ginCallerKey {
			name = "both context surfaces"
		}
		t.Run(name, func(t *testing.T) {
			kb := &types.KnowledgeBase{ID: "kb", TenantID: 1}
			lookups := 0
			svc := &stubKBOnlyService{getByID: func(context.Context, string) (*types.KnowledgeBase, error) {
				lookups++
				return kb, nil
			}}
			kg := &handlerKnowledgeAccessStub{
				knowledge: &types.Knowledge{ID: "doc", KnowledgeBaseID: "kb", TenantID: 1},
			}
			h := &KnowledgeHandler{kbService: svc, kgService: kg}
			kbHandler := &KnowledgeBaseHandler{service: svc}
			r := gin.New()
			r.Use(middleware.ErrorHandler(), func(c *gin.Context) {
				ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, uint64(1))
				ctx = context.WithValue(ctx, types.TenantRoleContextKey, types.TenantRoleViewer)
				c.Request = c.Request.WithContext(ctx)
				if ginCallerKey {
					c.Set(types.TenantIDContextKey.String(), uint64(1))
				}
				c.Next()
			})
			r.GET("/:id", middleware.RequireKBAccess(
				middleware.KBIDFromParam("id"),
				types.OrgRoleViewer,
				svc,
			), func(c *gin.Context) {
				_, _, effective, permission, err := kbHandler.validateAndGetKnowledgeBase(c)
				require.NoError(t, err)
				require.Equal(t, uint64(1), effective)
				require.Equal(t, types.OrgRoleAdmin, permission, "ownership resolves to admin")
				_, _, _, permission, err = h.validateKnowledgeBaseAccess(c)
				require.NoError(t, err)
				require.Equal(t, types.OrgRoleAdmin, permission)
				_, ctx, err := h.resolveKnowledgeAndValidateKBAccess(c, "doc", types.OrgRoleViewer)
				require.NoError(t, err)
				require.Equal(t, uint64(1), types.MustTenantIDFromContext(ctx))
				require.Equal(t, uint64(1), types.CallerFromContext(ctx).TenantID)
				require.Equal(t, 1, lookups, "handlers must reuse the guarded KB")
				// The ownership grant covers writes too: the guarded grant is
				// reused for an Editor check without a second lookup.
				_, _, err = h.resolveKnowledgeAndValidateKBAccess(c, "doc", types.OrgRoleEditor)
				require.NoError(t, err)
				c.Status(http.StatusNoContent)
			})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/kb", nil))
			require.Equal(t, http.StatusNoContent, w.Code, w.Body.String())
		})
	}
}

func TestKBHandlerDoesNotReuseGrantForAnotherResourceOrCaller(t *testing.T) {
	for _, tt := range []struct {
		name, requestedID string
		caller            uint64
	}{
		{name: "another KB", requestedID: "other", caller: 1},
		{name: "another caller", requestedID: "kb", caller: 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			c.Set(types.TenantIDContextKey.String(), tt.caller)
			c.Set(middleware.KBAccessContextKey, &middleware.KBAccess{
				KnowledgeBase: &types.KnowledgeBase{ID: "kb", TenantID: 2},
				Caller: types.Caller{
					TenantID: 1,
					Role:     types.TenantRoleViewer,
				}, EffectiveTenantID: 2, Permission: types.OrgRoleAdmin,
			})
			svc := &stubKBOnlyService{getByID: func(context.Context, string) (*types.KnowledgeBase, error) {
				return &types.KnowledgeBase{ID: tt.requestedID, TenantID: 2}, nil
			}}
			_, err := resolveHandlerKBAccess(c, tt.requestedID, svc)
			require.Error(t, err)
		})
	}
}

func TestKBHandlerAPIKeyScopeStillAppliesToCachedGrant(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx := types.WithTenantAPIKeyScope(
		context.Background(),
		types.TenantAPIKeyScope{KnowledgeBaseIDs: types.StringArray{"other"}},
	)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	c.Set(types.TenantIDContextKey.String(), uint64(1))
	c.Set(middleware.KBAccessContextKey, &middleware.KBAccess{
		KnowledgeBase: &types.KnowledgeBase{ID: "kb", TenantID: 1},
		Caller: types.Caller{
			TenantID: 1,
			Role:     types.TenantRoleViewer,
		}, EffectiveTenantID: 1, Permission: types.OrgRoleAdmin,
	})
	_, err := resolveHandlerKBAccess(c, "kb", nil)
	require.Error(t, err)
}
