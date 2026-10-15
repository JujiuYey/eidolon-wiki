package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newSharedAccessContext returns a plain caller context (tenant 1) for the
// service-level access-scope tests.
func newSharedAccessContext() context.Context {
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(1))
	ctx = context.WithValue(ctx, types.UserIDContextKey, "user-1")
	return ctx
}

func seedKnowledge(t *testing.T, db *gorm.DB, knowledge *types.Knowledge) {
	t.Helper()
	require.NoError(t, db.Create(knowledge).Error)
}

// setupKnowledgeDB returns an in-memory database with the knowledge table,
// for tests that drive a knowledgeService against real rows.
func setupKnowledgeDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.Knowledge{}))
	return db
}
