package infrastructure

import (
	"fmt"

	pb "leapfor.xyz/esqyma/golang/v1/infrastructure/storage"
)

// StorageConfigAdapter provides helpers to convert between map[string]any config
// and proto StorageProviderConfig
//
// This adapter serves as the bridge between the application configuration layer
// (which loads from environment variables/files as map[string]any) and the
// storage providers (which use strongly-typed proto contracts).
//
// Benefits of this approach:
// - Configuration remains flexible (can load from various sources)
// - Storage providers get type-safe proto configs
// - Clear separation between config loading and business logic
// - Easy to validate and transform configuration
type StorageConfigAdapter struct{}

// NewStorageConfigAdapter creates a new config adapter
func NewStorageConfigAdapter() *StorageConfigAdapter {
	return &StorageConfigAdapter{}
}

// ConvertMapToProtoConfig converts map[string]any config to proto config
// This allows ProviderManager to pass map configs while adapters use proto types
func (a *StorageConfigAdapter) ConvertMapToProtoConfig(providerName string, config map[string]any) (*pb.StorageProviderConfig, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	protoConfig := &pb.StorageProviderConfig{
		Enabled:     getBool(config, "enabled", true),
		DisplayName: getString(config, "display_name", providerName),
	}

	// Determine provider type and extract specific config
	switch providerName {
	case "local":
		protoConfig.Provider = pb.StorageProvider_STORAGE_PROVIDER_LOCAL
		protoConfig.Config = &pb.StorageProviderConfig_LocalConfig{
			LocalConfig: &pb.LocalStorageConfig{
				BaseDirectory:         getString(config, "base_directory", "./storage"),
				AutoCreateDirectories: getBool(config, "auto_create_directories", true),
				FilePermissions:       getString(config, "file_permissions", "0644"),
				DirectoryPermissions:  getString(config, "directory_permissions", "0755"),
			},
		}

	case "gcs":
		protoConfig.Provider = pb.StorageProvider_STORAGE_PROVIDER_GCP
		protoConfig.Config = &pb.StorageProviderConfig_GcsConfig{
			GcsConfig: &pb.GcsStorageConfig{
				ProjectId:                getString(config, "project_id", ""),
				ServiceAccountKey:        getString(config, "service_account_key", ""),
				ServiceAccountEmail:      getString(config, "service_account_email", ""),
				UseAdc:                   getBool(config, "use_adc", false),
				WorkloadIdentityProvider: getString(config, "workload_identity_provider", ""),
				DefaultBucket:            getString(config, "default_bucket", ""),
				EndpointUrl:              getString(config, "endpoint_url", ""),
				DefaultKmsKeyName:        getString(config, "default_kms_key_name", ""),
				UniformBucketLevelAccess: getBool(config, "uniform_bucket_level_access", false),
			},
		}

	case "s3":
		protoConfig.Provider = pb.StorageProvider_STORAGE_PROVIDER_AWS
		protoConfig.Config = &pb.StorageProviderConfig_S3Config{
			S3Config: &pb.S3StorageConfig{
				Region:                     getString(config, "region", "us-east-1"),
				AccessKeyId:                getString(config, "access_key_id", ""),
				SecretAccessKey:            getString(config, "secret_access_key", ""),
				SessionToken:               getString(config, "session_token", ""),
				UseIamRole:                 getBool(config, "use_iam_role", false),
				IamRoleArn:                 getString(config, "iam_role_arn", ""),
				DefaultBucket:              getString(config, "default_bucket", ""),
				EndpointUrl:                getString(config, "endpoint_url", ""),
				UsePathStyle:               getBool(config, "use_path_style", false),
				DefaultKmsKeyId:            getString(config, "default_kms_key_id", ""),
				DefaultSseAlgorithm:        getString(config, "default_sse_algorithm", ""),
				EnableTransferAcceleration: getBool(config, "enable_transfer_acceleration", false),
				UseDualStack:               getBool(config, "use_dual_stack", false),
			},
		}

	case "azure":
		protoConfig.Provider = pb.StorageProvider_STORAGE_PROVIDER_AZURE
		protoConfig.Config = &pb.StorageProviderConfig_AzureConfig{
			AzureConfig: &pb.AzureStorageConfig{
				StorageAccount:          getString(config, "storage_account", ""),
				AccountKey:              getString(config, "account_key", ""),
				ConnectionString:        getString(config, "connection_string", ""),
				UseManagedIdentity:      getBool(config, "use_managed_identity", false),
				ManagedIdentityClientId: getString(config, "managed_identity_client_id", ""),
				TenantId:                getString(config, "tenant_id", ""),
				DefaultContainer:        getString(config, "default_container", ""),
				EndpointUrl:             getString(config, "endpoint_url", ""),
				DefaultEncryptionScope:  getString(config, "default_encryption_scope", ""),
			},
		}

	default:
		return nil, fmt.Errorf("unknown storage provider: %s", providerName)
	}

	return protoConfig, nil
}

// Helper functions getString, getBool, getInt32 are defined in helpers.go
