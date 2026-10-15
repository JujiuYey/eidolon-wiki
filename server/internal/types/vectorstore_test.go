package types

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Env store builder, response DTO, types metadata
// ---------------------------------------------------------------------------

// mockEnvLookup creates a simple env lookup function from a map.
func mockEnvLookup(env map[string]string) EnvLookupFunc {
	return func(key string) string {
		return env[key]
	}
}

func TestIsEnvStoreID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected bool
	}{
		{"env postgres ID", "__env_postgres__", true},
		{"env qdrant ID", "__env_qdrant__", true},
		{"env prefix only", "__env_", true},
		{"UUID ID", "550e8400-e29b-41d4-a716-446655440000", false},
		{"empty string", "", false},
		{"similar but not prefix", "_env_postgres__", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsEnvStoreID(tt.id))
		})
	}
}

func TestBuildEnvVectorStores(t *testing.T) {
	envMap := map[string]string{
		"QDRANT_HOST":    "qdrant-host",
		"QDRANT_API_KEY": "qd-key",
	}
	lookup := mockEnvLookup(envMap)

	t.Run("empty RETRIEVE_DRIVER returns nil", func(t *testing.T) {
		stores := BuildEnvVectorStores("", lookup)
		assert.Nil(t, stores)
	})

	t.Run("single driver postgres", func(t *testing.T) {
		stores := BuildEnvVectorStores("postgres", lookup)
		require.Len(t, stores, 1)
		assert.Equal(t, "__env_postgres__", stores[0].ID)
		assert.Equal(t, "PostgreSQL", stores[0].Name)
		assert.Equal(t, PostgresRetrieverEngineType, stores[0].EngineType)
		assert.True(t, stores[0].ConnectionConfig.UseDefaultConnection)
	})

	t.Run("multiple drivers", func(t *testing.T) {
		stores := BuildEnvVectorStores("postgres,qdrant", lookup)
		require.Len(t, stores, 2)
		assert.Equal(t, "__env_postgres__", stores[0].ID)
		assert.Equal(t, "__env_qdrant__", stores[1].ID)
		assert.Equal(t, "qdrant-host", stores[1].ConnectionConfig.Host)
	})

	t.Run("env store retains raw api key (not masked)", func(t *testing.T) {
		stores := BuildEnvVectorStores("qdrant", lookup)
		require.Len(t, stores, 1)
		assert.Equal(t, "qd-key", stores[0].ConnectionConfig.APIKey)
	})

	t.Run("unknown driver is skipped", func(t *testing.T) {
		stores := BuildEnvVectorStores("postgres,unknown_db", lookup)
		require.Len(t, stores, 1)
		assert.Equal(t, "__env_postgres__", stores[0].ID)
	})

	// Engines dropped from this build must be inert in RETRIEVE_DRIVER —
	// skipped like any unknown driver, never half-constructed.
	t.Run("removed engine drivers are skipped", func(t *testing.T) {
		for _, driver := range []string{
			"elasticsearch_v8", "elasticsearch_v7", "infinity", "elasticfaiss",
			"milvus", "weaviate", "doris", "tencent_vectordb", "opensearch", "sqlite",
		} {
			stores := BuildEnvVectorStores(driver, lookup)
			assert.Nil(t, stores, "driver %s should no longer build an env store", driver)
		}
	})

	t.Run("whitespace trimmed", func(t *testing.T) {
		stores := BuildEnvVectorStores(" postgres , qdrant ", lookup)
		require.Len(t, stores, 2)
	})

	t.Run("all supported drivers", func(t *testing.T) {
		stores := BuildEnvVectorStores("postgres,qdrant", lookup)
		require.Len(t, stores, 2)

		ids := make([]string, len(stores))
		for i, s := range stores {
			ids[i] = s.ID
		}
		assert.Contains(t, ids, "__env_postgres__")
		assert.Contains(t, ids, "__env_qdrant__")
	})

	t.Run("qdrant env store", func(t *testing.T) {
		stores := BuildEnvVectorStores("qdrant", lookup)
		require.Len(t, stores, 1)
		assert.Equal(t, "qdrant-host", stores[0].ConnectionConfig.Host)
		assert.Equal(t, "qd-key", stores[0].ConnectionConfig.APIKey)
	})
}

