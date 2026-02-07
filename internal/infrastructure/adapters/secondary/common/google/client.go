//go:build google

package google

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/common/gcp"
)

// GoogleClientManager manages Google Cloud clients
//
// This replaces the previous singleton pattern with explicit dependency injection,
// making the code more testable and easier to reason about.
type GoogleClientManager struct {
	storageClient *storage.Client
	config        *gcp.CredentialConfig
}

// GoogleConfig holds Google-specific configuration
type GoogleConfig struct {
	StorageTimeout time.Duration
}

// DefaultGoogleConfig returns default Google configuration from environment
func DefaultGoogleConfig() *GoogleConfig {
	timeout := 30 * time.Second
	if timeoutStr := os.Getenv("GOOGLE_STORAGE_TIMEOUT"); timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		}
	}

	return &GoogleConfig{
		StorageTimeout: timeout,
	}
}

// NewGoogleClientManager creates a new Google client manager
//
// This initializes all Google Cloud clients needed for the application.
// It uses the shared gcp.CredentialConfig for authentication.
func NewGoogleClientManager(ctx context.Context, config *GoogleConfig) (*GoogleClientManager, error) {
	if config == nil {
		config = DefaultGoogleConfig()
	}

	// Get credential configuration using shared package
	credConfig := gcp.DefaultCredentialConfig("GOOGLE_")

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

	return &GoogleClientManager{
		storageClient: storageClient,
		config:        credConfig,
	}, nil
}

// GetStorageClient returns the Google Cloud Storage client
func (m *GoogleClientManager) GetStorageClient() *storage.Client {
	return m.storageClient
}

// GetProjectID returns the GCP project ID
func (m *GoogleClientManager) GetProjectID() string {
	return m.config.ProjectID
}

// Close closes all Google Cloud clients
func (m *GoogleClientManager) Close() error {
	if m.storageClient != nil {
		return m.storageClient.Close()
	}
	return nil
}