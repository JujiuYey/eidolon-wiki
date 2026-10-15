// Package container implements dependency injection container setup
// Provides centralized configuration for services, repositories, and handlers
// This package is responsible for wiring up all dependencies and ensuring proper lifecycle management
package container

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	modelruntime "github.com/Tencent/WeKnora/internal/models/runtime"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"github.com/panjf2000/ants/v2"
	"github.com/qdrant/go-client/qdrant"
	"github.com/redis/go-redis/v9"
	"go.uber.org/dig"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/application/repository"
	neo4jRepo "github.com/Tencent/WeKnora/internal/application/repository/retriever/neo4j"
	postgresRepo "github.com/Tencent/WeKnora/internal/application/repository/retriever/postgres"
	qdrantRepo "github.com/Tencent/WeKnora/internal/application/repository/retriever/qdrant"
	"github.com/Tencent/WeKnora/internal/application/service"
	chatpipeline "github.com/Tencent/WeKnora/internal/application/service/chat_pipeline"
	"github.com/Tencent/WeKnora/internal/application/service/file"
	"github.com/Tencent/WeKnora/internal/application/service/retriever"
	"github.com/Tencent/WeKnora/internal/common"
	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/database"
	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/Tencent/WeKnora/internal/handler/session"
	"github.com/Tencent/WeKnora/internal/infrastructure/docparser"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/mcpserver"
	"github.com/Tencent/WeKnora/internal/models/api"
	"github.com/Tencent/WeKnora/internal/models/embedding"
	"github.com/Tencent/WeKnora/internal/models/limiter" // register built-in vendors
	"github.com/Tencent/WeKnora/internal/models/utils/ollama"
	"github.com/Tencent/WeKnora/internal/router"
	"github.com/Tencent/WeKnora/internal/storageallowlist"
	"github.com/Tencent/WeKnora/internal/stream"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/WeKnora/internal/utils"
)

