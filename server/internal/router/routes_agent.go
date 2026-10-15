package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/handler"
)

// RegisterCustomAgentRoutes registers custom agent routes.
//
// Mutating routes use OwnedAgentOrAdmin: the original creator can edit
// their agent, otherwise Admin+ is required. Built-in agents
// (IsBuiltin=true) have an empty creator and are always Admin+. Reads
// are Viewer+, copy is Contributor+ (the copy is owned by the caller).
func RegisterCustomAgentRoutes(r *gin.RouterGroup, agentHandler *handler.CustomAgentHandler, g *rbacGuards) {
	agents := g.apiKeyGroup(r.Group("/agents"), apiKeyFullAccess())
	// agentsRead are the agent read endpoints. They stay full-access only for
	// plain scoped keys (agent config can carry sensitive model bindings),
	// but read_agents, chat, or manage_agents may read them.
	agentsRead := agents.With(apiKeyReadAgents(apiKeyManageAgents(apiKeyChat(apiKeyFullAccess()))))
	// agentsWrite are the agent authoring endpoints. Owner by default, but a
	// key granted manage_agents may author agents without full Owner.
	agentsWrite := agents.With(apiKeyManageAgents(apiKeyFullAccess()))
	{
		// Get placeholder definitions (must be before /:id to avoid conflict) — Viewer+
		agentsRead.GET("/placeholders", g.Viewer(), agentHandler.GetPlaceholders)
		// List smart-reasoning agent type presets (rag-qa / wiki-qa / hybrid / custom) — Viewer+
		agentsRead.GET("/type-presets", g.Viewer(), agentHandler.GetAgentTypePresets)
		// Create custom agent — Contributor+
		agentsWrite.POST("", g.Contributor(), agentHandler.CreateAgent)
		// List all agents (including built-in) — Viewer+
		agentsRead.GET("", g.Viewer(), agentHandler.ListAgents)
		// Get agent by ID — Viewer+
		agentsRead.GET("/:id", g.Viewer(), agentHandler.GetAgent)
		// Update agent — creator OR Admin+
		agentsWrite.PUT("/:id", g.OwnedAgentOrAdmin(), agentHandler.UpdateAgent)
		// Delete agent — creator OR Admin+
		agentsWrite.DELETE("/:id", g.OwnedAgentOrAdmin(), agentHandler.DeleteAgent)
		// Copy agent — Contributor+ (copy is owned by the caller)
		agentsWrite.POST("/:id/copy", g.Contributor(), agentHandler.CopyAgent)
	}
	// Registered outside the group to avoid Gin route conflict with /agents/:id subroutes
	g.apiKeyRoute(r, http.MethodGet, "/agents/:id/suggested-questions",
		apiKeyReadAgents(apiKeyManageAgents(apiKeyChat(apiKeyFullAccess()))), g.Viewer(), agentHandler.GetSuggestedQuestions)
}

// RegisterUserFavoriteRoutes wires the per-user starred-resource endpoints.
//
// Authorization: the handler always derives (user_id, tenant_id) from the
// auth context — there is no admin-style "see another user's favorites"
// path — so a Viewer floor is the right gate. The endpoints intentionally
// don't follow the OwnedXOrAdmin pattern: favorites aren't owned by the
// resource's creator, they're owned by the user *doing* the starring.
func RegisterUserFavoriteRoutes(r *gin.RouterGroup, h *handler.UserResourceFavoriteHandler, g *rbacGuards) {
	// Favorites are per-user; not declared for API keys (default-deny).
	favs := r.Group("/user/favorites")
	{
		favs.GET("", g.Viewer(), h.ListFavorites)
		favs.POST("", g.Viewer(), h.AddFavorite)
		favs.DELETE("/:type/:id", g.Viewer(), h.RemoveFavorite)
	}
}