func TestFindEnvVectorStore(t *testing.T) {
	lookup := mockEnvLookup(map[string]string{})

	t.Run("found", func(t *testing.T) {
		store := FindEnvVectorStore("postgres", lookup, "__env_postgres__")
		require.NotNil(t, store)
		assert.Equal(t, "__env_postgres__", store.ID)
	})

	t.Run("not found", func(t *testing.T) {
		store := FindEnvVectorStore("postgres", lookup, "__env_unknown__")
		assert.Nil(t, store)
	})

	t.Run("empty driver returns nil", func(t *testing.T) {
		store := FindEnvVectorStore("", lookup, "__env_postgres__")
		assert.Nil(t, store)
	})
}

func TestNewVectorStoreResponse(t *testing.T) {
	store := &VectorStore{
		ID:         "test-id",
		Name:       "test-store",
		EngineType: QdrantRetrieverEngineType,
		ConnectionConfig: ConnectionConfig{
			Host:     "qdrant",
			Password: "secret",
			APIKey:   "my-api-key",
		},
	}

	t.Run("masks sensitive fields", func(t *testing.T) {
		resp := NewVectorStoreResponse(store, "user", false)
		assert.Equal(t, RedactedSecretPlaceholder, resp.ConnectionConfig.Password)
		assert.Equal(t, RedactedSecretPlaceholder, resp.ConnectionConfig.APIKey)
		assert.Equal(t, "qdrant", resp.ConnectionConfig.Host) // non-sensitive preserved
	})

	t.Run("preserves source and readonly", func(t *testing.T) {
		resp := NewVectorStoreResponse(store, "env", true)
		assert.Equal(t, "env", resp.Source)
		assert.True(t, resp.ReadOnly)
	})

	t.Run("does not mutate original store", func(t *testing.T) {
		_ = NewVectorStoreResponse(store, "user", false)
		assert.Equal(t, "secret", store.ConnectionConfig.Password)
		assert.Equal(t, "my-api-key", store.ConnectionConfig.APIKey)
	})

	t.Run("empty sensitive fields not masked to ***", func(t *testing.T) {
		noSecret := &VectorStore{
			ID:               "test-id",
			ConnectionConfig: ConnectionConfig{Host: "qdrant"},
		}
		resp := NewVectorStoreResponse(noSecret, "user", false)
		assert.Equal(t, "", resp.ConnectionConfig.Password)
		assert.Equal(t, "", resp.ConnectionConfig.APIKey)
	})
}

func TestGetVectorStoreTypes(t *testing.T) {
	types := GetVectorStoreTypes()

	t.Run("returns only qdrant as a registerable external engine", func(t *testing.T) {
		require.Len(t, types, 1)
		assert.Equal(t, "qdrant", types[0].Type)
	})

	t.Run("qdrant exposes connection and index fields", func(t *testing.T) {
		var qdrantType VectorStoreTypeInfo
		for _, typ := range types {
			if typ.Type == "qdrant" {
				qdrantType = typ
				break
			}
		}
		require.NotEmpty(t, qdrantType.ConnectionFields)
		require.NotEmpty(t, qdrantType.IndexFields)

		conn := map[string]VectorStoreFieldInfo{}
		for _, f := range qdrantType.ConnectionFields {
			conn[f.Name] = f
		}
		assert.True(t, conn["host"].Required)
		assert.True(t, conn["api_key"].Sensitive)

		index := map[string]VectorStoreFieldInfo{}
		for _, f := range qdrantType.IndexFields {
			index[f.Name] = f
		}
		assert.Contains(t, index, "collection_prefix")
		assert.Contains(t, index, "shard_number")
		assert.Contains(t, index, "replication_factor")
	})

	t.Run("postgres and sqlite are not offered as DB stores", func(t *testing.T) {
		typeNames := make([]string, 0, len(types))
		for _, typ := range types {
			typeNames = append(typeNames, typ.Type)
		}
		assert.NotContains(t, typeNames, "postgres")
		assert.NotContains(t, typeNames, "sqlite")
	})

	t.Run("display names have no parenthetical suffix", func(t *testing.T) {
		for _, typ := range types {
			assert.NotContains(t, typ.DisplayName, "(", "display_name should not contain parenthetical suffix: %s", typ.DisplayName)
		}
	})
}

// testAESKey is a 32-byte key for testing AES-GCM encryption.
const testAESKey = "01234567890123456789012345678901"

// ---------------------------------------------------------------------------
// VectorStore
// ---------------------------------------------------------------------------

