package container

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"gorm.io/gorm"

	postgresRepo "github.com/Tencent/WeKnora/internal/application/repository/retriever/postgres"
	qdrantRepo "github.com/Tencent/WeKnora/internal/application/repository/retriever/qdrant"
	"github.com/Tencent/WeKnora/internal/application/service/retriever"
	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/Tencent/WeKnora/internal/utils"
)

// NewEngineFactory returns an EngineFactory function closed over db and cfg.
// Registered in dig and injected into VectorStoreService for dynamic
// registry updates.
func NewEngineFactory(db *gorm.DB, cfg *config.Config) interfaces.EngineFactory {
	return func(ctx context.Context, store types.VectorStore) (interfaces.RetrieveEngineService, error) {
		return createEngineServiceFromStore(ctx, store, db, cfg)
	}
}

// createEngineServiceFromStore creates a RetrieveEngineService from a VectorStore's config.
// This is the DB store counterpart of the env-based initialization in initRetrieveEngineRegistry.
func createEngineServiceFromStore(
	ctx context.Context,
	store types.VectorStore,
	db *gorm.DB,
	cfg *config.Config,
) (interfaces.RetrieveEngineService, error) {
	if err := validateRuntimeVectorStoreAddresses(store); err != nil {
		return nil, err
	}
	switch store.EngineType {
	case types.PostgresRetrieverEngineType:
		return createPostgresEngine(store, db)
	case types.QdrantRetrieverEngineType:
		return createQdrantEngine(store)
	default:
		return nil, fmt.Errorf("unsupported engine type: %s", store.EngineType)
	}
}

// validateRuntimeVectorStoreAddresses is the final guard for persisted or
// imported vector-store rows. Create/test handlers validate the same fields at
// the input boundary, but runtime construction must not assume every stored row
// was written through those handlers.
func validateRuntimeVectorStoreAddresses(store types.VectorStore) error {
	cc := store.ConnectionConfig
	check := func(label, endpoint string) error {
		endpoint = strings.TrimSpace(endpoint)
		if endpoint == "" {
			return nil
		}
		if err := utils.ValidateURLForSSRF(endpoint); err != nil {
			return fmt.Errorf("%s failed SSRF validation: %w", label, err)
		}
		return nil
	}

	switch store.EngineType {
	case types.PostgresRetrieverEngineType:
		return nil
	case types.QdrantRetrieverEngineType:
		endpoint := cc.Host
		if endpoint != "" && cc.Port != 0 {
			endpoint = net.JoinHostPort(strings.Trim(cc.Host, "[]"), strconv.Itoa(cc.Port))
		}
		return check("qdrant address", endpoint)
	default:
		return fmt.Errorf("vector store engine %q has no SSRF address policy", store.EngineType)
	}
}

func createPostgresEngine(store types.VectorStore, db *gorm.DB) (interfaces.RetrieveEngineService, error) {
	if store.ConnectionConfig.UseDefaultConnection {
		repo := postgresRepo.NewPostgresRetrieveEngineRepository(db)
		return retriever.NewKVHybridRetrieveEngine(repo, types.PostgresRetrieverEngineType), nil
	}
	// Phase 1: only UseDefaultConnection is supported.
	// Custom connections require connection pool management and migration handling.
	return nil, fmt.Errorf("custom postgres connections not yet supported; use use_default_connection=true")
}

func createQdrantEngine(store types.VectorStore) (interfaces.RetrieveEngineService, error) {
	cc := store.ConnectionConfig
	port := cc.Port
	if port == 0 {
		port = 6334
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:        cc.Host,
		Port:        port,
		APIKey:      cc.APIKey,
		UseTLS:      cc.UseTLS,
		GrpcOptions: []grpc.DialOption{grpc.WithContextDialer(utils.SSRFSafeGRPCDialer)},
	})
	if err != nil {
		return nil, fmt.Errorf("create qdrant client: %w", err)
	}
	repo := qdrantRepo.NewQdrantRetrieveEngineRepository(client, &store.IndexConfig)
	return retriever.NewKVHybridRetrieveEngine(repo, types.QdrantRetrieverEngineType), nil
}
