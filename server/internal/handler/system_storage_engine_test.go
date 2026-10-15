package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/storageallowlist"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeStorageBackendRepo is a minimal StorageBackendRepository used to exercise
// GetStorageEngineStatus' multi-instance awareness without a database.
type fakeStorageBackendRepo struct{ backends []*types.StorageBackend }

func (f *fakeStorageBackendRepo) Create(context.Context, *types.StorageBackend) error {
	return nil
}
func (f *fakeStorageBackendRepo) GetByID(context.Context, uint64, string) (*types.StorageBackend, error) {
	return nil, nil
}
func (f *fakeStorageBackendRepo) List(context.Context, uint64) ([]*types.StorageBackend, error) {
	return f.backends, nil
}
func (f *fakeStorageBackendRepo) Update(context.Context, *types.StorageBackend) error { return nil }
func (f *fakeStorageBackendRepo) Delete(context.Context, uint64, string) error        { return nil }
func (f *fakeStorageBackendRepo) FindLegacyAlias(context.Context, uint64, string) (*types.StorageBackend, error) {
	return nil, nil
}

func TestGetStorageEngineStatus_IncludesMinio(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv(storageallowlist.AllowListEnv, "")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/system/storage-engine-status", nil)

	h := &SystemHandler{}
	h.GetStorageEngineStatus(c)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Engines []StorageEngineStatusItem `json:"engines"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)

	names := make([]string, 0, len(resp.Data.Engines))
	minioStatus := StorageEngineStatusItem{}
	for _, engine := range resp.Data.Engines {
		names = append(names, engine.Name)
		if engine.Name == "minio" {
			minioStatus = engine
		}
	}
	assert.Contains(t, names, "local")
	assert.Contains(t, names, "minio")
	assert.True(t, minioStatus.Allowed)
	assert.False(t, minioStatus.Available)
}

func TestGetStorageEngineStatus_MinioConfiguredFromTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv(storageallowlist.AllowListEnv, "")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/system/storage-engine-status", nil)
	tenant := &types.Tenant{
		StorageEngineConfig: &types.StorageEngineConfig{
			MinIO: &types.MinIOEngineConfig{
				Mode:            "remote",
				Endpoint:        "minio.example.com:9000",
				AccessKeyID:     "ak",
				SecretAccessKey: "sk",
				BucketName:      "bucket",
			},
		},
	}
	c.Set(types.TenantInfoContextKey.String(), tenant)

	h := &SystemHandler{}
	h.GetStorageEngineStatus(c)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data struct {
			Engines []StorageEngineStatusItem `json:"engines"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	var minioStatus *StorageEngineStatusItem
	for i := range resp.Data.Engines {
		if resp.Data.Engines[i].Name == "minio" {
			minioStatus = &resp.Data.Engines[i]
			break
		}
	}
	require.NotNil(t, minioStatus)
	assert.True(t, minioStatus.Available)
}

// A workspace that configured MinIO only through the new multi-instance Storage
// settings (storage_backends), with an empty legacy StorageEngineConfig, must
// still be reported as available; a disabled backend must not.
func TestGetStorageEngineStatus_AvailableFromActiveBackend(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv(storageallowlist.AllowListEnv, "")

	run := func(t *testing.T, backends []*types.StorageBackend) map[string]bool {
		t.Helper()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/system/storage-engine-status", nil)
		c.Set(types.TenantInfoContextKey.String(), &types.Tenant{ID: 42})

		h := &SystemHandler{storageBackendRepo: &fakeStorageBackendRepo{backends: backends}}
		h.GetStorageEngineStatus(c)

		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data struct {
				Engines []StorageEngineStatusItem `json:"engines"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

		status := map[string]bool{}
		for _, engine := range resp.Data.Engines {
			status[engine.Name] = engine.Available
		}
		return status
	}

	t.Run("active backend is available", func(t *testing.T) {
		status := run(t, []*types.StorageBackend{
			{Provider: "minio", Status: types.StorageBackendStatusActive},
		})
		assert.True(t, status["minio"], "active MinIO backend should be available")
	})

	t.Run("disabled backend is not available", func(t *testing.T) {
		status := run(t, []*types.StorageBackend{
			{Provider: "minio", Status: types.StorageBackendStatusDisabled},
		})
		assert.False(t, status["minio"], "disabled MinIO backend should not be available")
	})
}
