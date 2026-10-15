package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeParserEngineConfigForUpdate_PreservesRedactedSecrets(t *testing.T) {
	existing := &ParserEngineConfig{
		MinerUAPIKey:          "mineru-secret",
		MinerUServerAPIKey:    "mineru-server-secret",
		PaddleOCRVLCloudToken: "paddle-secret",
		MinerUEndpoint:        "http://mineru",
	}
	incoming := &ParserEngineConfig{
		MinerUAPIKey:          RedactedSecretPlaceholder,
		MinerUServerAPIKey:    RedactedSecretPlaceholder,
		PaddleOCRVLCloudToken: RedactedSecretPlaceholder,
		MinerUEndpoint:        "http://mineru-new",
	}
	merged := MergeParserEngineConfigForUpdate(incoming, existing)
	require.NotNil(t, merged)
	assert.Equal(t, "mineru-secret", merged.MinerUAPIKey)
	assert.Equal(t, "mineru-server-secret", merged.MinerUServerAPIKey)
	assert.Equal(t, "paddle-secret", merged.PaddleOCRVLCloudToken)
	assert.Equal(t, "http://mineru-new", merged.MinerUEndpoint)
}

func TestParserEngineConfigForResponse_MasksMinerUSecrets(t *testing.T) {
	cfg := &ParserEngineConfig{
		MinerUAPIKey:       "mineru-secret",
		MinerUServerAPIKey: "mineru-server-secret",
		MinerUEndpoint:     "http://mineru",
	}
	resp := ParserEngineConfigForResponse(cfg, true)
	require.NotNil(t, resp)
	assert.Equal(t, RedactedSecretPlaceholder, resp.MinerUAPIKey)
	assert.Equal(t, RedactedSecretPlaceholder, resp.MinerUServerAPIKey)
	assert.Equal(t, "http://mineru", resp.MinerUEndpoint)
}

func TestMergeParserEngineConfigForUpdate_PreservesLegacyChatParserRules(t *testing.T) {
	existing := &ParserEngineConfig{
		ChatParserEngineRules: []ParserEngineRule{
			{FileTypes: []string{"pdf"}, Engine: "mineru"},
		},
	}
	incoming := &ParserEngineConfig{
		MinerUEndpoint: "http://mineru-new",
	}
	merged := MergeParserEngineConfigForUpdate(incoming, existing)
	require.NotNil(t, merged)
	require.Len(t, merged.ChatParserEngineRules, 1)
	assert.Equal(t, "mineru", merged.ChatParserEngineRules[0].Engine)
}

func TestMergeStorageEngineConfigForUpdate_PreservesRedactedSecrets(t *testing.T) {
	existing := &StorageEngineConfig{
		DefaultProvider: "minio",
		MinIO: &MinIOEngineConfig{
			AccessKeyID:     "access-id",
			SecretAccessKey: "secret-key",
			BucketName:      "bucket",
		},
	}
	incoming := &StorageEngineConfig{
		DefaultProvider: "minio",
		MinIO: &MinIOEngineConfig{
			AccessKeyID:     RedactedSecretPlaceholder,
			SecretAccessKey: RedactedSecretPlaceholder,
			BucketName:      "bucket-new",
		},
	}
	merged := MergeStorageEngineConfigForUpdate(incoming, existing)
	require.NotNil(t, merged)
	require.NotNil(t, merged.MinIO)
	assert.Equal(t, "access-id", merged.MinIO.AccessKeyID)
	assert.Equal(t, "secret-key", merged.MinIO.SecretAccessKey)
	assert.Equal(t, "bucket-new", merged.MinIO.BucketName)
}

func TestParserEngineConfigForResponse_NilSafe(t *testing.T) {
	assert.Nil(t, ParserEngineConfigForResponse(nil, true))
}

func TestStorageEngineConfigForResponse_NilSafe(t *testing.T) {
	assert.Nil(t, StorageEngineConfigForResponse(nil, true))
}
