package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EnvStoreIDPrefix is the prefix for virtual env store IDs.
const EnvStoreIDPrefix = "__env_"

// IsEnvStoreID checks if the given ID is an env store virtual ID.
func IsEnvStoreID(id string) bool {
	return strings.HasPrefix(id, EnvStoreIDPrefix)
}

// EnvLookupFunc is a function type for looking up environment variables.
// In production: os.Getenv, in tests: custom lookup function.
type EnvLookupFunc func(string) string

// VectorStore represents a configured vector database instance for a workspace.
// Each workspace can register multiple VectorStore entries (even of the same engine type)
// to support multi-store scenarios (e.g., separate Qdrant clusters).
type VectorStore struct {
	// Unique identifier (UUID, auto-generated)
	ID string `yaml:"id" json:"id" gorm:"type:varchar(36);primaryKey"`
	// Workspace ID for scoping
	TenantID uint64 `yaml:"tenant_id" json:"tenant_id"`
	// User-friendly name, e.g., "qdrant-hot"
	Name string `yaml:"name" json:"name" gorm:"type:varchar(255);not null"`
	// Engine type: postgres, qdrant
	EngineType RetrieverEngineType `yaml:"engine_type" json:"engine_type" gorm:"type:varchar(50);not null"`
	// Driver-specific connection parameters (sensitive fields encrypted with AES-GCM)
	ConnectionConfig ConnectionConfig `yaml:"connection_config" json:"connection_config" gorm:"type:json"`
	// Optional index/collection configuration (engine-specific defaults if empty)
	IndexConfig IndexConfig `yaml:"index_config" json:"index_config" gorm:"type:json"`
	// Timestamps
	CreatedAt time.Time      `yaml:"created_at" json:"created_at"`
	UpdatedAt time.Time      `yaml:"updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `yaml:"deleted_at" json:"deleted_at" gorm:"index"`
}

// TableName returns the table name for VectorStore
func (VectorStore) TableName() string {
	return "vector_stores"
}

// BeforeCreate is a GORM hook that runs before creating a new record.
// Automatically generates a UUID for new vector stores.
func (v *VectorStore) BeforeCreate(tx *gorm.DB) error {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	return nil
}

// validEngineTypes defines the engine types that can be registered as a
// DB-managed VectorStore (i.e., persisted to the vector_stores table and
// listed in GetVectorStoreTypes for the UI dropdown).
//
// Excluded engines:
//   - Postgres — only meaningful when bound to the app's default DB
//     connection (UseDefaultConnection=true). The Postgres retriever's
//     embeddings table is a single hard-coded name with no per-store
//     partitioning, so registering a second Postgres store on the same
//     instance has no separation effect — every KB sharing this engine
//     ends up in the same physical table. It is still reachable via the
//     env store (RETRIEVE_DRIVER=postgres), which routes through a
//     separate code path (BuildEnvVectorStores) and does not pass
//     through this validation.
var validEngineTypes = map[RetrieverEngineType]bool{
	QdrantRetrieverEngineType: true,
}

// IsValidEngineType checks whether the given engine type is valid for VectorStore.
func IsValidEngineType(t RetrieverEngineType) bool {
	return validEngineTypes[t]
}

// Validate checks required fields and engine type validity.
func (v *VectorStore) Validate() error {
	if v.Name == "" {
		return errors.NewValidationError("name is required")
	}
	if !validEngineTypes[v.EngineType] {
		return errors.NewValidationError(fmt.Sprintf("unsupported engine type: %s", v.EngineType))
	}
	if v.TenantID == 0 {
		return errors.NewValidationError("tenant_id is required")
	}
	return nil
}

// ---------------------------------------------------------------------------
// ConnectionConfig
// ---------------------------------------------------------------------------

// ConnectionConfig holds driver-specific connection parameters.
// Sensitive fields (Password, APIKey) are encrypted with AES-GCM at rest.
type ConnectionConfig struct {
	// Common
	Addr     string `yaml:"addr" json:"addr,omitempty"`
	Username string `yaml:"username" json:"username,omitempty"`
	Password string `yaml:"password" json:"password,omitempty"` // AES-GCM encrypted
	APIKey   string `yaml:"api_key" json:"api_key,omitempty"`   // AES-GCM encrypted
	// Qdrant
	Host   string `yaml:"host" json:"host,omitempty"`
	Port   int    `yaml:"port" json:"port,omitempty"`
	UseTLS bool   `yaml:"use_tls" json:"use_tls,omitempty"`
	// Postgres
	UseDefaultConnection bool `yaml:"use_default_connection" json:"use_default_connection,omitempty"`
	// Version is the detected server version (e.g., "16.2", "1.12.6").
	// Auto-populated by TestConnection on successful connectivity check.
	Version string `yaml:"version" json:"version,omitempty"`
}

// Value implements the driver.Valuer interface.
// Encrypts Password and APIKey before persisting to database.
func (c ConnectionConfig) Value() (driver.Value, error) {
	if key := utils.GetAESKey(); key != nil {
		if c.Password != "" {
			if encrypted, err := utils.EncryptAESGCM(c.Password, key); err == nil {
				c.Password = encrypted
			}
		}
		if c.APIKey != "" {
			if encrypted, err := utils.EncryptAESGCM(c.APIKey, key); err == nil {
				c.APIKey = encrypted
			}
		}
	}
	return json.Marshal(c)
}

// Scan implements the sql.Scanner interface.
// Decrypts Password and APIKey after loading from database.
func (c *ConnectionConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	if err := json.Unmarshal(b, c); err != nil {
		return err
	}
	password, err := utils.DecryptStoredSecret(c.Password)
	if err != nil {
		return fmt.Errorf("decrypt vector store connection password: %w", err)
	}
	c.Password = password
	apiKey, err := utils.DecryptStoredSecret(c.APIKey)
	if err != nil {
		return fmt.Errorf("decrypt vector store connection api_key: %w", err)
	}
	c.APIKey = apiKey
	return nil
}

// GetEndpoint returns a normalized endpoint string for duplicate detection.
func (c ConnectionConfig) GetEndpoint() string {
	if c.Addr != "" {
		return c.Addr
	}
	if c.Host != "" {
		port := c.Port
		if port == 0 {
			port = 6334 // Qdrant default port
		}
		return fmt.Sprintf("%s:%d", c.Host, port)
	}
	if c.UseDefaultConnection {
		return "__default_postgres__"
	}
	return ""
}

// MaskSensitiveFields returns a copy with Password and APIKey replaced by the
// shared RedactedSecretPlaceholder. Empty values stay empty so the frontend
// can distinguish "set (hidden)" from "not set" without an extra flag.
func (c ConnectionConfig) MaskSensitiveFields() ConnectionConfig {
	masked := c
	if masked.Password != "" {
		masked.Password = RedactedSecretPlaceholder
	}
	if masked.APIKey != "" {
		masked.APIKey = RedactedSecretPlaceholder
	}
	return masked
}

// ---------------------------------------------------------------------------
// IndexConfig
// ---------------------------------------------------------------------------

// IndexConfig holds optional index/collection configuration for the vector store.
// If empty, engine-specific defaults are used.
type IndexConfig struct {
	IndexName        string `yaml:"index_name" json:"index_name,omitempty"`               // Postgres
	CollectionPrefix string `yaml:"collection_prefix" json:"collection_prefix,omitempty"` // Qdrant
	CollectionName   string `yaml:"collection_name" json:"collection_name,omitempty"`     // fallback when no prefix is set

	// --- Scalability fields ---
	ShardNumber       int `yaml:"shard_number" json:"shard_number,omitempty"`             // Qdrant: number of shards per collection
	ReplicationFactor int `yaml:"replication_factor" json:"replication_factor,omitempty"` // Qdrant: number of replicas
}

// Value implements the driver.Valuer interface.
func (c IndexConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implements the sql.Scanner interface.
func (c *IndexConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, c)
}

// GetIndexNameOrDefault returns the effective index/collection name,
// falling back to engine-specific defaults when the user has not specified one.
func (c IndexConfig) GetIndexNameOrDefault(engineType RetrieverEngineType) string {
	switch engineType {
	case QdrantRetrieverEngineType:
		if c.CollectionPrefix != "" {
			return c.CollectionPrefix
		}
		return "weknora_embeddings"
	default:
		return c.IndexName
	}
}

// ---------------------------------------------------------------------------
// IndexConfig — getter helpers (pointer receiver for nil safety)
// ---------------------------------------------------------------------------

// GetShardNumber returns the configured shard_number (Qdrant), or def if unset/zero.
func (c *IndexConfig) GetShardNumber(def int) int {
	if c != nil && c.ShardNumber > 0 {
		return c.ShardNumber
	}
	return def
}

// GetReplicationFactor returns the configured replication_factor (Qdrant), or def if unset/zero.
func (c *IndexConfig) GetReplicationFactor(def int) int {
	if c != nil && c.ReplicationFactor > 0 {
		return c.ReplicationFactor
	}
	return def
}

// ---------------------------------------------------------------------------
// IndexConfig — resolve helpers (for Repository layer, with env var fallback)
// ---------------------------------------------------------------------------

// ResolveIndexName returns the index name from IndexConfig, falling back to env var and then default.
// Used by Repository constructors. For service-layer duplicate checking, use GetIndexNameOrDefault instead.
func ResolveIndexName(ic *IndexConfig, envKey, defaultVal string) string {
	if ic != nil && ic.IndexName != "" {
		return ic.IndexName
	}
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return defaultVal
}

// ResolveCollectionName returns the collection name from IndexConfig, falling back to env var and then default.
// Priority: CollectionPrefix > CollectionName > env var > defaultVal.
// CollectionPrefix is checked first because Qdrant uses it as the collection
// base name; CollectionName is the plain fallback. If both are set,
// CollectionPrefix wins.
func ResolveCollectionName(ic *IndexConfig, envKey, defaultVal string) string {
	if ic != nil {
		if ic.CollectionPrefix != "" {
			return ic.CollectionPrefix
		}
		if ic.CollectionName != "" {
			return ic.CollectionName
		}
	}
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return defaultVal
}

// OptionalUint32 converts int to *uint32 for Qdrant SDK.
// Returns nil for values <= 0, which tells Qdrant to use its server default.
func OptionalUint32(v int) *uint32 {
	if v <= 0 {
		return nil
	}
	u := uint32(v)
	return &u
}

// ---------------------------------------------------------------------------
// IndexConfig — validation
// ---------------------------------------------------------------------------

const (
	// maxShards is the upper bound for shard-related configuration values.
	maxShards = 64
	// maxReplicas is the upper bound for replication-related configuration values.
	maxReplicas = 10
)

// validIndexNamePattern restricts index/collection names to safe characters.
// Must start with a letter, followed by alphanumeric, underscore, or hyphen. Max 128 chars.
var validIndexNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,127}$`)