// BuildContainer constructs the dependency injection container
// Registers all components, services, repositories and handlers needed by the application
// Creates a fully configured application container with proper dependency resolution
// Parameters:
//   - container: Base dig container to add dependencies to
//
// Returns:
//   - Configured container with all application dependencies registered
func BuildContainer(container *dig.Container) *dig.Container {
	// Deployment-level model catalog overlay (config/models.json, optional).
	// Register providers explicitly, then validate and publish one catalog generation.
	if err := modelruntime.Initialize(config.ConfigDir()); err != nil {
		logger.Warnf(context.Background(), "Load models catalog overlay failed: %v", err)
	}
	ctx := context.Background()
	logger.Debugf(ctx, "[Container] Starting container initialization...")

	// Register resource cleaner for proper cleanup of resources
	must(container.Provide(NewResourceCleaner, dig.As(new(interfaces.ResourceCleaner))))

	// Core infrastructure configuration
	logger.Debugf(ctx, "[Container] Registering core infrastructure...")
	must(container.Provide(config.LoadConfig))
	must(container.Provide(initDatabase))
	must(container.Provide(initFileService))
	must(container.Provide(initRedisClient))
	must(container.Provide(initAntsPool))

	// Register goroutine pool cleanup handler
	must(container.Invoke(registerPoolCleanup))

	// Initialize retrieval engine registry for search capabilities
	logger.Debugf(ctx, "[Container] Registering retrieval engine registry...")
	must(container.Provide(initRetrieveEngineRegistry))

	// External service clients
	logger.Debugf(ctx, "[Container] Registering external service clients...")
	must(container.Provide(initDocReaderClient))
	must(container.Provide(docparser.NewImageResolver))
	must(container.Provide(initOllamaService))
	must(container.Provide(initNeo4jClient))
	must(container.Provide(stream.NewStreamManager))

	// Data repositories layer
	logger.Debugf(ctx, "[Container] Registering repositories...")
	must(container.Provide(repository.NewTenantRepository))
	must(container.Provide(repository.NewTenantAPIKeyRepository))
	must(container.Provide(repository.NewTenantMemberRepository))
	must(container.Provide(repository.NewKnowledgeBaseRepository))
	must(container.Provide(repository.NewKnowledgeRepository))
	must(container.Provide(repository.NewKnowledgeSpanRepository))
	must(container.Provide(repository.NewChunkRepository))
	must(container.Provide(repository.NewKnowledgeTagRepository))
	must(container.Provide(repository.NewSessionRepository))
	must(container.Provide(repository.NewMessageRepository))
	must(container.Provide(repository.NewMessageSuggestionRepository))
	must(container.Provide(repository.NewModelRepository))
	must(container.Provide(repository.NewUserRepository))
	must(container.Provide(repository.NewAuthTokenRepository))
	must(container.Provide(repository.NewSystemSettingRepository))
	must(container.Provide(repository.NewModelCatalogRepository))
	must(container.Provide(neo4jRepo.NewNeo4jRepository))
	must(container.Provide(repository.NewCustomAgentRepository))
	must(container.Provide(repository.NewMCPEndpointRepository))
	must(container.Provide(repository.NewUserResourceFavoriteRepository))
	must(container.Provide(repository.NewWikiPageRepository))
	must(container.Provide(repository.NewTaskPendingOpsRepository))
	must(container.Provide(repository.NewTaskDeadLetterRepository))

	// Business service layer
	logger.Debugf(ctx, "[Container] Registering business services...")
	must(container.Provide(service.NewTenantService))
	must(container.Provide(service.NewTenantAPIKeyService))
	must(container.Provide(service.NewTenantMemberService))
	must(container.Provide(service.NewKnowledgeBaseService))
	must(container.Provide(service.NewKnowledgeService))
	must(container.Provide(service.NewSpanTracker))
	must(container.Provide(service.NewChunkService))
	must(container.Provide(service.NewKnowledgeTagService))
	must(container.Provide(embedding.NewBatchEmbedder))
	must(container.Provide(service.NewModelService))
	must(container.Provide(service.NewUserService))
	must(container.Provide(service.NewSystemSettingService))
	must(container.Provide(service.NewModelCatalogService))
	must(container.Provide(service.NewWeKnoraCloudService))

	// Extract services - register individual extracters with names
	must(container.Provide(service.NewChunkExtractService, dig.Name("chunkExtractor")))
	must(container.Provide(service.NewImageMultimodalService, dig.Name("imageMultimodal")))
	must(container.Provide(service.NewKnowledgePostProcessService, dig.Name("knowledgePostProcess")))
	must(container.Provide(service.NewKnowledgeAutoTagService, dig.Name("knowledgeAutoTag")))
	must(container.Provide(service.NewKnowledgeBaseProfileService))
	must(container.Provide(func(s *service.KnowledgeBaseProfileService) interfaces.KnowledgeBaseProfileService {
		return s
	}))
	must(container.Provide(func(s *service.KnowledgeBaseProfileService) interfaces.TaskHandler { return s },
		dig.Name("knowledgeBaseProfile")))

	must(container.Provide(service.NewMessageService))
	must(container.Provide(service.NewMessageSuggestionService))
	must(container.Provide(service.NewCustomAgentService))
	must(container.Provide(service.NewUserResourceFavoriteService))
	must(container.Provide(service.NewWikiPageService))
	must(container.Provide(service.NewWikiIngestService, dig.Name("wikiIngest")))
	must(container.Provide(service.NewWikiLintService))
	must(container.Provide(service.NewMCPEndpointService))
	must(container.Provide(mcpserver.NewServer))

	must(container.Provide(repository.NewVectorStoreRepository))
	must(container.Provide(repository.NewStorageBackendRepository))
	must(container.Provide(repository.NewResourceRepository))
	must(container.Provide(repository.NewTemporaryDocumentRepository))
	must(container.Provide(service.NewResourceCatalog))
	// TenantStoreOwnership adapter used by the retriever factory functions
	// to verify that a resolved VectorStore belongs to the caller's tenant.
	must(container.Provide(retriever.NewVectorStoreRepoOwnership))
	must(container.Provide(NewEngineFactory))
	// StoreRegistry: same instance as RetrieveEngineRegistry, exposed as StoreRegistry interface.
	// NewRetrieveEngineRegistry always returns *retriever.RetrieveEngineRegistry which implements both.
	must(container.Provide(func(r interfaces.RetrieveEngineRegistry) (interfaces.StoreRegistry, error) {
		sr, ok := r.(*retriever.RetrieveEngineRegistry)
		if !ok {
			return nil, fmt.Errorf("registry does not implement StoreRegistry")
		}
		return sr, nil
	}))
	must(container.Provide(service.NewVectorStoreService))
	must(container.Provide(service.NewStorageBackendServiceWithResources))
	must(container.Provide(func(s *service.StorageBackendService) interfaces.StorageBackendService { return s }))
	must(container.Provide(func(s *service.StorageBackendService) interfaces.StorageBackendResolver { return s }))

	// Agent service layer (requires event bus, web search service)
	// SessionService is passed as parameter to CreateAgentEngine method when creating AgentService
	logger.Debugf(ctx, "[Container] Registering event bus and agent service...")
	must(container.Provide(event.NewEventBus))
	must(container.Provide(service.NewAgentService))

	// Session service (depends on agent service)
	// SessionService is created after AgentService and passes itself to AgentService.CreateAgentEngine when needed
	logger.Debugf(ctx, "[Container] Registering session service...")
	must(container.Provide(service.NewSessionService))

	// Turn gating for chat sessions: process-local, since weknora-slim runs
	// single-node. Rewind takes the same gate to lock the session while it
	// truncates the conversation.
	must(container.Provide(service.NewSessionBusyGate))
	must(container.Provide(service.NewSessionRewindServiceFromRepos))

	logger.Debugf(ctx, "[Container] Registering task enqueuer...")
	redisAvailable := os.Getenv("REDIS_ADDR") != ""
	if redisAvailable {
		must(container.Provide(router.NewAsyncqClient, dig.As(new(interfaces.TaskEnqueuer))))
		// Dedicated pools guarantee capacity for each stage. The shared pool
		// additionally subscribes to core/enrichment queues to provide elastic
		// borrowing while post-process and maintenance remain hard-isolated.
		must(container.Provide(router.NewCoreAsynqServer, dig.Name("coreAsynqServer")))
		must(container.Provide(router.NewPostProcessAsynqServer, dig.Name("postProcessAsynqServer")))
		must(container.Provide(router.NewEnrichmentAsynqServer, dig.Name("enrichmentAsynqServer")))
		must(container.Provide(router.NewMaintenanceAsynqServer, dig.Name("maintenanceAsynqServer")))
		must(container.Provide(router.NewSharedAsynqServer, dig.Name("sharedAsynqServer")))
		must(container.Provide(router.NewWikiAsynqServer, dig.Name("wikiAsynqServer")))
		// Asynq inspector for cancel-by-knowledge-id (best-effort
		// dequeue of pending/scheduled/retry tasks + active-task cancel).
		must(container.Provide(router.NewAsynqInspector))
		must(container.Provide(router.NewAsynqTaskInspector))
		// Install the distributed per-model chat concurrency governor. Only
		// available with Redis (the shared semaphore backend); Lite mode is
		// single-process and low-volume, so it runs ungated.
		must(container.Invoke(registerModelConcurrencyLimiter))
	} else {
		syncExec := router.NewSyncTaskExecutor()
		must(container.Provide(func() interfaces.TaskEnqueuer { return syncExec }))
		must(container.Provide(func() *router.SyncTaskExecutor { return syncExec }))
		// Lite mode: no Redis means no asynq inspector. SyncTaskExecutor
		// dispatches inline goroutines that the checkpoint-based abort
		// already handles.
		must(container.Provide(router.NewNoopTaskInspector))
		// Even without Redis, background ingestion/enrichment can burst the
		// worker pool against one provider, so install an in-process governor.
		must(container.Invoke(registerLiteModelConcurrencyLimiter))
	}
	must(container.Provide(service.NewTemporaryDocumentService))
	must(container.Invoke(startTemporaryDocumentCleanup))

	// Chat pipeline components for processing chat requests
	logger.Debugf(ctx, "[Container] Registering chat pipeline plugins...")

	must(container.Provide(service.NewHousekeepingService))
	must(container.Invoke(startHousekeepingService))
	logger.Debugf(ctx, "[Container] Knowledge housekeeping runner registered")
	must(container.Provide(chatpipeline.NewEventManager))
	must(container.Invoke(chatpipeline.NewPluginSearch))
	must(container.Invoke(chatpipeline.NewPluginRerank))
	must(container.Invoke(chatpipeline.NewPluginMerge))
	must(container.Invoke(chatpipeline.NewPluginIntoChatMessage))
	must(container.Invoke(chatpipeline.NewPluginChatCompletion))
	must(container.Invoke(chatpipeline.NewPluginChatCompletionStream))
	must(container.Invoke(chatpipeline.NewPluginFilterTopK))
	must(container.Invoke(chatpipeline.NewPluginQueryUnderstand))
	must(container.Invoke(chatpipeline.NewPluginLoadHistory))
	must(container.Invoke(chatpipeline.NewPluginExtractEntity))
	must(container.Invoke(chatpipeline.NewPluginSearchEntity))
	must(container.Invoke(chatpipeline.NewPluginSearchParallel))
	must(container.Invoke(chatpipeline.NewPluginWikiBoost))
	logger.Debugf(ctx, "[Container] Chat pipeline plugins registered")

	// HTTP handlers layer
	logger.Debugf(ctx, "[Container] Registering HTTP handlers...")
	must(container.Provide(handler.NewTenantHandler))
	must(container.Provide(handler.NewKnowledgeBaseHandler))
	must(container.Provide(handler.NewKnowledgeHandler))
	must(container.Provide(handler.NewChunkHandler))
	must(container.Provide(handler.NewFAQHandler))
	must(container.Provide(handler.NewTagHandler))
	must(container.Provide(session.NewHandler))
	must(container.Provide(handler.NewMessageHandler))
	must(container.Provide(handler.NewMessageSuggestionHandler))
	must(container.Provide(handler.NewModelHandler))
	must(container.Provide(handler.NewInitializationHandler))
	must(container.Provide(handler.NewAuthHandler))
	must(container.Provide(handler.NewSystemHandler))
	must(container.Provide(handler.NewModelCredentialsHandler))
	must(container.Provide(handler.NewVectorStoreHandler))
	must(container.Provide(handler.NewStorageBackendHandler))
	must(container.Provide(handler.NewCustomAgentHandler))
	must(container.Provide(handler.NewUserResourceFavoriteHandler))

	// Wiki page handler
	must(container.Provide(handler.NewWikiPageHandler))
	must(container.Provide(handler.NewMCPEndpointHandler))
	must(container.Provide(handler.NewWeKnoraCloudHandler))
	logger.Debugf(ctx, "[Container] HTTP handlers registered")

	// Wire the chat package's local image resolver so multimodal chat can read
	// local:// images that live under a tenant's configured storage PathPrefix
	// (which is not encoded in the local:// URL).
	must(container.Invoke(registerChatLocalImageResolver))

	// Router configuration
	logger.Debugf(ctx, "[Container] Registering router and starting task server...")
	must(container.Provide(router.NewRouter))
	if redisAvailable {
		must(container.Invoke(router.RunAsynqServer))
	} else {
		must(container.Invoke(router.RegisterSyncHandlers))
	}
	// Wiki operation rows are durable, while their wake-up triggers may be
	// lost across a process restart (always in Lite mode, and in Redis mode if
	// persistence succeeded immediately before trigger enqueue failed). Re-arm
	// them only after the matching handlers are ready.
	must(container.Invoke(recoverPendingWikiTasks))

	logger.Infof(ctx, "[Container] Container initialization completed successfully")
	return container
}

