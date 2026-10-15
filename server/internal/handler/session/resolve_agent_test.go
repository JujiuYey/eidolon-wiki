package session

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type resolveOwnAgentStub struct {
	agent *types.CustomAgent
	err   error
}

func (s *resolveOwnAgentStub) GetAgentByID(context.Context, string) (*types.CustomAgent, error) {
	return s.agent, s.err
}

func (s *resolveOwnAgentStub) CreateAgent(context.Context, *types.CustomAgent) (*types.CustomAgent, error) {
	panic("not implemented")
}
func (s *resolveOwnAgentStub) GetAgentByIDAndTenant(context.Context, string, uint64) (*types.CustomAgent, error) {
	panic("not implemented")
}
func (s *resolveOwnAgentStub) ListAgents(context.Context) ([]*types.CustomAgent, error) {
	panic("not implemented")
}
func (s *resolveOwnAgentStub) UpdateAgent(
	context.Context, *types.CustomAgent, *string,
) (*types.CustomAgent, error) {
	panic("not implemented")
}
func (s *resolveOwnAgentStub) DeleteAgent(context.Context, string) error {
	panic("not implemented")
}
func (s *resolveOwnAgentStub) CopyAgent(context.Context, string) (*types.CustomAgent, error) {
	panic("not implemented")
}
func (s *resolveOwnAgentStub) GetSuggestedQuestions(context.Context, string, []string, []string, []types.TagScope, int) ([]types.SuggestedQuestion, error) {
	panic("not implemented")
}
func (s *resolveOwnAgentStub) GetKnowledgeSuggestedQuestions(context.Context, string, []string, []string, []types.TagScope, int) ([]types.SuggestedQuestion, error) {
	panic("not implemented")
}

func newResolveAgentTestContext(tenantID uint64) (*gin.Context, context.Context) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Set(types.UserIDContextKey.String(), "user-1")
	c.Set(types.TenantIDContextKey.String(), tenantID)
	ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, tenantID)
	c.Request = c.Request.WithContext(ctx)
	return c, ctx
}

func TestResolveAgent_EmptyIDReturnsNil(t *testing.T) {
	_, ctx := newResolveAgentTestContext(7)
	h := &Handler{customAgentService: &resolveOwnAgentStub{}}

	require.Nil(t, h.resolveAgent(ctx, ""))
}

// Agents resolve inside the caller's workspace: the service scopes the lookup
// to the tenant on the context.
func TestResolveAgent_ResolvesTheCallerAgent(t *testing.T) {
	_, ctx := newResolveAgentTestContext(7)
	agent := &types.CustomAgent{ID: "own-agent", TenantID: 7, Name: "Local"}
	h := &Handler{customAgentService: &resolveOwnAgentStub{agent: agent}}

	require.Same(t, agent, h.resolveAgent(ctx, "own-agent"))
}

func TestResolveAgent_UnfoundAgentReturnsNil(t *testing.T) {
	_, ctx := newResolveAgentTestContext(7)
	h := &Handler{customAgentService: &resolveOwnAgentStub{err: errors.New("agent not found")}}

	require.Nil(t, h.resolveAgent(ctx, "foreign-agent"))
}