// ValidateIndexConfig checks IndexConfig fields for safe values.
// Call this from the service layer before persisting a VectorStore.
func ValidateIndexConfig(ic IndexConfig) error {
	// Validate string fields (index/collection names)
	if ic.IndexName != "" && !validIndexNamePattern.MatchString(ic.IndexName) {
		return errors.NewValidationError(
			"index_name must start with a letter and contain only alphanumeric, underscore, or hyphen characters (max 128)")
	}
	if ic.CollectionPrefix != "" && !validIndexNamePattern.MatchString(ic.CollectionPrefix) {
		return errors.NewValidationError(
			"collection_prefix must start with a letter and contain only alphanumeric, underscore, or hyphen characters (max 128)")
	}
	if ic.CollectionName != "" && !validIndexNamePattern.MatchString(ic.CollectionName) {
		return errors.NewValidationError(
			"collection_name must start with a letter and contain only alphanumeric, underscore, or hyphen characters (max 128)")
	}

	// Validate numeric fields (shards/replicas) — must be within safe bounds
	if ic.ShardNumber < 0 || ic.ShardNumber > maxShards {
		return errors.NewValidationError(fmt.Sprintf("shard_number must be between 0 and %d", maxShards))
	}
	if ic.ReplicationFactor < 0 || ic.ReplicationFactor > maxReplicas {
		return errors.NewValidationError(fmt.Sprintf("replication_factor must be between 0 and %d", maxReplicas))
	}

	return nil
}