// registerChatLocalImageResolver wires the chat package's LocalImageResolver
// hook. Stored local:// URLs are relative to the resolved storage base dir and
// do NOT encode the owning tenant's configured PathPrefix, so resolving them to
// disk bytes requires rebuilding the FileService from that tenant's storage
// config. The owning tenant is parsed from the URL's first path segment, which
// correctly handles cross-tenant shared resources (e.g. shared KB images).
func registerChatLocalImageResolver(
	tenantRepo interfaces.TenantRepository,
	storageResolver interfaces.StorageBackendResolver,
	resourceCatalog interfaces.ResourceCatalog,
) {
	api.LocalImageResolver = func(storageURL string) ([]byte, bool) {
		// The object storage clients bound connection setup but leave the
		// transfer to this context, so give it a deadline: a chat turn must
		// not hang on one image whose download stalls.
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		physicalPath, resource, err := resourceCatalog.ResolvePath(ctx, storageURL)
		if err != nil {
			return nil, false
		}
		tenantID := secutils.ParseTenantIDFromStoragePath(physicalPath)
		if resource != nil {
			tenantID = resource.TenantID
		}
		if tenantID == 0 {
			return nil, false
		}
		tenant, err := tenantRepo.GetTenantByID(ctx, tenantID)
		if err != nil || tenant == nil {
			return nil, false
		}
		baseDir := strings.TrimSpace(os.Getenv("LOCAL_STORAGE_BASE_DIR"))
		backendID, inner, scoped := types.ParseStorageBackendPath(physicalPath)
		if resource != nil && resource.StorageBackendID != "" {
			backendID = resource.StorageBackendID
		}
		providerPath := physicalPath
		if scoped {
			providerPath = inner
		}
		provider := types.ParseProviderScheme(providerPath)
		if provider == "" {
			provider = "local"
		}
		fileSvc, _, err := storageResolver.ResolveFileService(ctx, tenant, backendID, provider, baseDir)
		if err != nil {
			return nil, false
		}
		rc, err := fileSvc.GetFile(ctx, physicalPath)
		if err != nil {
			return nil, false
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			return nil, false
		}
		return data, true
	}
}