func TestVectorStore_Validate(t *testing.T) {
	valid := VectorStore{
		Name:       "test-store",
		EngineType: QdrantRetrieverEngineType,
		TenantID:   1,
	}

	t.Run("valid input returns nil", func(t *testing.T) {
		assert.NoError(t, valid.Validate())
	})

	t.Run("empty name returns error", func(t *testing.T) {
		s := valid
		s.Name = ""
		err := s.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("unsupported engine type returns error", func(t *testing.T) {
		s := valid
		s.EngineType = "unknown"
		err := s.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported engine type")
	})

	t.Run("zero tenant_id returns error", func(t *testing.T) {
		s := valid
		s.TenantID = 0
		err := s.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant_id is required")
	})
}

func TestVectorStore_BeforeCreate(t *testing.T) {
	t.Run("generates UUID when ID is empty", func(t *testing.T) {
		v := &VectorStore{}
		err := v.BeforeCreate(&gorm.DB{})
		require.NoError(t, err)
		assert.NotEmpty(t, v.ID)
		assert.Len(t, v.ID, 36) // UUID format: 8-4-4-4-12
	})

	t.Run("preserves existing ID", func(t *testing.T) {
		v := &VectorStore{ID: "existing-id"}
		err := v.BeforeCreate(&gorm.DB{})
		require.NoError(t, err)
		assert.Equal(t, "existing-id", v.ID)
	})
}

func TestVectorStore_TableName(t *testing.T) {
	assert.Equal(t, "vector_stores", VectorStore{}.TableName())
}

func TestIsValidEngineType(t *testing.T) {
	validTypes := []RetrieverEngineType{
		QdrantRetrieverEngineType,
	}
	for _, et := range validTypes {
		t.Run("valid: "+string(et), func(t *testing.T) {
			assert.True(t, IsValidEngineType(et))
		})
	}

	// Postgres is intentionally NOT registerable as a DB store — it only
	// makes sense as an env store driven by RETRIEVE_DRIVER (see the doc
	// comment on validEngineTypes). Engines removed from this build are
	// invalid for the same reason: nothing stands behind them anymore.
	invalidTypes := []RetrieverEngineType{
		"unknown",
		"",
		PostgresRetrieverEngineType,
		RetrieverEngineType("sqlite"),
		RetrieverEngineType("elasticsearch"),
		RetrieverEngineType("milvus"),
		RetrieverEngineType("weaviate"),
	}
	for _, et := range invalidTypes {
		name := string(et)
		if name == "" {
			name = "(empty)"
		}
		t.Run("invalid: "+name, func(t *testing.T) {
			assert.False(t, IsValidEngineType(et))
		})
	}
}

// ---------------------------------------------------------------------------
// ConnectionConfig
// ---------------------------------------------------------------------------

func TestConnectionConfig_ValueScan(t *testing.T) {
	t.Run("encrypts password and api_key on Value, decrypts on Scan", func(t *testing.T) {
		t.Setenv("SYSTEM_AES_KEY", testAESKey)

		original := ConnectionConfig{
			Addr:     "qdrant:6334",
			Username: "admin",
			Password: "secret-pass",
			APIKey:   "sk-api-key",
		}

		// Value — encrypt
		raw, err := original.Value()
		require.NoError(t, err)

		// Verify the serialized JSON has encrypted fields
		var intermediate map[string]interface{}
		require.NoError(t, json.Unmarshal(raw.([]byte), &intermediate))
		assert.True(t, strings.HasPrefix(intermediate["password"].(string), "enc:v1:"))
		assert.True(t, strings.HasPrefix(intermediate["api_key"].(string), "enc:v1:"))
		// Non-sensitive fields remain plaintext
		assert.Equal(t, "qdrant:6334", intermediate["addr"])
		assert.Equal(t, "admin", intermediate["username"])

		// Scan — decrypt
		var scanned ConnectionConfig
		err = scanned.Scan(raw.([]byte))
		require.NoError(t, err)
		assert.Equal(t, "secret-pass", scanned.Password)
		assert.Equal(t, "sk-api-key", scanned.APIKey)
		assert.Equal(t, "qdrant:6334", scanned.Addr)
		assert.Equal(t, "admin", scanned.Username)
	})

	t.Run("skips encryption when fields are empty", func(t *testing.T) {
		t.Setenv("SYSTEM_AES_KEY", testAESKey)

		original := ConnectionConfig{Addr: "qdrant:6334"}
		raw, err := original.Value()
		require.NoError(t, err)

		var intermediate map[string]interface{}
		require.NoError(t, json.Unmarshal(raw.([]byte), &intermediate))
		_, hasPassword := intermediate["password"]
		_, hasAPIKey := intermediate["api_key"]
		assert.False(t, hasPassword)
		assert.False(t, hasAPIKey)
	})

	t.Run("skips encryption when AES key is not set", func(t *testing.T) {
		t.Setenv("SYSTEM_AES_KEY", "")

		original := ConnectionConfig{
			Password: "secret-pass",
			APIKey:   "sk-api-key",
		}
		raw, err := original.Value()
		require.NoError(t, err)

		var intermediate map[string]interface{}
		require.NoError(t, json.Unmarshal(raw.([]byte), &intermediate))
		assert.Equal(t, "secret-pass", intermediate["password"])
		assert.Equal(t, "sk-api-key", intermediate["api_key"])
	})

	t.Run("does not double-encrypt already encrypted values", func(t *testing.T) {
		t.Setenv("SYSTEM_AES_KEY", testAESKey)

		original := ConnectionConfig{Password: "secret-pass"}
		raw1, err := original.Value()
		require.NoError(t, err)

		// Scan to get the encrypted form, then re-serialize
		var scanned ConnectionConfig
		require.NoError(t, json.Unmarshal(raw1.([]byte), &scanned))
		// scanned.Password is now "enc:v1:..."
		raw2, err := scanned.Value()
		require.NoError(t, err)

		// Both serialized forms should produce the same decrypted result
		var result ConnectionConfig
		require.NoError(t, result.Scan(raw2.([]byte)))
		assert.Equal(t, "secret-pass", result.Password)
	})

	t.Run("Scan nil value returns no error", func(t *testing.T) {
		var c ConnectionConfig
		assert.NoError(t, c.Scan(nil))
	})

	t.Run("Scan non-byte value returns no error", func(t *testing.T) {
		var c ConnectionConfig
		assert.NoError(t, c.Scan(42))
	})

	t.Run("original struct is not mutated by Value", func(t *testing.T) {
		t.Setenv("SYSTEM_AES_KEY", testAESKey)

		original := ConnectionConfig{Password: "secret-pass"}
		_, err := original.Value()
		require.NoError(t, err)
		assert.Equal(t, "secret-pass", original.Password, "value receiver should not mutate original")
	})
}

func TestConnectionConfig_GetEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		config   ConnectionConfig
		expected string
	}{
		{
			name:     "returns Addr when set",
			config:   ConnectionConfig{Addr: "qdrant:6334"},
			expected: "qdrant:6334",
		},
		{
			name:     "returns host:port when Host and Port set",
			config:   ConnectionConfig{Host: "qdrant-prod", Port: 6334},
			expected: "qdrant-prod:6334",
		},
		{
			name:     "defaults Port to 6334 when Host set and Port is 0",
			config:   ConnectionConfig{Host: "qdrant-prod"},
			expected: "qdrant-prod:6334",
		},
		{
			name:     "returns sentinel for default postgres connection",
			config:   ConnectionConfig{UseDefaultConnection: true},
			expected: "__default_postgres__",
		},
		{
			name:     "returns empty string when nothing is set",
			config:   ConnectionConfig{},
			expected: "",
		},
		{
			name:     "Addr takes precedence over Host",
			config:   ConnectionConfig{Addr: "qdrant:6334", Host: "qdrant"},
			expected: "qdrant:6334",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.GetEndpoint())
		})
	}
}