// ---------------------------------------------------------------------------
// StoreDisplay — API-safe projection embedded in other resource responses
// ---------------------------------------------------------------------------

// Vector store source classifiers used by API responses.
// Kept as package-level constants so handlers and services share a single
// vocabulary instead of repeating magic strings.
const (
	StoreSourceEnv         = "env"         // env-driven (RETRIEVE_DRIVER)
	StoreSourceUser        = "user"        // DB-managed VectorStore row
	StoreSourceUnavailable = "unavailable" // bound store row missing / registry miss
)

// StoreDisplay is the API-safe projection of a VectorStore for embedding in
// other resource responses (notably KnowledgeBase). It carries only the
// display-safe identifiers — never connection credentials.
//
// Source is one of the StoreSource* constants. EngineType is the underlying
// engine name (e.g. "qdrant"). Status mirrors Source by default but
// is split out to give the UI a stable boolean-like signal independent of
// future Source value additions.
type StoreDisplay struct {
	Name       string `json:"vector_store_name,omitempty"`
	Source     string `json:"vector_store_source,omitempty"`
	EngineType string `json:"vector_store_engine_type,omitempty"`
	Status     string `json:"vector_store_status,omitempty"` // "available" / "unavailable"
}

// DefaultStoreDisplay is the display payload for KBs that fall back to the
// tenant's env stores (VectorStoreID == nil).
func DefaultStoreDisplay() StoreDisplay {
	return StoreDisplay{
		Name:   "System default",
		Source: StoreSourceEnv,
		Status: "available",
	}
}