// must is a helper function for error handling
// Panics if the error is not nil, useful for configuration steps that must succeed
// Parameters:
//   - err: Error to check
func must(err error) {
	if err != nil {
		panic(err)
	}
}

// defaultModelMaxConcurrency is the per-model cap on concurrent background
// (ingestion/enrichment) chat calls when WEKNORA_MODEL_MAX_CONCURRENCY /
// model.max_concurrency is unset. summary / question / graph enrichment all
// share the same model, so this bounds their combined pressure on one provider
// across every replica. Interactive chat is never gated.
const defaultModelMaxConcurrency = 32

// resolveModelMaxConcurrency reads the per-model background concurrency limit
// from system settings / env, defaulting to defaultModelMaxConcurrency when
// unset. A configured value of 0 (or negative) is honoured and disables the
// governor — that is the supported way to turn throttling off via config/env.
func resolveModelMaxConcurrency(ss interfaces.SystemSettingService) int {
	if ss == nil {
		return defaultModelMaxConcurrency
	}
	return int(ss.GetInt(context.Background(), "model.max_concurrency",
		"WEKNORA_MODEL_MAX_CONCURRENCY", int64(defaultModelMaxConcurrency)))
}

// registerModelConcurrencyLimiter builds the Redis-backed per-model background
// concurrency governor (chat + vlm) and installs it. Only available with Redis
// (the shared semaphore backend); Lite mode uses registerLiteModelConcurrencyLimiter.
func registerModelConcurrencyLimiter(rdb *redis.Client, ss interfaces.SystemSettingService) {
	limit := resolveModelMaxConcurrency(ss)
	limiter.SetGovernor(limiter.NewRedisLimiter(rdb), limit)
	if limit <= 0 {
		logger.Infof(context.Background(),
			"[ModelLimiter] background concurrency governor DISABLED (model.max_concurrency<=0)")
		return
	}
	logger.Infof(context.Background(),
		"[ModelLimiter] background model concurrency governed per-model, limit=%d (distributed via redis)", limit)
}

// registerLiteModelConcurrencyLimiter installs an in-process per-model governor
// for Lite mode (no Redis). Lite runs a single process, so an in-process
// semaphore is sufficient to keep a background ingestion storm from bursting
// the whole worker pool against one provider.
func registerLiteModelConcurrencyLimiter(ss interfaces.SystemSettingService) {
	limit := resolveModelMaxConcurrency(ss)
	limiter.SetGovernor(limiter.NewLocalLimiter(), limit)
	if limit <= 0 {
		logger.Infof(context.Background(),
			"[ModelLimiter] background concurrency governor DISABLED (model.max_concurrency<=0)")
		return
	}
	logger.Infof(context.Background(),
		"[ModelLimiter] background model concurrency governed per-model, limit=%d (in-process, lite mode)", limit)
}

func initRedisClient() (*redis.Client, error) {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		logger.Infof(context.Background(), "[Redis] No REDIS_ADDR configured, Redis disabled (Lite mode)")
		return nil, nil
	}
	db, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		db = 0
	}

	client := redis.NewClient(&redis.Options{
		Addr:      redisAddr,
		Username:  os.Getenv("REDIS_USERNAME"),
		Password:  os.Getenv("REDIS_PASSWORD"),
		DB:        db,
		TLSConfig: common.RedisTLSConfig(),
	})

	_, err = client.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("连接Redis失败: %w", err)
	}

	return client, nil
}