func TestConnectionConfig_MaskSensitiveFields(t *testing.T) {
	t.Run("masks password and api_key", func(t *testing.T) {
		c := ConnectionConfig{
			Addr:     "qdrant:6334",
			Username: "admin",
			Password: "secret-pass",
			APIKey:   "sk-api-key",
		}
		masked := c.MaskSensitiveFields()
		assert.Equal(t, RedactedSecretPlaceholder, masked.Password)
		assert.Equal(t, RedactedSecretPlaceholder, masked.APIKey)
		assert.Equal(t, "qdrant:6334", masked.Addr)
		assert.Equal(t, "admin", masked.Username)
	})

	t.Run("does not mask empty fields", func(t *testing.T) {
		c := ConnectionConfig{Addr: "qdrant:6334"}
		masked := c.MaskSensitiveFields()
		assert.Empty(t, masked.Password)
		assert.Empty(t, masked.APIKey)
	})

	t.Run("does not mutate original", func(t *testing.T) {
		c := ConnectionConfig{Password: "secret-pass", APIKey: "sk-api-key"}
		_ = c.MaskSensitiveFields()
		assert.Equal(t, "secret-pass", c.Password)
		assert.Equal(t, "sk-api-key", c.APIKey)
	})
}

// ---------------------------------------------------------------------------
// IndexConfig
// ---------------------------------------------------------------------------

