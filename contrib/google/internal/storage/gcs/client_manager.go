package gcs

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"github.com/erniealice/espyna-golang/contrib/google/internal/common/gcp"
)

// GCSClientManager manages Google Cloud Storage clients
//
// This replaces the previous singleton pattern with explicit dependency injection,
// making the code more testable and easier to reason about.
type GCSClientManager struct {
	storageClient *storage.Client
	config        *gcp.CredentialConfig
}

// GCSClientConfig holds GCS-specific configuration
type GCSClientConfig struct {
	StorageTimeout time.Duration
}

// DefaultGCSConfig returns default GCS configuration from environment
func DefaultGCSConfig() *GCSClientConfig {
	timeout := 30 * time.Second
	if timeoutStr := os.Getenv("STORAGE_GCS_TIMEOUT"); timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		}
	}

	return &GCSClientConfig{
		StorageTimeout: timeout,
	}
}

// NewGCSClientManager creates a new GCS client manager
//
// This initializes the Google Cloud Storage client needed for the application.
// It uses the shared gcp.CredentialConfig for authentication.
func NewGCSClientManager(ctx context.Context, config *GCSClientConfig) (*GCSClientManager, error) {
	if config == nil {
		config = DefaultGCSConfig()
	}

	// Get credential configuration using shared package (STORAGE/gcs concern).
	credConfig := gcp.DefaultCredentialConfig("STORAGE_GCS_")

	// Validate credential config
	if err := credConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid credential config: %w", err)
	}

	// Create context with timeout for storage client initialization
	storageCtx, cancel := context.WithTimeout(ctx, config.StorageTimeout)
	defer cancel()

	// Get client option from shared package
	opt, err := gcp.GetClientOption(credConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get client option: %w", err)
	}

	// Create storage client
	var storageClient *storage.Client
	if opt != nil {
		storageClient, err = storage.NewClient(storageCtx, opt)
	} else {
		// Use Application Default Credentials
		storageClient, err = storage.NewClient(storageCtx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	log.Println("✅ Google Cloud Storage client initialized successfully")

	return &GCSClientManager{
		storageClient: storageClient,
		config:        credConfig,
	}, nil
}

// GetStorageClient returns the Google Cloud Storage client
func (m *GCSClientManager) GetStorageClient() *storage.Client {
	return m.storageClient
}

// GetProjectID returns the GCP project ID
func (m *GCSClientManager) GetProjectID() string {
	return m.config.ProjectID
}

// Close closes all Google Cloud clients
func (m *GCSClientManager) Close() error {
	if m.storageClient != nil {
		return m.storageClient.Close()
	}
	return nil
}
