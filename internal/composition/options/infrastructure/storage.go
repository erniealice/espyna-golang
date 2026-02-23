package infrastructure

import (
	"fmt"
	"strings"
)

// =============================================================================
// STORAGE CONFIGURATION TYPES
// =============================================================================

// StorageConfig is a union type that can hold any storage configuration
type StorageConfig struct {
	Local *LocalStorageConfig
	GCS   *GCSConfig
	S3    *S3Config
	Mock  bool
}

// LocalStorageConfig defines configuration for local file storage
type LocalStorageConfig struct {
	BasePath string `json:"base_path"`
}

// Validate validates the local storage configuration
func (c LocalStorageConfig) Validate() error {
	if c.BasePath == "" {
		c.BasePath = "./storage"
	}
	return nil
}

// GCSConfig defines configuration for Google Cloud Storage
type GCSConfig struct {
	BucketName      string `json:"bucket_name"`
	CredentialsPath string `json:"credentials_path,omitempty"`
	ProjectID       string `json:"project_id"`
}

// Validate validates the GCS configuration
func (c GCSConfig) Validate() error {
	if c.BucketName == "" {
		return fmt.Errorf("GCS bucket name is required")
	}
	if c.ProjectID == "" {
		return fmt.Errorf("GCS project ID is required")
	}
	return nil
}

// S3Config defines configuration for AWS S3 storage
type S3Config struct {
	BucketName string `json:"bucket_name"`
	Region     string `json:"region"`
	AccessKey  string `json:"access_key,omitempty"`
	SecretKey  string `json:"secret_key,omitempty"`
}

// Validate validates the S3 configuration
func (c S3Config) Validate() error {
	if c.BucketName == "" {
		return fmt.Errorf("S3 bucket name is required")
	}
	if c.Region == "" {
		c.Region = "us-east-1"
	}
	return nil
}

// =============================================================================
// ENVIRONMENT CONFIGURATION LOADERS
// =============================================================================

func createLocalStorageConfigFromEnv() LocalStorageConfig {
	return LocalStorageConfig{
		BasePath: GetEnv("LOCAL_STORAGE_PATH", "./storage"),
	}
}

func createGCSConfigFromEnv() GCSConfig {
	return GCSConfig{
		BucketName:      GetEnv("GOOGLE_CLOUD_STORAGE_BUCKET_NAME", ""),
		CredentialsPath: GetEnv("GOOGLE_APPLICATION_CREDENTIALS", ""),
		ProjectID:       GetEnv("GOOGLE_CLOUD_PROJECT_ID", ""),
	}
}

func createS3ConfigFromEnv() S3Config {
	return S3Config{
		BucketName: GetEnv("S3_BUCKET_NAME", ""),
		Region:     GetEnv("S3_REGION", "us-east-1"),
		AccessKey:  GetEnv("S3_ACCESS_KEY", ""),
		SecretKey:  GetEnv("S3_SECRET_KEY", ""),
	}
}

// =============================================================================
// STORAGE PROVIDER OPTIONS
// =============================================================================

// WithStorageFromEnv dynamically selects storage provider based on CONFIG_STORAGE_PROVIDER
func WithStorageFromEnv() ContainerOption {
	return func(c Container) error {
		storageProvider := strings.ToLower(GetEnv("CONFIG_STORAGE_PROVIDER", "local_storage"))

		switch storageProvider {
		case "gcp_storage":
			return WithGoogleCloudStorage(createGCSConfigFromEnv())(c)
		case "s3":
			return WithS3Storage(createS3ConfigFromEnv())(c)
		case "local_storage", "":
			return WithLocalStorage(createLocalStorageConfigFromEnv())(c)
		default:
			return fmt.Errorf("unsupported storage provider: %s", storageProvider)
		}
	}
}

// WithLocalStorage configures local file storage
func WithLocalStorage(config LocalStorageConfig) ContainerOption {
	return func(c Container) error {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid local storage configuration: %w", err)
		}

		if setter, ok := c.(StorageConfigSetter); ok {
			setter.SetStorageConfig(StorageConfig{Local: &config})
		} else {
			return fmt.Errorf("container does not implement SetStorageConfig method")
		}

		fmt.Printf("📁 Configured local storage: %s\n", config.BasePath)
		return nil
	}
}

// WithMockStorage configures mock storage for testing/development
func WithMockStorage() ContainerOption {
	return func(c Container) error {
		if setter, ok := c.(StorageConfigSetter); ok {
			setter.SetStorageConfig(StorageConfig{Mock: true})
		} else {
			return fmt.Errorf("container does not implement SetStorageConfig method")
		}

		fmt.Printf("🧪 Configured mock storage\n")
		return nil
	}
}

// WithGoogleCloudStorage configures Google Cloud Storage
func WithGoogleCloudStorage(config GCSConfig) ContainerOption {
	return func(c Container) error {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid GCS configuration: %w", err)
		}

		if setter, ok := c.(StorageConfigSetter); ok {
			setter.SetStorageConfig(StorageConfig{GCS: &config})
		} else {
			return fmt.Errorf("container does not implement SetStorageConfig method")
		}

		fmt.Printf("☁️ Configured Google Cloud Storage: %s\n", config.BucketName)
		return nil
	}
}

// WithS3Storage configures AWS S3 storage
func WithS3Storage(config S3Config) ContainerOption {
	return func(c Container) error {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid S3 configuration: %w", err)
		}

		if setter, ok := c.(StorageConfigSetter); ok {
			setter.SetStorageConfig(StorageConfig{S3: &config})
		} else {
			return fmt.Errorf("container does not implement SetStorageConfig method")
		}

		fmt.Printf("📦 Configured S3 storage: %s\n", config.BucketName)
		return nil
	}
}