func TestIndexConfig_ValueScan(t *testing.T) {
	t.Run("round-trip serialization", func(t *testing.T) {
		original := IndexConfig{
			IndexName:         "my_index",
			ShardNumber:       4,
			ReplicationFactor: 2,
		}
		raw, err := original.Value()
		require.NoError(t, err)

		var scanned IndexConfig
		require.NoError(t, scanned.Scan(raw.([]byte)))
		assert.Equal(t, original, scanned)
	})

	t.Run("empty config serializes to {}", func(t *testing.T) {
		raw, err := IndexConfig{}.Value()
		require.NoError(t, err)
		assert.JSONEq(t, `{}`, string(raw.([]byte)))
	})

	t.Run("Scan nil value returns no error", func(t *testing.T) {
		var c IndexConfig
		assert.NoError(t, c.Scan(nil))
	})

	t.Run("Scan non-byte value returns no error", func(t *testing.T) {
		var c IndexConfig
		assert.NoError(t, c.Scan(42))
	})
}

func TestIndexConfig_GetIndexNameOrDefault(t *testing.T) {
	tests := []struct {
		name       string
		config     IndexConfig
		engineType RetrieverEngineType
		expected   string
	}{
		// Qdrant
		{
			name:       "qdrant with custom collection prefix",
			config:     IndexConfig{CollectionPrefix: "custom_embeddings"},
			engineType: QdrantRetrieverEngineType,
			expected:   "custom_embeddings",
		},
		{
			name:       "qdrant default",
			config:     IndexConfig{},
			engineType: QdrantRetrieverEngineType,
			expected:   "weknora_embeddings",
		},
		{
			// GetIndexNameOrDefault honors only the prefix for qdrant; the
			// plain CollectionName fallback lives in ResolveCollectionName,
			// which the repository layer uses instead.
			name:       "qdrant ignores plain collection name",
			config:     IndexConfig{CollectionName: "custom_collection"},
			engineType: QdrantRetrieverEngineType,
			expected:   "weknora_embeddings",
		},
		// Postgres (index_name passthrough, no engine default)
		{
			name:       "postgres returns index_name",
			config:     IndexConfig{IndexName: "pg_index"},
			engineType: PostgresRetrieverEngineType,
			expected:   "pg_index",
		},
		{
			name:       "postgres returns empty when unset",
			config:     IndexConfig{},
			engineType: PostgresRetrieverEngineType,
			expected:   "",
		},
		// Unknown engine falls into the same passthrough branch.
		{
			name:       "unknown engine falls back to index_name",
			config:     IndexConfig{IndexName: "idx"},
			engineType: RetrieverEngineType("nosuch"),
			expected:   "idx",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.GetIndexNameOrDefault(tt.engineType))
		})
	}
}

// ---------------------------------------------------------------------------
// IndexConfig — getter helpers
// ---------------------------------------------------------------------------

func TestIndexConfig_GetterHelpers(t *testing.T) {
	t.Run("nil receiver returns default", func(t *testing.T) {
		var ic *IndexConfig
		assert.Equal(t, 0, ic.GetShardNumber(0))
		assert.Equal(t, 1, ic.GetShardNumber(1))
		assert.Equal(t, 0, ic.GetReplicationFactor(0))
	})

	t.Run("zero value returns default", func(t *testing.T) {
		ic := &IndexConfig{}
		assert.Equal(t, 0, ic.GetShardNumber(0))
		assert.Equal(t, 1, ic.GetShardNumber(1))
		assert.Equal(t, 0, ic.GetReplicationFactor(0))
	})

	t.Run("positive value overrides default", func(t *testing.T) {
		ic := &IndexConfig{
			ShardNumber:       4,
			ReplicationFactor: 3,
		}
		assert.Equal(t, 4, ic.GetShardNumber(0))
		assert.Equal(t, 3, ic.GetReplicationFactor(0))
	})

	t.Run("negative value returns default (treated as unset)", func(t *testing.T) {
		ic := &IndexConfig{
			ShardNumber:       -5,
			ReplicationFactor: -1,
		}
		assert.Equal(t, 0, ic.GetShardNumber(0))
		assert.Equal(t, 0, ic.GetReplicationFactor(0))
	})
}

// ---------------------------------------------------------------------------
// IndexConfig — resolve helpers
// ---------------------------------------------------------------------------