// UnavailableStoreDisplay is used when the bound store cannot be resolved
// (deleted row, registry miss, transient infra error). The UI can branch on
// Status to guide recovery (admin tool, rebind, etc.).
func UnavailableStoreDisplay() StoreDisplay {
	return StoreDisplay{
		Source: StoreSourceUnavailable,
		Status: "unavailable",
	}
}

// ---------------------------------------------------------------------------
// VectorStoreResponse — API response DTO
// ---------------------------------------------------------------------------

// VectorStoreResponse is the API response DTO for vector store.
// Wraps VectorStore with additional metadata (source, readonly).
type VectorStoreResponse struct {
	VectorStore
	Source   string `json:"source"`   // "env" or "user"
	ReadOnly bool   `json:"readonly"` // env stores are read-only
}

// NewVectorStoreResponse creates a response DTO from a VectorStore
// with sensitive fields masked.
func NewVectorStoreResponse(store *VectorStore, source string, readonly bool) VectorStoreResponse {
	masked := *store
	masked.ConnectionConfig = store.ConnectionConfig.MaskSensitiveFields()
	return VectorStoreResponse{
		VectorStore: masked,
		Source:      source,
		ReadOnly:    readonly,
	}
}

// ---------------------------------------------------------------------------
// VectorStore type metadata — for /types endpoint
// ---------------------------------------------------------------------------

// VectorStoreTypeInfo describes a supported engine type and its configuration schema.
type VectorStoreTypeInfo struct {
	Type             string                 `json:"type"`
	DisplayName      string                 `json:"display_name"`
	ConnectionFields []VectorStoreFieldInfo `json:"connection_fields"`
	IndexFields      []VectorStoreFieldInfo `json:"index_fields,omitempty"`
}

// VectorStoreFieldInfo describes a single configuration field exposed
// by /api/v1/vector-stores/types for the registration UI. The optional
// validation hints (`Immutable`, `Min`, `Max`, `Enum`) are used by both
// the frontend (to disable / constrain inputs) and the backend
// (defense-in-depth validation in the service layer).
type VectorStoreFieldInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "string", "number", "boolean"
	Required    bool   `json:"required"`
	Sensitive   bool   `json:"sensitive,omitempty"`
	Default     any    `json:"default,omitempty"`
	Description string `json:"description,omitempty"`

	// Immutable marks a field whose value cannot be changed after the
	// VectorStore is first created. The UI shows the input as read-only
	// in edit mode; the backend rejects modification attempts. Used by
	// engines whose underlying index structure is fixed at create time.
	Immutable bool `json:"immutable,omitempty"`

	// Min / Max set inclusive bounds for "number"-typed fields. nil
	// means no bound on that side. Frontend uses these to constrain
	// inputs; backend re-validates as defense-in-depth.
	Min *float64 `json:"min,omitempty"`
	Max *float64 `json:"max,omitempty"`

	// Enum constrains the allowed string values. Empty means no
	// constraint. Used by fields whose value space is closed.
	Enum []string `json:"enum,omitempty"`
}

