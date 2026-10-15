package types

// ParserEngineConfigForResponse returns a copy with secret fields redacted
// when maskSecrets is true.
func ParserEngineConfigForResponse(cfg *ParserEngineConfig, maskSecrets bool) *ParserEngineConfig {
	if cfg == nil {
		return nil
	}
	out := *cfg
	if !maskSecrets {
		return &out
	}
	if out.MinerUAPIKey != "" {
		out.MinerUAPIKey = RedactedSecretPlaceholder
	}
	if out.MinerUServerAPIKey != "" {
		out.MinerUServerAPIKey = RedactedSecretPlaceholder
	}
	if out.PaddleOCRVLCloudToken != "" {
		out.PaddleOCRVLCloudToken = RedactedSecretPlaceholder
	}
	return &out
}

// StorageEngineConfigForResponse returns a copy with provider secret fields
// redacted when maskSecrets is true.
func StorageEngineConfigForResponse(cfg *StorageEngineConfig, maskSecrets bool) *StorageEngineConfig {
	if cfg == nil {
		return nil
	}
	out := *cfg
	if !maskSecrets {
		return &out
	}
	if out.MinIO != nil {
		minio := *out.MinIO
		if minio.AccessKeyID != "" {
			minio.AccessKeyID = RedactedSecretPlaceholder
		}
		if minio.SecretAccessKey != "" {
			minio.SecretAccessKey = RedactedSecretPlaceholder
		}
		out.MinIO = &minio
	}
	return &out
}

// CredentialsConfigForResponse returns a copy with app_secret redacted when
// maskSecrets is true.
func CredentialsConfigForResponse(cfg *CredentialsConfig, maskSecrets bool) *CredentialsConfig {
	if cfg == nil {
		return nil
	}
	out := *cfg
	if !maskSecrets {
		return &out
	}
	if out.WeKnoraCloud != nil {
		cloud := *out.WeKnoraCloud
		if cloud.AppSecret != "" {
			cloud.AppSecret = RedactedSecretPlaceholder
		}
		out.WeKnoraCloud = &cloud
	}
	return &out
}

// MergeParserEngineConfigForUpdate applies preserve semantics to secret fields
// on tenant KV PUT.
func MergeParserEngineConfigForUpdate(incoming, existing *ParserEngineConfig) *ParserEngineConfig {
	if incoming == nil {
		return nil
	}
	out := *incoming
	var prev ParserEngineConfig
	if existing != nil {
		prev = *existing
	}
	out.MinerUAPIKey = PreserveIfRedacted(out.MinerUAPIKey, prev.MinerUAPIKey)
	out.MinerUServerAPIKey = PreserveIfRedacted(out.MinerUServerAPIKey, prev.MinerUServerAPIKey)
	out.PaddleOCRVLCloudToken = PreserveIfRedacted(out.PaddleOCRVLCloudToken, prev.PaddleOCRVLCloudToken)
	// Chat attachment parser rules are configured per agent; preserve any legacy
	// tenant-level rules when the settings UI omits this field on engine updates.
	if incoming.ChatParserEngineRules == nil && existing != nil {
		out.ChatParserEngineRules = existing.ChatParserEngineRules
	}
	return &out
}

// MergeStorageEngineConfigForUpdate applies preserve semantics to provider
// secret fields on tenant KV PUT.
func MergeStorageEngineConfigForUpdate(incoming, existing *StorageEngineConfig) *StorageEngineConfig {
	if incoming == nil {
		return nil
	}
	out := *incoming
	if out.MinIO != nil {
		minio := *out.MinIO
		var prev MinIOEngineConfig
		if existing != nil && existing.MinIO != nil {
			prev = *existing.MinIO
		}
		minio.AccessKeyID = PreserveIfRedacted(minio.AccessKeyID, prev.AccessKeyID)
		minio.SecretAccessKey = PreserveIfRedacted(minio.SecretAccessKey, prev.SecretAccessKey)
		out.MinIO = &minio
	}
	return &out
}