func TestResolveIndexName(t *testing.T) {
	t.Run("nil IndexConfig falls back to env var", func(t *testing.T) {
		t.Setenv("TEST_INDEX", "env_index")
		assert.Equal(t, "env_index", ResolveIndexName(nil, "TEST_INDEX", "default"))
	})

	t.Run("nil IndexConfig falls back to default when env empty", func(t *testing.T) {
		t.Setenv("TEST_INDEX", "")
		assert.Equal(t, "xwrag_default", ResolveIndexName(nil, "TEST_INDEX", "xwrag_default"))
	})

	t.Run("IndexConfig value takes precedence over env var", func(t *testing.T) {
		t.Setenv("TEST_INDEX", "env_index")
		ic := &IndexConfig{IndexName: "custom_index"}
		assert.Equal(t, "custom_index", ResolveIndexName(ic, "TEST_INDEX", "default"))
	})

	t.Run("empty IndexConfig.IndexName falls back to env var", func(t *testing.T) {
		t.Setenv("TEST_INDEX", "env_index")
		ic := &IndexConfig{}
		assert.Equal(t, "env_index", ResolveIndexName(ic, "TEST_INDEX", "default"))
	})
}

func TestResolveCollectionName(t *testing.T) {
	t.Run("nil IndexConfig falls back to env var", func(t *testing.T) {
		t.Setenv("QDRANT_COLLECTION", "env_collection")
		assert.Equal(t, "env_collection", ResolveCollectionName(nil, "QDRANT_COLLECTION", "default"))
	})

	t.Run("CollectionPrefix takes precedence over CollectionName", func(t *testing.T) {
		ic := &IndexConfig{CollectionPrefix: "prefix_name", CollectionName: "full_name"}
		assert.Equal(t, "prefix_name", ResolveCollectionName(ic, "QDRANT_COLLECTION", "default"))
	})

	t.Run("CollectionName used when CollectionPrefix empty", func(t *testing.T) {
		ic := &IndexConfig{CollectionName: "full_name"}
		assert.Equal(t, "full_name", ResolveCollectionName(ic, "TEST_COLLECTION", "default"))
	})

	t.Run("empty IndexConfig falls back to default", func(t *testing.T) {
		t.Setenv("QDRANT_COLLECTION", "")
		ic := &IndexConfig{}
		assert.Equal(t, "weknora_embeddings", ResolveCollectionName(ic, "QDRANT_COLLECTION", "weknora_embeddings"))
	})
}

// ---------------------------------------------------------------------------
// OptionalUint32
// ---------------------------------------------------------------------------

func TestOptionalUint32(t *testing.T) {
	t.Run("zero returns nil", func(t *testing.T) {
		assert.Nil(t, OptionalUint32(0))
	})

	t.Run("negative returns nil", func(t *testing.T) {
		assert.Nil(t, OptionalUint32(-1))
		assert.Nil(t, OptionalUint32(-100))
	})

	t.Run("positive returns pointer to uint32", func(t *testing.T) {
		result := OptionalUint32(3)
		require.NotNil(t, result)
		assert.Equal(t, uint32(3), *result)
	})

	t.Run("large positive value", func(t *testing.T) {
		result := OptionalUint32(64)
		require.NotNil(t, result)
		assert.Equal(t, uint32(64), *result)
	})
}

// ---------------------------------------------------------------------------
// ValidateIndexConfig
// ---------------------------------------------------------------------------