// initDatabase initializes database connection
// Creates and configures database connection based on environment configuration
// Supports multiple database backends (PostgreSQL)
// Parameters:
//   - cfg: Application configuration
//
// Returns:
//   - Configured database connection
//   - Error if connection fails
func initDatabase(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	var migrateDSN string
	switch os.Getenv("DB_DRIVER") {
	case "postgres":
		// DSN for GORM (key-value format)
		gormDSN := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
			os.Getenv("DB_HOST"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_NAME"),
			"disable",
		)
		dialector = postgres.Open(gormDSN)

		// DSN for golang-migrate (URL format)
		// URL-encode password to handle special characters like !@#
		dbPassword := os.Getenv("DB_PASSWORD")
		encodedPassword := url.QueryEscape(dbPassword)

		// Check if postgres is in RETRIEVE_DRIVER to determine skip_embedding
		retrieveDriver := strings.Split(os.Getenv("RETRIEVE_DRIVER"), ",")
		skipEmbedding := "true"
		if slices.Contains(retrieveDriver, "postgres") {
			skipEmbedding = "false"
		}
		logger.Infof(context.Background(), "Skip embedding: %s", skipEmbedding)

		migrateDSN = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=disable&options=-c%%20app.skip_embedding=%s",
			os.Getenv("DB_USER"),
			encodedPassword, // Use encoded password
			os.Getenv("DB_HOST"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_NAME"),
			skipEmbedding,
		)

		// Debug log (don't log password)
		logger.Infof(context.Background(), "DB Config: user=%s host=%s port=%s dbname=%s",
			os.Getenv("DB_USER"),
			os.Getenv("DB_HOST"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_NAME"),
		)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", os.Getenv("DB_DRIVER"))
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, err
	}

	// Sanity check: dialect-specific code in services (notably the
	// vector_stores delete guard) compares Dialector.Name() to "postgres"
	// string literals. A future driver swap that produces a different name
	// (e.g., a wrapper dialect for managed PG) would silently fall back to
	// a non-postgres path, dropping the row-level X-lock. Catching the
	// mismatch at startup is loud and inexpensive.
	if name := db.Dialector.Name(); name != "postgres" {
		return nil, fmt.Errorf(
			"unsupported gorm dialector %q; expected postgres "+
				"(see vectorStoreService.isPostgres for impact)", name)
	}

	// Run database migrations automatically (optional, can be disabled via env var)
	// To disable auto-migration, set AUTO_MIGRATE=false
	// To enable auto-recovery from dirty state, set AUTO_RECOVER_DIRTY=true
	if os.Getenv("AUTO_MIGRATE") != "false" {
		logger.Infof(context.Background(), "Running database migrations...")

		autoRecover := os.Getenv("AUTO_RECOVER_DIRTY") != "false"
		migrationOpts := database.MigrationOptions{
			AutoRecoverDirty: autoRecover,
		}

		// Run base migrations (all versioned migrations including embeddings)
		// The embeddings migration will be conditionally executed based on skip_embedding parameter in DSN
		if err := database.RunMigrationsWithOptions(migrateDSN, migrationOpts); err != nil {
			// Log warning but don't fail startup - migrations might be handled externally
			logger.Warnf(context.Background(), "Database migration failed: %v", err)
			logger.Warnf(
				context.Background(),
				"Continuing with application startup. Please run migrations manually if needed.",
			)
		}

		// Post-migration: resolve __pending_env__ storage provider markers for historical KBs.
		// The SQL migration marks KBs that have documents but no provider with "__pending_env__";
		// we replace that with the actual STORAGE_TYPE from the environment.
		resolveStorageProviderPending(db)
		migrateLegacyStorageBackends(db)

		// Post-migration: declarative built-in models from config/builtin_models.yaml (optional).
		// The loader validates each row's catalog parameters through this hook;
		// the wiring lives here because internal/types cannot import the catalog.
		types.ValidateModelParameters = modelruntime.ValidateRow
		if err := types.LoadBuiltinModelsConfig(context.Background(), db, config.ConfigDir()); err != nil {
			logger.Warnf(context.Background(), "Load builtin models config failed: %v", err)
		}
	} else {
		logger.Infof(context.Background(), "Auto-migration is disabled (AUTO_MIGRATE=false)")
	}

	// Get underlying SQL DB object
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Configure connection pool parameters
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Duration(10) * time.Minute)

	return db, nil
}

// resolveStorageProviderPending replaces the "__pending_env__" sentinel in
// knowledge_bases.storage_provider_config with the actual STORAGE_TYPE from the environment.
// This runs once after SQL migrations to bind historical KBs to their real storage provider.
func resolveStorageProviderPending(db *gorm.DB) {
	storageType := strings.TrimSpace(os.Getenv("STORAGE_TYPE"))
	if storageType == "" {
		storageType = "local"
	}
	storageType = strings.ToLower(storageType)

	result := db.Exec(
		`UPDATE knowledge_bases SET storage_provider_config = ? WHERE storage_provider_config IS NOT NULL AND storage_provider_config->>'provider' = '__pending_env__'`,
		fmt.Sprintf(`{"provider":"%s"}`, storageType),
	)
	if result.Error != nil {
		logger.Warnf(context.Background(), "Failed to resolve __pending_env__ storage providers: %v", result.Error)
	} else if result.RowsAffected > 0 {
		logger.Infof(context.Background(), "Resolved %d knowledge bases with __pending_env__ storage provider → %s", result.RowsAffected, storageType)
	}

	// Sync PostgreSQL sequences with actual MAX values to prevent duplicate key
	// errors. The old code assigned seq_id via SELECT MAX()+1 in application
	// code, which could push values past the DB sequence counter.
	syncSequences(db)

	// Reset any pending tasks left over from previous aborted runs (Lite App mode)
	resetPendingTasks(db)
}

