package router

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/dig"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/Tencent/WeKnora/internal/handler/session"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/mcpserver"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// RouterParams 路由参数
type RouterParams struct {
	dig.In

	Config                   *config.Config
	FileService              interfaces.FileService
	UserService              interfaces.UserService
	KBService                interfaces.KnowledgeBaseService
	KnowledgeService         interfaces.KnowledgeService
	ChunkService             interfaces.ChunkService
	SessionService           interfaces.SessionService
	MessageService           interfaces.MessageService
	ModelService             interfaces.ModelService
	KBHandler                *handler.KnowledgeBaseHandler
	KnowledgeHandler         *handler.KnowledgeHandler
	TenantHandler            *handler.TenantHandler
	TenantService            interfaces.TenantService
	TenantAPIKeyService      interfaces.TenantAPIKeyService
	TenantMemberService      interfaces.TenantMemberService
	ChunkHandler             *handler.ChunkHandler
	SessionHandler           *session.Handler
	MessageHandler           *handler.MessageHandler
	MessageSuggestionHandler *handler.MessageSuggestionHandler
	ModelHandler             *handler.ModelHandler
	ModelCredentialsHandler  *handler.ModelCredentialsHandler
	AuthHandler              *handler.AuthHandler
	InitializationHandler    *handler.InitializationHandler
	SystemHandler            *handler.SystemHandler
	VectorStoreHandler       *handler.VectorStoreHandler
	StorageBackendHandler    *handler.StorageBackendHandler
	StorageBackendResolver   interfaces.StorageBackendResolver
	ResourceCatalog          interfaces.ResourceCatalog
	FAQHandler               *handler.FAQHandler
	TagHandler               *handler.TagHandler
	CustomAgentHandler       *handler.CustomAgentHandler
	UserFavoriteHandler      *handler.UserResourceFavoriteHandler
	MCPEndpointHandler       *handler.MCPEndpointHandler
	MCPEndpointService       interfaces.MCPEndpointService
	MCPServer                *mcpserver.Server
	RedisClient              *redis.Client
	WeKnoraCloudHandler      *handler.WeKnoraCloudHandler
	WikiPageHandler          *handler.WikiPageHandler
}