func TestValidateIndexConfig(t *testing.T) {
	t.Run("empty config is valid", func(t *testing.T) {
		assert.NoError(t, ValidateIndexConfig(IndexConfig{}))
	})

	t.Run("valid config with all fields", func(t *testing.T) {
		ic := IndexConfig{
			IndexName:         "my_index",
			CollectionPrefix:  "my_collection",
			CollectionName:    "custom-collection-name",
			ShardNumber:       4,
			ReplicationFactor: 2,
		}
		assert.NoError(t, ValidateIndexConfig(ic))
	})

	// --- Name validation ---
	t.Run("index_name with special chars rejected", func(t *testing.T) {
		ic := IndexConfig{IndexName: "my index*"}
		err := ValidateIndexConfig(ic)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "index_name")
	})

	t.Run("index_name starting with number rejected", func(t *testing.T) {
		ic := IndexConfig{IndexName: "123abc"}
		err := ValidateIndexConfig(ic)
		require.Error(t, err)
	})

	t.Run("collection_prefix with slash rejected", func(t *testing.T) {
		ic := IndexConfig{CollectionPrefix: "path/to/collection"}
		err := ValidateIndexConfig(ic)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "collection_prefix")
	})

	t.Run("collection_name with dot rejected", func(t *testing.T) {
		ic := IndexConfig{CollectionName: "my.collection"}
		err := ValidateIndexConfig(ic)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "collection_name")
	})

	t.Run("valid names with underscore and hyphen", func(t *testing.T) {
		ic := IndexConfig{
			IndexName:        "my_index-v2",
			CollectionPrefix: "Weknora_embeddings",
			CollectionName:   "custom-collection-name",
		}
		assert.NoError(t, ValidateIndexConfig(ic))
	})

	// --- Numeric bounds ---
	t.Run("shard_number exceeds max", func(t *testing.T) {
		ic := IndexConfig{ShardNumber: 100}
		err := ValidateIndexConfig(ic)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "shard_number")
	})

	t.Run("negative shard_number rejected", func(t *testing.T) {
		ic := IndexConfig{ShardNumber: -1}
		err := ValidateIndexConfig(ic)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "shard_number")
	})

	t.Run("replication_factor exceeds max", func(t *testing.T) {
		ic := IndexConfig{ReplicationFactor: 50}
		err := ValidateIndexConfig(ic)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "replication_factor")
	})

	t.Run("shard_number at max boundary is valid", func(t *testing.T) {
		ic := IndexConfig{ShardNumber: 64}
		assert.NoError(t, ValidateIndexConfig(ic))
	})

	t.Run("replication_factor at max boundary is valid", func(t *testing.T) {
		ic := IndexConfig{ReplicationFactor: 10}
		assert.NoError(t, ValidateIndexConfig(ic))
	})
}

// ---------------------------------------------------------------------------
// IndexConfig — scalability fields round-trip
// ---------------------------------------------------------------------------

func TestIndexConfig_ScalabilityFieldsRoundTrip(t *testing.T) {
	t.Run("scalability fields serialize and deserialize", func(t *testing.T) {
		original := IndexConfig{
			IndexName:         "my_index",
			CollectionPrefix:  "my_prefix",
			ShardNumber:       4,
			ReplicationFactor: 2,
		}
		raw, err := original.Value()
		require.NoError(t, err)

		var scanned IndexConfig
		require.NoError(t, scanned.Scan(raw.([]byte)))
		assert.Equal(t, original, scanned)
	})

	t.Run("scalability fields omitted when zero", func(t *testing.T) {
		raw, err := IndexConfig{IndexName: "test"}.Value()
		require.NoError(t, err)

		var parsed map[string]interface{}
		require.NoError(t, json.Unmarshal(raw.([]byte), &parsed))
		assert.NotContains(t, parsed, "shard_number")
		assert.NotContains(t, parsed, "replication_factor")
	})
}

// TestVectorStore_PostgresSqliteNotRegisterable pins the write-path and
// read-path consistency for the engines that are only meaningful as env
// stores (postgres) or not shipped at all (sqlite). They must:
//
//  1. Be rejected by Validate() so POST /vector-stores returns a 4xx
//     instead of silently persisting a row that has no separation effect.
//  2. Be absent from GetVectorStoreTypes() so the UI dropdown doesn't
//     offer them as a choice.
//
// Both checks live here so a future change that re-introduces one path
// fails this test pair instead of silently re-opening the inconsistency.
func TestVectorStore_PostgresSqliteNotRegisterable(t *testing.T) {
	t.Run("Validate rejects postgres as DB store", func(t *testing.T) {
		v := &VectorStore{
			Name: "test", TenantID: 1,
			EngineType: PostgresRetrieverEngineType,
		}
		err := v.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported engine type")
	})

	t.Run("Validate rejects sqlite as DB store", func(t *testing.T) {
		v := &VectorStore{
			Name: "test", TenantID: 1,
			EngineType: RetrieverEngineType("sqlite"),
		}
		err := v.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported engine type")
	})

	t.Run("GetVectorStoreTypes omits postgres and sqlite", func(t *testing.T) {
		listed := GetVectorStoreTypes()
		var got []string
		for _, info := range listed {
			got = append(got, info.Type)
		}
		assert.NotContains(t, got, string(PostgresRetrieverEngineType),
			"postgres must not appear in the UI dropdown — env-store only")
		assert.NotContains(t, got, "sqlite",
			"sqlite must not appear in the UI dropdown — not shipped")
	})
}

// ---------------------------------------------------------------------------
// VectorStoreFieldInfo schema extensions (Immutable / Min / Max / Enum)
// ---------------------------------------------------------------------------

