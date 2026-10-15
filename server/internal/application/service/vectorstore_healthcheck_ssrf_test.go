package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestTestConnection_QdrantBlocksUnsafeAddrAtDialSink(t *testing.T) {
	svc := NewVectorStoreService(&mockVectorStoreRepo{}, nil, nil, nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := svc.TestConnection(ctx, types.QdrantRetrieverEngineType, types.ConnectionConfig{
		Host: "169.254.169.254",
		Port: 6334,
	})
	if err == nil {
		t.Fatal("expected qdrant connectivity probe to reject unsafe address")
	}
	if !strings.Contains(err.Error(), "failed to connect to qdrant") {
		t.Fatalf("unexpected error: %v", err)
	}
}