// GetVectorStoreTypes returns metadata for all supported engine types.
func GetVectorStoreTypes() []VectorStoreTypeInfo {
	return []VectorStoreTypeInfo{
		// PostgreSQL is excluded from the type list because it only supports the
		// app's default DB connection (UseDefaultConnection=true). It appears as
		// an env store when configured via RETRIEVE_DRIVER but cannot be added
		// as a DB store.
		{
			Type:        "qdrant",
			DisplayName: "Qdrant",
			ConnectionFields: []VectorStoreFieldInfo{
				{Name: "host", Type: "string", Required: true, Description: "Host", Default: "localhost"},
				{Name: "port", Type: "number", Required: false, Description: "Port", Default: 6334},
				{Name: "api_key", Type: "string", Required: false, Sensitive: true, Description: "API Key"},
				{Name: "use_tls", Type: "boolean", Required: false, Description: "Use TLS", Default: false},
			},
			IndexFields: []VectorStoreFieldInfo{
				{Name: "collection_prefix", Type: "string", Required: false, Description: "Collection Prefix", Default: "weknora_embeddings"},
				{Name: "shard_number", Type: "number", Required: false, Description: "Shard Number", Default: 1},
				{Name: "replication_factor", Type: "number", Required: false, Description: "Replication Factor", Default: 1},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// BuildEnvVectorStores — virtual stores from RETRIEVE_DRIVER env var
// ---------------------------------------------------------------------------

// BuildEnvVectorStores builds virtual VectorStore entries from RETRIEVE_DRIVER.
// Returns []VectorStore (not VectorStoreResponse) so that business logic (e.g.,
// duplicate checking) can use them directly. API responses should wrap them
// via NewVectorStoreResponse.
//
// Pure function — does not call os.Getenv directly.
//
// Usage:
//
//	types.BuildEnvVectorStores(os.Getenv("RETRIEVE_DRIVER"), os.Getenv)
func BuildEnvVectorStores(retrieveDriver string, envLookup EnvLookupFunc) []VectorStore {
	if retrieveDriver == "" {
		return nil
	}

	drivers := strings.Split(retrieveDriver, ",")
	var stores []VectorStore

	for _, driver := range drivers {
		driver = strings.TrimSpace(driver)
		if driver == "" {
			continue
		}

		store := buildEnvStoreForDriver(driver, envLookup)
		if store != nil {
			stores = append(stores, *store)
		}
	}
	return stores
}

// FindEnvVectorStore finds a specific env store by its virtual ID.
func FindEnvVectorStore(retrieveDriver string, envLookup EnvLookupFunc, id string) *VectorStore {
	for _, s := range BuildEnvVectorStores(retrieveDriver, envLookup) {
		if s.ID == id {
			return &s
		}
	}
	return nil
}

func buildEnvStoreForDriver(driver string, envLookup EnvLookupFunc) *VectorStore {
	switch driver {
	case "postgres":
		return &VectorStore{
			ID:         "__env_postgres__",
			Name:       "PostgreSQL",
			EngineType: PostgresRetrieverEngineType,
			ConnectionConfig: ConnectionConfig{
				UseDefaultConnection: true,
			},
		}
	case "qdrant":
		return &VectorStore{
			ID:         "__env_qdrant__",
			Name:       "Qdrant",
			EngineType: QdrantRetrieverEngineType,
			ConnectionConfig: ConnectionConfig{
				Host:   envLookup("QDRANT_HOST"),
				APIKey: envLookup("QDRANT_API_KEY"),
			},
		}
	default:
		return nil
	}
}