// TestVectorStoreFieldInfo_OmitemptyPreservesWire confirms that the
// four optional fields (Immutable / Min / Max / Enum) are omitted from
// JSON when unset, so VectorStoreFieldInfo entries without these fields
// serialize identically before and after the schema extension.
func TestVectorStoreFieldInfo_OmitemptyPreservesWire(t *testing.T) {
	legacy := VectorStoreFieldInfo{
		Name:        "addr",
		Type:        "string",
		Required:    true,
		Description: "URL",
		Default:     "http://localhost:9200",
	}
	out, err := json.Marshal(legacy)
	require.NoError(t, err)
	// The optional fields must all be omitted from the wire format.
	assert.NotContains(t, string(out), `"immutable"`)
	assert.NotContains(t, string(out), `"min"`)
	assert.NotContains(t, string(out), `"max"`)
	assert.NotContains(t, string(out), `"enum"`)
}

// TestVectorStoreFieldInfo_NewFieldsSerialize verifies the optional fields
// appear in JSON when set.
func TestVectorStoreFieldInfo_NewFieldsSerialize(t *testing.T) {
	t.Run("immutable", func(t *testing.T) {
		f := VectorStoreFieldInfo{Name: "knn_engine", Type: "string", Immutable: true}
		out, err := json.Marshal(f)
		require.NoError(t, err)
		assert.Contains(t, string(out), `"immutable":true`)
	})

	t.Run("min and max", func(t *testing.T) {
		minV, maxV := float64(4), float64(64)
		f := VectorStoreFieldInfo{Name: "hnsw_m", Type: "number", Min: &minV, Max: &maxV}
		out, err := json.Marshal(f)
		require.NoError(t, err)
		assert.Contains(t, string(out), `"min":4`)
		assert.Contains(t, string(out), `"max":64`)
	})

	t.Run("enum", func(t *testing.T) {
		f := VectorStoreFieldInfo{
			Name: "knn_engine", Type: "string",
			Enum: []string{"lucene", "faiss"},
		}
		out, err := json.Marshal(f)
		require.NoError(t, err)
		assert.Contains(t, string(out), `"enum":["lucene","faiss"]`)
	})

	t.Run("empty enum omits via omitempty", func(t *testing.T) {
		f := VectorStoreFieldInfo{Name: "addr", Type: "string", Enum: []string{}}
		out, err := json.Marshal(f)
		require.NoError(t, err)
		assert.NotContains(t, string(out), `"enum"`)
	})
}

// TestVectorStoreFieldInfo_RoundTrip ensures deserialization back into
// the struct works for all optional fields, including the *float64 pointer
// distinction (nil vs explicit 0).
func TestVectorStoreFieldInfo_RoundTrip(t *testing.T) {
	t.Run("min=0 deserializes as &0, not nil", func(t *testing.T) {
		raw := `{"name":"replicas","type":"number","min":0,"max":5}`
		var f VectorStoreFieldInfo
		require.NoError(t, json.Unmarshal([]byte(raw), &f))
		require.NotNil(t, f.Min)
		assert.Equal(t, float64(0), *f.Min)
		require.NotNil(t, f.Max)
		assert.Equal(t, float64(5), *f.Max)
	})

	t.Run("missing min/max deserialize as nil", func(t *testing.T) {
		raw := `{"name":"addr","type":"string"}`
		var f VectorStoreFieldInfo
		require.NoError(t, json.Unmarshal([]byte(raw), &f))
		assert.Nil(t, f.Min)
		assert.Nil(t, f.Max)
	})

	t.Run("min and max are independent pointers", func(t *testing.T) {
		// Setting only Min must leave Max nil (and vice versa). A
		// refactor that accidentally aliases the two via a shared
		// local would still pass the previous two subtests because
		// they always set both ends.
		raw := `{"name":"upper_only","type":"number","max":10}`
		var f VectorStoreFieldInfo
		require.NoError(t, json.Unmarshal([]byte(raw), &f))
		assert.Nil(t, f.Min)
		require.NotNil(t, f.Max)
		assert.Equal(t, float64(10), *f.Max)
	})

	t.Run("nil enum omits via omitempty", func(t *testing.T) {
		// `nil` and `[]string{}` both round-trip through omitempty
		// identically — pin both shapes so a future Go encoder
		// behavior change cannot regress the wire format.
		f := VectorStoreFieldInfo{Name: "addr", Type: "string"} // Enum nil
		out, err := json.Marshal(f)
		require.NoError(t, err)
		assert.NotContains(t, string(out), `"enum"`)
	})
}