// migrateLegacyStorageBackends backfills the storage_backends table from each
// workspace's legacy StorageEngineConfig (or environment defaults) and binds
// existing knowledge bases to the resulting backend.
//
// The table, columns and indexes are created by the SQL migrations
// (migrations/versioned/000068 for Postgres, migrations/sqlite/000000_init for
// SQLite); this step only handles data that cannot be expressed portably in
// SQL: environment snapshots, JSON→config mapping, AES-encrypted credentials,
// UUID generation and the per-startup refresh of env-backed aliases.
// The migration is idempotent: one legacy_alias row per tenant/provider.
func migrateLegacyStorageBackends(db *gorm.DB) {
	var tenants []*types.Tenant
	if err := db.Find(&tenants).Error; err != nil {
		logger.Warnf(context.Background(), "Failed to load workspaces for storage backend migration: %v", err)
		return
	}
	if len(tenants) == 0 {
		return
	}

	// Load every alias in a single query. Probing each tenant/provider pair with
	// First() makes GORM log "record not found" for every miss, which floods the
	// startup log with workspaces × providers lines on fresh installs.
	var aliases []*types.StorageBackend
	if err := db.Where("legacy_alias = ?", true).Find(&aliases).Error; err != nil {
		logger.Warnf(context.Background(), "Failed to load legacy storage aliases: %v", err)
		return
	}
	existingAliases := make(map[uint64]map[string]*types.StorageBackend, len(aliases))
	for _, alias := range aliases {
		byProvider := existingAliases[alias.TenantID]
		if byProvider == nil {
			byProvider = make(map[string]*types.StorageBackend)
			existingAliases[alias.TenantID] = byProvider
		}
		byProvider[alias.Provider] = alias
	}

	for _, tenant := range tenants {
		legacy := tenant.StorageEngineConfig
		defaultProvider := ""
		if legacy != nil {
			defaultProvider = strings.ToLower(strings.TrimSpace(legacy.DefaultProvider))
		}
		if defaultProvider == "" {
			defaultProvider = strings.ToLower(strings.TrimSpace(os.Getenv("STORAGE_TYPE")))
		}
		if defaultProvider == "" {
			defaultProvider = "local"
		}

		backendIDs := make(map[string]string)
		for _, provider := range storageallowlist.Supported() {
			if existing := existingAliases[tenant.ID][provider]; existing != nil {
				// Environment-backed aliases are snapshots, not user-owned config.
				// Refresh them at every startup so credential rotation does not
				// leave the persisted resolver on stale values. If the workspace
				// later gains an explicit legacy config, promote the alias to user
				// source and stop automatic refreshes.
				if existing.Source == types.StorageBackendSourceEnv {
					desired := types.StorageBackendFromLegacy(tenant.ID, provider, legacy)
					if desired == nil && provider == defaultProvider {
						desired = types.StorageBackendFromEnvironment(tenant.ID)
					}
					if desired != nil && desired.Provider == provider {
						_ = db.Model(&types.StorageBackend{}).Where("id = ?", existing.ID).Updates(map[string]interface{}{
							"name": desired.Name, "config": desired.Config, "source": desired.Source, "status": desired.Status, "updated_at": time.Now(),
						}).Error
					}
				}
				backendIDs[provider] = existing.ID
				continue
			}
			backend := types.StorageBackendFromLegacy(tenant.ID, provider, legacy)
			if backend == nil && provider == defaultProvider {
				backend = types.StorageBackendFromEnvironment(tenant.ID)
			}
			if backend == nil {
				continue
			}
			if err := db.Create(backend).Error; err != nil {
				logger.Warnf(context.Background(), "Failed to migrate %s storage for workspace %d: %v", provider, tenant.ID, err)
				continue
			}
			backendIDs[provider] = backend.ID
		}
		if tenant.DefaultStorageBackendID == nil {
			if id := backendIDs[defaultProvider]; id != "" {
				if err := db.Model(&types.Tenant{}).Where("id = ?", tenant.ID).Update("default_storage_backend_id", id).Error; err != nil {
					logger.Warnf(context.Background(), "Failed to set default storage backend for workspace %d: %v", tenant.ID, err)
				}
			}
		}

		var kbs []*types.KnowledgeBase
		if err := db.Where("tenant_id = ? AND storage_backend_id IS NULL", tenant.ID).Find(&kbs).Error; err != nil {
			continue
		}
		for _, kb := range kbs {
			provider := kb.GetStorageProvider()
			if provider == "" {
				provider = defaultProvider
			}
			if id := backendIDs[provider]; id != "" {
				_ = db.Model(&types.KnowledgeBase{}).Where("id = ? AND storage_backend_id IS NULL", kb.ID).Update("storage_backend_id", id).Error
			}
		}
	}
}

// syncSequences ensures PostgreSQL sequences for auto-increment columns (seq_id)
// are at least as high as the current MAX value in each table. This is needed
// because older code assigned seq_id via application-level MAX()+1, which could
// advance values past the DB sequence counter and cause duplicate key errors.
func syncSequences(db *gorm.DB) {
	if db.Dialector.Name() != "postgres" {
		return
	}
	pairs := [][2]string{
		{"chunks", "chunks_seq_id_seq"},
		{"knowledge_tags", "knowledge_tags_seq_id_seq"},
	}
	for _, p := range pairs {
		table, seq := p[0], p[1]
		sql := fmt.Sprintf(
			`SELECT setval('%s', GREATEST(nextval('%s'), (SELECT COALESCE(MAX(seq_id), 0) FROM %s)))`,
			seq, seq, table,
		)
		if err := db.Exec(sql).Error; err != nil {
			logger.Warnf(context.Background(), "Failed to sync sequence %s: %v", seq, err)
		} else {
			logger.Infof(context.Background(), "Synced sequence %s with table %s", seq, table)
		}
	}
}

// initFileService initializes file storage service
// Creates the appropriate file storage service based on configuration
// Supports multiple storage backends (MinIO, COS, local filesystem)
// Parameters:
//   - cfg: Application configuration
//
// Returns:
//   - Configured file service implementation
//   - Error if initialization fails
func initFileService(cfg *config.Config, catalog interfaces.ResourceCatalog) (interfaces.FileService, error) {
	inner, err := initRawFileService(cfg)
	if err != nil {
		return nil, err
	}
	return file.NewResourceCatalogFileService(inner, catalog), nil
}

