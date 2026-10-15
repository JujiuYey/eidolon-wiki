package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/Tencent/WeKnora/internal/types"
)

// RegisterTenantRoutes 注册空间相关的路由
//
// Tenant-internal RBAC for /tenants/:id:
//   - GET   /:id          Viewer+ (read tenant settings)
//   - PUT   /:id          Owner+ (mutate tenant config)
//   - GET/POST/PUT/DELETE /:id/api-keys   Owner+ (scoped API key management)
//
// All /tenants/:id endpoints share g.PathTenantMatch() at the group
// level: middleware/access.go enforces "URL :id == active tenant"
// so an Owner-of-A cannot drive operations against tenant B by changing
// the URL. This used to be authorizeTenantAccess in tenant.go;
// collapsing it into one route guard means the declaration itself
// documents the rule.
//
// Tenant creation/deletion and member management live outside the
// slimmed deployment: signup provisions the personal workspace via
// UserService.Register (CreateTenant + EnsureOwner with rollback), so
// there is no self-service POST /tenants surface to expose.
func RegisterTenantRoutes(
	r *gin.RouterGroup,
	handler *handler.TenantHandler,
	g *rbacGuards,
) {
	// 空间路由组
	tenantRoutes := r.Group("/tenants")
	{
		g.apiKeyRoute(tenantRoutes, http.MethodGet, "", apiKeyManageTenantSettings(apiKeyFullAccess()), handler.ListTenants)

		// Generic KV configuration management (tenant-level). Tenant ID
		// is obtained from authentication context; the URL :key is a
		// config key, not a tenant ID, so these stay outside the
		// PathTenantMatch group. Tenant-level surface: full-access keys may
		// call it, and scoped keys need manage_tenant_settings.
		g.apiKeyRoute(tenantRoutes, http.MethodGet, "/kv/:key", apiKeyManageTenantSettings(apiKeyFullAccess()), g.Viewer(), handler.GetTenantKV)
		g.apiKeyRoute(tenantRoutes, http.MethodPut, "/kv/:key", apiKeyManageTenantSettings(apiKeyFullAccess()), g.Admin(), handler.UpdateTenantKV)

		// Per-tenant endpoints share PathTenantMatch at the group level.
		// Most /tenants/:id/* endpoints stay undeclared for API keys by
		// default — tenant lifecycle and key/principal management require
		// full tenant access or JWT ownership.
		tenantByID := tenantRoutes.Group("/:id", g.PathTenantMatch())
		{
			g.apiKeyRoute(tenantByID, http.MethodGet, "",
				apiKeyPlatform(types.APIKeyCapabilitySystemTenantsRead, types.APIKeyCapabilitySystemTenantsManage),
				g.Viewer(), handler.GetTenant)
			g.apiKeyRoute(tenantByID, http.MethodPut, "",
				apiKeyPlatform(types.APIKeyCapabilitySystemTenantsManage), g.Owner(), handler.UpdateTenant)
			tenantByID.GET("/api-keys", g.Owner(), handler.ListAPIKeys)
			tenantByID.POST("/api-keys", g.Owner(), handler.CreateAPIKey)
			tenantByID.PUT("/api-keys/:key_id", g.Owner(), handler.UpdateAPIKey)
			tenantByID.DELETE("/api-keys/:key_id", g.Owner(), handler.DeleteAPIKey)
			tenantByID.GET("/api-principal-config", g.Owner(), handler.GetAPIPrincipalConfig)
			tenantByID.PUT("/api-principal-config", g.Owner(), handler.UpdateAPIPrincipalConfig)
			tenantByID.POST("/api-principal-test-token", g.Owner(), handler.CreateAPIPrincipalTestToken)
		}
	}
}

// RegisterAuthRoutes registers authentication routes
func RegisterAuthRoutes(r *gin.RouterGroup, handler *handler.AuthHandler, g *rbacGuards) {
	r.POST("/auth/register", handler.Register)
	r.POST("/auth/login", handler.Login)
	r.POST("/auth/auto-setup", handler.AutoSetup)
	r.GET("/auth/config", handler.GetAuthConfig)
	r.POST("/auth/refresh", handler.RefreshToken)
	r.GET("/auth/validate", handler.ValidateToken)
	r.POST("/auth/logout", handler.Logout)
	// auth/me returns only the caller's own identity/profile, so it is safe
	// for any valid API key. Chat clients / MCP call it to discover "who am I";
	// leaving it default-deny was why scoped keys got a 403 here.
	g.apiKeyRoute(r, http.MethodGet, "/auth/me", apiKeyAny(), handler.GetCurrentUser)
	r.PUT("/auth/me/preferences", handler.UpdateMyPreferences)
	r.POST("/auth/change-password", handler.ChangePassword)
}

// RegisterSystemRoutes registers system information routes
//
// Reads (GetSystemInfo / ListParserEngines / GetStorageEngineStatus)
// are gated to Viewer+ — any tenant member can see "is the parser
// reachable". The /*-check / /reconnect endpoints actively probe
// remote services with tenant credentials and could trigger network
// fanout, so they're Admin+.
func RegisterSystemRoutes(
	r *gin.RouterGroup,
	handler *handler.SystemHandler,
	g *rbacGuards,
) {
	systemRoutes := g.apiKeyGroup(r.Group("/system"), apiKeyManageVectorStores(apiKeyFullAccess()))
	{
		systemRoutes.GET("/info", g.Viewer(), handler.GetSystemInfo)
		systemRoutes.GET("/parser-engines", g.Viewer(), handler.ListParserEngines)
		systemRoutes.POST("/parser-engines/check", g.Admin(), handler.CheckParserEngines)
		systemRoutes.POST("/docreader/reconnect", g.Admin(), handler.ReconnectDocReader)
		systemRoutes.GET("/storage-engine-status", g.Viewer(), handler.GetStorageEngineStatus)
		systemRoutes.POST("/storage-engine-check", g.Admin(), handler.CheckStorageEngine)
	}
}