// NewRouter 创建新的路由
func NewRouter(params RouterParams) *gin.Engine {
	r := gin.New()
	r.ContextWithFallback = true
	// 清理 FormFile/MultipartForm 解析产生的 multipart 临时文件，避免容器 /tmp 持续增长。
	r.Use(middleware.MultipartFormCleanup())

	// Trusted proxies: gin defaults to trusting ALL proxies, which makes
	// c.ClientIP() honor a client-supplied X-Forwarded-For. Public, unauthed
	// surfaces (resource grants, presigned files) rate-limit per ClientIP, so
	// a spoofed XFF would trivially bypass the limiter. Restrict to the
	// fronting proxy network so only the real client IP (appended by nginx) is
	// returned. Configurable via WEKNORA_TRUSTED_PROXIES (comma-separated
	// CIDRs/IPs).
	if err := r.SetTrustedProxies(trustedProxies()); err != nil {
		logger.Errorf(context.Background(), "[Router] failed to set trusted proxies: %v", err)
	}

	// CORS 中间件应放在最前面。
	// 注意：通配符 AllowOrigins 下浏览器会拒绝一切带凭据（cookie）的跨域
	// 请求（CORS 规范禁止 "*" 与 credentials 组合），因此 AllowCredentials
	// 实际只对未来改为回显具体 Origin 时才生效；当前认证全部走显式的
	// Authorization / X-API-Key 头，不依赖 ambient 凭据。若引入 cookie
	// 认证，必须先把 AllowOrigins 换成受控清单。
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Accept", "Authorization", "X-API-Key", "X-Request-ID", "X-Tenant-ID",
			"X-External-User-ID", "X-External-User-Token", "X-WeKnora-Desktop-Token",
			// Streamable HTTP MCP clients running in a browser send these on
			// the /mcp/:endpoint_id surface.
			"MCP-Protocol-Version", "Mcp-Session-Id", "Last-Event-ID",
		},
		ExposeHeaders:    []string{"Content-Length", "Access-Control-Allow-Origin", "Mcp-Session-Id"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 基础中间件（不需要认证）
	r.Use(middleware.RequestID())
	r.Use(middleware.Language())
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.ErrorHandler())

	// 健康检查（不需要认证）
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 前端静态文件（仅 Lite 版本内嵌前端）
	if handler.Edition == "lite" {
		serveFrontendStatic(r)
	}

	// Workspace MCP server surface (/mcp/:endpoint_id): bearer-token auth per
	// endpoint, so it must precede the global Auth middleware.
	RegisterMCPServerRoutes(r, params.MCPServer, params.MCPEndpointService, params.TenantService)

	// Short-lived capability URLs for IM and other clients that cannot attach
	// WeKnora authentication headers.
	serveResourceGrants(r, params.ResourceCatalog, params.TenantService, params.FileService, params.StorageBackendResolver)

	// 认证中间件
	r.Use(middleware.Auth(params.TenantService, params.UserService, params.TenantMemberService, params.TenantAPIKeyService))

	// 文件服务：统一代理本地/MinIO/COS/TOS存储后端（需要认证）
	serveFilesWithResources(r, params.FileService, params.StorageBackendResolver, params.ResourceCatalog)

	// Presigned file access: no auth required, signature-verified.
	servePresignedFiles(r, params.TenantService, params.StorageBackendResolver)

	// Diagnostic preview of presigned URLs (Admin only, behind auth middleware).
	servePresignedPreview(r, params.Config, params.StorageBackendResolver, params.ResourceCatalog)

	// 需要认证的API路由
	v1 := r.Group("/api/v1")
	{
		rbacGuards := newRBACGuards(
			params.KBHandler,
			params.CustomAgentHandler,
			params.KnowledgeHandler,
			params.ChunkHandler,
			params.WikiPageHandler,
			params.KBService,
			params.KnowledgeService,
			params.ChunkService,
		)

		// API-key gate: single authority for X-API-Key principals. Runs
		// first on every /api/v1 route (JWT sessions pass straight
		// through) and denies any route not explicitly declared via the
		// apiKeyGroup helpers. Must be attached BEFORE the Register* calls
		// so that sub-groups inherit it.
		v1.Use(rbacGuards.apiKeyAuthorizer.Middleware())

		RegisterAuthRoutes(v1, params.AuthHandler, rbacGuards)
		RegisterTenantRoutes(v1, params.TenantHandler, rbacGuards)
		RegisterKnowledgeBaseRoutes(v1, params.KBHandler, rbacGuards)
		RegisterImageAttrRoutes(v1, params.KBHandler, rbacGuards)
		// KB-scoped image proxy: serves images bound to a knowledge base
		// through the KB access guard, complementing the tenant-scoped
		// /files route.
		serveKBScopedFiles(
			v1,
			rbacGuards,
			params.TenantService,
			params.FileService,
			params.StorageBackendResolver,
			params.ResourceCatalog,
		)
		// Message-scoped image proxy: assistant replies belong to the
		// caller's session; referenced resources must be owned by the
		// caller's workspace. Authorization is derived from the persisted
		// message, never from a client-provided workspace ID.
		serveMessageScopedFiles(
			v1,
			rbacGuards,
			params.MessageService,
			params.TenantService,
			params.FileService,
			params.StorageBackendResolver,
			params.ResourceCatalog,
		)
		RegisterKnowledgeTagRoutes(v1, params.TagHandler, rbacGuards)
		RegisterKnowledgeRoutes(v1, params.KnowledgeHandler, rbacGuards)
		RegisterFAQRoutes(v1, params.FAQHandler, rbacGuards)
		RegisterChunkRoutes(v1, params.ChunkHandler, rbacGuards)
		RegisterSessionRoutes(v1, params.SessionHandler, params.MessageSuggestionHandler, rbacGuards)
		RegisterChatRoutes(v1, params.SessionHandler, rbacGuards)
		RegisterMessageRoutes(v1, params.MessageHandler, rbacGuards)
		RegisterModelRoutes(v1, params.ModelHandler, params.ModelCredentialsHandler, rbacGuards)
		RegisterInitializationRoutes(v1, params.InitializationHandler, rbacGuards)
		RegisterSystemRoutes(v1, params.SystemHandler, rbacGuards)
		RegisterSystemAdminRoutes(v1, params.SystemHandler, rbacGuards)
		RegisterVectorStoreRoutes(v1, params.VectorStoreHandler, rbacGuards)
		RegisterStorageBackendRoutes(v1, params.StorageBackendHandler, rbacGuards)
		RegisterCustomAgentRoutes(v1, params.CustomAgentHandler, rbacGuards)
		RegisterUserFavoriteRoutes(v1, params.UserFavoriteHandler, rbacGuards)
		RegisterMCPEndpointRoutes(v1, params.MCPEndpointHandler, rbacGuards)
		RegisterWeKnoraCloudRoutes(v1, params.WeKnoraCloudHandler, rbacGuards)
		RegisterWikiPageRoutes(v1, params.WikiPageHandler, rbacGuards)
		RegisterChunkerDebugRoutes(v1, rbacGuards)

		// Fail fast if any declared API-key policy points at a route
		// template that does not actually exist (typo / path drift). A
		// stale template would silently 403 every API key on that route,
		// so we panic at startup instead of shipping a dead policy.
		rbacGuards.assertAPIKeyPoliciesMatchRoutes(r)
	}

	return r
}

// trustedProxies returns the proxy CIDRs/IPs whose X-Forwarded-For headers
// gin should trust when resolving the client IP. Defaults to loopback and
// private ranges (covers the bundled nginx in a container network); override
// with WEKNORA_TRUSTED_PROXIES (comma-separated). An explicit empty value
// disables proxy trust entirely so ClientIP() returns the direct peer.
func trustedProxies() []string {
	raw, ok := os.LookupEnv("WEKNORA_TRUSTED_PROXIES")
	if !ok {
		return []string{
			"127.0.0.0/8",
			"::1/128",
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
			"fc00::/7",
		}
	}
	proxies := make([]string, 0)
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			proxies = append(proxies, p)
		}
	}
	return proxies
}