func initRawFileService(_ *config.Config) (interfaces.FileService, error) {
	storageType := strings.TrimSpace(os.Getenv("STORAGE_TYPE"))
	if storageType == "" {
		storageType = "local"
	}
	switch storageType {
	case "minio":
		if os.Getenv("MINIO_ENDPOINT") == "" ||
			os.Getenv("MINIO_ACCESS_KEY_ID") == "" ||
			os.Getenv("MINIO_SECRET_ACCESS_KEY") == "" ||
			os.Getenv("MINIO_BUCKET_NAME") == "" {
			return nil, fmt.Errorf("missing MinIO configuration")
		}
		return file.NewMinioFileService(
			os.Getenv("MINIO_ENDPOINT"),
			os.Getenv("MINIO_ACCESS_KEY_ID"),
			os.Getenv("MINIO_SECRET_ACCESS_KEY"),
			os.Getenv("MINIO_BUCKET_NAME"),
			strings.EqualFold(os.Getenv("MINIO_USE_SSL"), "true"),
		)
	case "local":
		baseDir := os.Getenv("LOCAL_STORAGE_BASE_DIR")
		if baseDir == "" {
			baseDir = "/data/files"
		}
		externalURL := strings.TrimSpace(os.Getenv("APP_EXTERNAL_URL"))
		return file.NewLocalFileService(baseDir, externalURL), nil
	case "dummy":
		return file.NewDummyFileService(), nil
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}
}

// newEnvQdrantClient builds the env-configured qdrant client, refusing a host
// whitelist-only mode does not admit before the client exists.
//
// This client is the one outbound path the dial-time guard cannot
// cover: gRPC resolves its own target through its dns resolver before calling
// any dialer, so by the time a dialer sees the address the name is gone and
// the query has already happened. Judging the configured name here is what
// keeps a non-whitelisted vector store host from ever being resolved (#3378).
func newEnvQdrantClient(host string, port int, apiKey string, useTLS bool) (*qdrant.Client, error) {
	if err := secutils.CheckSSRFWhitelistOnly(host); err != nil {
		return nil, err
	}
	return qdrant.NewClient(&qdrant.Config{Host: host, Port: port, APIKey: apiKey, UseTLS: useTLS})
}

// initRetrieveEngineRegistry initializes the retrieval engine registry from
// RETRIEVE_DRIVER. This build ships two engines only: PostgreSQL (pgvector
// over the business DB) and Qdrant.
//
// Parameters:
//   - db: Database connection
//   - cfg: Application configuration
//
// Returns:
//   - Configured retrieval engine registry
//   - Error if initialization fails
func initRetrieveEngineRegistry(
	db *gorm.DB, cfg *config.Config,
	storeRepo interfaces.VectorStoreRepository, engineFactory interfaces.EngineFactory,
) (interfaces.RetrieveEngineRegistry, error) {
	// storeRepo and engineFactory let the registry rebuild a store engine that
	// is absent from this process, which happens when startup skipped it after
	// a construction failure or when another instance registered it.
	registry := retriever.NewRetrieveEngineRegistry(storeRepo, engineFactory)
	retrieveDriver := strings.Split(os.Getenv("RETRIEVE_DRIVER"), ",")
	log := logger.GetLogger(context.Background())
	if slices.Contains(retrieveDriver, "postgres") {
		postgresRepo := postgresRepo.NewPostgresRetrieveEngineRepository(db)
		if err := registry.Register(
			retriever.NewKVHybridRetrieveEngine(postgresRepo, types.PostgresRetrieverEngineType),
		); err != nil {
			log.Errorf("Register postgres retrieve engine failed: %v", err)
		} else {
			log.Infof("Register postgres retrieve engine success")
		}
	}
	if slices.Contains(retrieveDriver, "qdrant") {
		qdrantHost := os.Getenv("QDRANT_HOST")
		if qdrantHost == "" {
			qdrantHost = "localhost"
		}

		qdrantPort := 6334 // Default port
		if portStr := os.Getenv("QDRANT_PORT"); portStr != "" {
			if port, err := strconv.Atoi(portStr); err == nil {
				qdrantPort = port
			}
		}

		// API key for authentication (optional)
		qdrantAPIKey := os.Getenv("QDRANT_API_KEY")

		// TLS configuration (optional, defaults to false)
		// Enable TLS unless explicitly set to "false" or "0" (case insensitive)
		qdrantUseTLS := false
		if useTLSStr := os.Getenv("QDRANT_USE_TLS"); useTLSStr != "" {
			useTLSLower := strings.ToLower(strings.TrimSpace(useTLSStr))
			qdrantUseTLS = useTLSLower != "false" && useTLSLower != "0"
		}

		log.Infof("Connecting to Qdrant at %s:%d (TLS: %v)", qdrantHost, qdrantPort, qdrantUseTLS)

		client, err := newEnvQdrantClient(qdrantHost, qdrantPort, qdrantAPIKey, qdrantUseTLS)
		if err != nil {
			log.Errorf("Create qdrant client failed: %v", err)
		} else {
			qdrantRepository := qdrantRepo.NewQdrantRetrieveEngineRepository(client, nil)
			if err := registry.Register(
				retriever.NewKVHybridRetrieveEngine(
					qdrantRepository, types.QdrantRetrieverEngineType,
				),
			); err != nil {
				log.Errorf("Register qdrant retrieve engine failed: %v", err)
			} else {
				log.Infof("Register qdrant retrieve engine success")
			}
		}
	}
	// ─── DB store registration (byStoreID) ───
	if storeReg, ok := registry.(*retriever.RetrieveEngineRegistry); ok {
		loadDBStoresIntoRegistry(storeReg, db, cfg)
	}

	return registry, nil
}

// loadDBStoresIntoRegistry loads VectorStore records from DB and registers them
// in the registry's byStoreID map. Failures are logged and skipped (non-fatal).
func loadDBStoresIntoRegistry(
	storeRegistry interfaces.StoreRegistry, db *gorm.DB, cfg *config.Config,
) {
	ctx := context.Background()
	log := logger.GetLogger(ctx)

	var stores []types.VectorStore
	// GORM soft delete automatically adds "deleted_at IS NULL" condition
	if err := db.Find(&stores).Error; err != nil {
		log.Warnf("Failed to load vector stores from DB: %v", err)
		return
	}

	if len(stores) == 0 {
		return
	}

	log.Infof("Loading %d vector store(s) from database", len(stores))
	for _, store := range stores {
		svc, err := createEngineServiceFromStore(ctx, store, db, cfg)
		if err != nil {
			log.Errorf("Failed to create engine for store %s (%s): %v", store.ID, store.Name, err)
			continue
		}
		storeRegistry.RegisterWithStoreID(store.ID, svc)
		log.Infof("Registered DB vector store: id=%s, name=%s, engine=%s", store.ID, store.Name, store.EngineType)
	}
}