// RegisterSystemAdminRoutes registers system administration routes.
//
// All endpoints under this group are gated to SystemAdmin users (i.e.
// User.IsSystemAdmin == true). These are platform-wide operations
// independent of per-tenant Owner/Admin/Contributor/Viewer roles —
// they let org-level superusers grant/revoke system-admin status and,
// in later milestones, will host global settings, built-in models, and
// cross-tenant observability.
//
// Mounted under /api/v1/system/admin/* so the URL scheme stays aligned
// with the existing /api/v1/system/info family. Front-end clients live
// in frontend/src/api/system/index.ts.
func RegisterSystemAdminRoutes(
	r *gin.RouterGroup,
	handler *handler.SystemHandler,
	g *rbacGuards,
) {
	// Apply SystemAdmin() at the group level — every route below inherits
	// the guard, so adding new endpoints can't accidentally drop the gate.
	adminRoutes := r.Group("/system/admin", g.SystemAdmin())
	{
		// Catalog mutation is reserved to authenticated system-admin users.
		// API keys remain default-denied by the API-key gate.
		adminRoutes.GET("/model-catalog", handler.GetModelCatalog)
		adminRoutes.POST("/model-catalog/preview", handler.PreviewModelCatalog)
		adminRoutes.PUT("/model-catalog", handler.PublishModelCatalog)

		// P0: SystemAdmin role management
		adminRoutes.POST("/promote", handler.PromoteUserToSystemAdmin)
		adminRoutes.POST("/revoke", handler.RevokeSystemAdmin)
		adminRoutes.GET("/list", handler.ListSystemAdmins)
		adminRoutes.POST("/users/reset-password", handler.ResetUserPassword)
		adminRoutes.POST("/users/create", handler.CreateSystemUser)
		adminRoutes.GET("/api-keys", handler.ListPlatformAPIKeys)
		adminRoutes.POST("/api-keys", handler.CreatePlatformAPIKey)
		adminRoutes.DELETE("/api-keys/:key_id", handler.DeletePlatformAPIKey)

		// P1: platform-wide system settings (DB-backed runtime tunables).
		// Reads return raw model rows / arrays (no `gin.H{"data":...}`
		// wrapping), matching the project's axios interceptor convention
		// — see frontend/src/utils/request.ts:97.
		g.apiKeyRoute(adminRoutes, http.MethodGet, "/settings",
			apiKeyPlatform(types.APIKeyCapabilitySystemSettingsRead, types.APIKeyCapabilitySystemSettingsManage),
			handler.ListSystemSettings)
		g.apiKeyRoute(adminRoutes, http.MethodGet, "/settings/:key",
			apiKeyPlatform(types.APIKeyCapabilitySystemSettingsRead, types.APIKeyCapabilitySystemSettingsManage),
			handler.GetSystemSetting)
		g.apiKeyRoute(adminRoutes, http.MethodPut, "/settings/:key",
			apiKeyPlatform(types.APIKeyCapabilitySystemSettingsManage), handler.UpdateSystemSetting)
		g.apiKeyRoute(adminRoutes, http.MethodDelete, "/settings/:key",
			apiKeyPlatform(types.APIKeyCapabilitySystemSettingsManage), handler.ResetSystemSetting)

		// Runtime operations: live asynq queue depths, safe task projections,
		// and state-checked task actions for the SystemAdmin dashboard. Lite
		// mode returns available=false.
		g.apiKeyRoute(adminRoutes, http.MethodGet, "/runtime/queues",
			apiKeyPlatform(types.APIKeyCapabilitySystemRuntimeRead, types.APIKeyCapabilitySystemRuntimeManage),
			handler.GetRuntimeQueues)
		g.apiKeyRoute(adminRoutes, http.MethodGet, "/runtime/queues/:queue/tasks",
			apiKeyPlatform(types.APIKeyCapabilitySystemRuntimeRead, types.APIKeyCapabilitySystemRuntimeManage),
			handler.ListRuntimeTasks)
		g.apiKeyRoute(adminRoutes, http.MethodPost, "/runtime/queues/:queue/tasks/:task_id/actions/:action",
			apiKeyPlatform(types.APIKeyCapabilitySystemRuntimeManage), handler.MutateRuntimeTask)
		g.apiKeyRoute(adminRoutes, http.MethodDelete, "/runtime/queues/:queue/archived",
			apiKeyPlatform(types.APIKeyCapabilitySystemRuntimeManage), handler.PurgeArchivedRuntimeTasks)

		// Bulk action — write the current default-quota setting onto
		// every existing tenant. Lives under /tenants instead of
		// /settings because it changes tenants, not the setting row.
		g.apiKeyRoute(adminRoutes, http.MethodPost, "/tenants/apply-default-storage-quota",
			apiKeyPlatform(types.APIKeyCapabilitySystemTenantsManage),
			handler.ApplyDefaultStorageQuotaToAllTenants)

	}
}
