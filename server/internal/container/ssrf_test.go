package container

import (
	"os"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/utils"
)

func TestMain(m *testing.M) {
	// Engine wiring tests intentionally use loopback httptest servers.
	utils.SetSSRFWhitelistFromRaw("127.0.0.1,::1,localhost")
	code := m.Run()
	utils.SetSSRFWhitelistFromRaw("")
	os.Exit(code)
}

func TestValidateRuntimeVectorStoreAddressesRejectsUnsafeEndpoints(t *testing.T) {
	tests := []types.VectorStore{
		{EngineType: types.QdrantRetrieverEngineType, ConnectionConfig: types.ConnectionConfig{Host: "169.254.169.254", Port: 6334}},
		// Unknown engine types carry no SSRF address policy and must be
		// rejected outright (fail-closed) rather than silently passing.
		{EngineType: types.RetrieverEngineType("some-future-engine"), ConnectionConfig: types.ConnectionConfig{Host: "169.254.169.254"}},
	}
	for _, store := range tests {
		if err := validateRuntimeVectorStoreAddresses(store); err == nil {
			t.Fatalf("expected %s unsafe address to be rejected", store.EngineType)
		}
	}
}

func TestValidateRuntimeVectorStoreAddressesAllowsNonNetworkEngines(t *testing.T) {
	for _, engineType := range []types.RetrieverEngineType{
		types.PostgresRetrieverEngineType,
	} {
		if err := validateRuntimeVectorStoreAddresses(types.VectorStore{EngineType: engineType}); err != nil {
			t.Fatalf("unexpected %s validation error: %v", engineType, err)
		}
	}
}
