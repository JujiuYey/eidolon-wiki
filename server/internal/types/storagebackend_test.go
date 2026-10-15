package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageBackendPathRoundTrip(t *testing.T) {
	path := BuildStorageBackendPath("backend-a", "minio://weknora/data/file.pdf")
	id, inner, ok := ParseStorageBackendPath(path)
	require.True(t, ok)
	assert.Equal(t, "backend-a", id)
	assert.Equal(t, "minio://weknora/data/file.pdf", inner)
	assert.Equal(t, "minio", ParseProviderScheme(path))
}

func TestSharesStorageBackendWithUsesConcreteInstance(t *testing.T) {
	aID, bID := "cos-a", "cos-b"
	a := &KnowledgeBase{StorageBackendID: &aID, StorageProviderConfig: &StorageProviderConfig{Provider: "minio"}}
	b := &KnowledgeBase{StorageBackendID: &bID, StorageProviderConfig: &StorageProviderConfig{Provider: "minio"}}
	assert.False(t, a.SharesStorageBackendWith(b, "", "minio"))

	b.StorageBackendID = &aID
	assert.True(t, a.SharesStorageBackendWith(b, "", "minio"))
}

func TestNewStorageBackendResponseMasksCredentials(t *testing.T) {
	backend := &StorageBackend{Config: StorageBackendConfig{AccessKeyID: "id", SecretAccessKey: "secret"}}
	response := NewStorageBackendResponse(backend)
	assert.Equal(t, RedactedSecretPlaceholder, response.Config.AccessKeyID)
	assert.Equal(t, RedactedSecretPlaceholder, response.Config.SecretAccessKey)
	assert.Equal(t, "id", backend.Config.AccessKeyID)
}

func TestStorageBackendFromEnvironment(t *testing.T) {
	t.Setenv("STORAGE_TYPE", "minio")
	t.Setenv("MINIO_ENDPOINT", "minio.example.com:9000")
	t.Setenv("MINIO_ACCESS_KEY_ID", "access")
	t.Setenv("MINIO_SECRET_ACCESS_KEY", "secret")
	t.Setenv("MINIO_BUCKET_NAME", "bucket")

	backend := StorageBackendFromEnvironment(42)
	require.NotNil(t, backend)
	assert.Equal(t, uint64(42), backend.TenantID)
	assert.Equal(t, "minio", backend.Provider)
	assert.Equal(t, StorageBackendSourceEnv, backend.Source)
	assert.True(t, backend.LegacyAlias)
	assert.Equal(t, "bucket", backend.Config.BucketName)
	assert.Equal(t, "minio.example.com:9000", backend.Config.Endpoint)
}

// The cloud providers this build no longer ships must fall through to nil,
// so an env left over from a fuller deployment cannot resurrect them.
func TestStorageBackendFromEnvironmentRejectsRemovedProviders(t *testing.T) {
	for _, provider := range []string{"s3", "aws-s3", "cos", "obs", "tos", "oss", "ks3"} {
		t.Run(provider, func(t *testing.T) {
			t.Setenv("STORAGE_TYPE", provider)
			assert.Nil(t, StorageBackendFromEnvironment(42))
		})
	}
}

func TestStorageBackendRejectsTraversingPathPrefix(t *testing.T) {
	backend := &StorageBackend{
		TenantID: 1, Name: "unsafe", Provider: "local",
		Config: StorageBackendConfig{PathPrefix: "../outside"},
	}
	require.Error(t, backend.Validate())
}