// initAntsPool initializes the goroutine pool
// Creates a managed goroutine pool for concurrent task execution
// Parameters:
//   - cfg: Application configuration
//
// Returns:
//   - Configured goroutine pool
//   - Error if initialization fails
func initAntsPool(cfg *config.Config) (*ants.Pool, error) {
	// Default to 5 if not specified in config
	poolSize := os.Getenv("CONCURRENCY_POOL_SIZE")
	if poolSize == "" {
		poolSize = "5"
	}
	poolSizeInt, err := strconv.Atoi(poolSize)
	if err != nil {
		return nil, err
	}
	// Set up the pool with pre-allocation for better performance
	return ants.NewPool(poolSizeInt, ants.WithPreAlloc(true))
}

// registerPoolCleanup registers the goroutine pool for cleanup
// Ensures proper cleanup of the goroutine pool when application shuts down
// Parameters:
//   - pool: Goroutine pool
//   - cleaner: Resource cleaner
func registerPoolCleanup(pool *ants.Pool, cleaner interfaces.ResourceCleaner) {
	cleaner.RegisterWithName("AntsPool", func() error {
		pool.Release()
		return nil
	})
}

// initDocReaderClient initializes the DocumentReader client (lightweight API).
func initDocReaderClient(cfg *config.Config) (interfaces.DocumentReader, error) {
	addr := strings.TrimSpace(os.Getenv("DOCREADER_ADDR"))
	transport := strings.TrimSpace(os.Getenv("DOCREADER_TRANSPORT"))
	if transport == "" {
		transport = "grpc"
	}
	if addr == "" {
		logger.Infof(context.Background(), "[DocConverter] No DOCREADER_ADDR configured, starting disconnected")
	}
	transport = strings.ToLower(transport)
	switch transport {
	case "http", "https":
		if addr != "" && !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
			addr = "http://" + addr
		}
		return docparser.NewHTTPDocumentReader(addr)
	default:
		return docparser.NewGRPCDocumentReader(addr)
	}
}

// initOllamaService initializes the Ollama service client
// Creates a client for interacting with Ollama API for model inference
// Parameters:
//   - None
//
// Returns:
//   - Configured Ollama service client
//   - Error if initialization fails
func initOllamaService() (*ollama.OllamaService, error) {
	// Get Ollama service from existing factory function
	return ollama.GetOllamaService()
}

func initNeo4jClient() (neo4j.Driver, error) {
	ctx := context.Background()
	if strings.ToLower(os.Getenv("NEO4J_ENABLE")) != "true" {
		logger.Debugf(ctx, "NOT SUPPORT RETRIEVE GRAPH")
		return nil, nil
	}
	uri := os.Getenv("NEO4J_URI")
	username := os.Getenv("NEO4J_USERNAME")
	password := os.Getenv("NEO4J_PASSWORD")

	// Retry configuration
	maxRetries := 30                 // Max retry attempts
	retryInterval := 2 * time.Second // Wait between retries

	var driver neo4j.Driver
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		driver, err = neo4j.NewDriver(uri, neo4j.BasicAuth(username, password, ""))
		if err != nil {
			logger.Warnf(ctx, "Failed to create Neo4j driver (attempt %d/%d): %v", attempt, maxRetries, err)
			time.Sleep(retryInterval)
			continue
		}

		err = driver.VerifyAuthentication(ctx, nil)
		if err == nil {
			if attempt > 1 {
				logger.Infof(ctx, "Successfully connected to Neo4j after %d attempts", attempt)
			}
			return driver, nil
		}

		logger.Warnf(ctx, "Failed to verify Neo4j authentication (attempt %d/%d): %v", attempt, maxRetries, err)
		driver.Close(ctx)
		time.Sleep(retryInterval)
	}

	return nil, fmt.Errorf("failed to connect to Neo4j after %d attempts: %w", maxRetries, err)
}

// startHousekeepingService starts the knowledge housekeeping cron and registers
// cleanup. This is the safety net that recovers any knowledge stuck in
// "processing" past a configurable threshold (see HousekeepingService for
// rationale). Best-effort: a startup error is logged but does NOT abort the
// container — the rest of the system stays usable.
func startHousekeepingService(svc *service.HousekeepingService, cleaner interfaces.ResourceCleaner) {
	if svc == nil {
		return
	}
	if err := svc.Start(context.Background()); err != nil {
		logger.Warnf(context.Background(), "[Container] housekeeping start failed: %v", err)
	}
	cleaner.RegisterWithName("KnowledgeHousekeeping", func() error {
		svc.StopWithin(cleanupStepTimeout)
		return nil
	})
}

// startTemporaryDocumentCleanup removes expired session attachments and their
// extracted images. The durable expiry timestamp is the source of truth; the
// ticker only controls how quickly storage is reclaimed.
func startTemporaryDocumentCleanup(svc interfaces.TemporaryDocumentService, cleaner interfaces.ResourceCleaner) {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := svc.CleanupExpired(context.Background()); err != nil {
					logger.Warnf(context.Background(), "[TemporaryDocument] cleanup failed: %v", err)
				}
			case <-stop:
				return
			}
		}
	}()
	cleaner.RegisterWithName("TemporaryDocumentCleanup", func() error {
		close(stop)
		return nil
	})
}
