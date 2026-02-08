//go:build mock_db

package mock

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	dbpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/database"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterDatabaseProvider(
		"mock",
		func() ports.DatabaseProvider {
			return NewMockAdapter()
		},
		transformConfig,
	)
	registry.RegisterDatabaseBuildFromEnv("mock", buildFromEnv)
	// Mock adapter uses default table names - no need for custom env vars
	registry.RegisterDatabaseTableConfigBuilder("mock", registry.DefaultDatabaseTableConfig)
}

// buildFromEnv creates and initializes a Mock adapter from environment variables.
func buildFromEnv() (ports.DatabaseProvider, error) {
	businessType := os.Getenv("BUSINESS_TYPE")
	if businessType == "" {
		businessType = "education"
	}

	protoConfig := &dbpb.DatabaseProviderConfig{
		Provider: dbpb.DatabaseProvider_DATABASE_PROVIDER_MOCK,
		Enabled:  true,
		Config: &dbpb.DatabaseProviderConfig_Mock{
			Mock: &dbpb.MockConfig{
				Name: businessType,
			},
		},
	}

	adapter := NewMockAdapter()
	if err := adapter.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("mock: failed to initialize: %w", err)
	}
	return adapter, nil
}

// transformConfig converts raw config map to Mock proto config.
func transformConfig(rawConfig map[string]any) (*dbpb.DatabaseProviderConfig, error) {
	protoConfig := &dbpb.DatabaseProviderConfig{
		Provider: dbpb.DatabaseProvider_DATABASE_PROVIDER_MOCK,
		Enabled:  true,
	}

	mockConfig := &dbpb.MockConfig{}

	if businessType, ok := rawConfig["default_business_type"].(string); ok {
		mockConfig.Name = businessType
	}

	protoConfig.Config = &dbpb.DatabaseProviderConfig_Mock{
		Mock: mockConfig,
	}

	return protoConfig, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// MockAdapter implements DatabaseProvider and RepositoryProvider for testing.
// This adapter follows the same self-registration pattern as Firestore/Postgres.
type MockAdapter struct {
	businessType     string
	enabled          bool
	simulateFailures bool
	mockData         map[string]any
}

// NewMockAdapter creates a new Mock database adapter.
func NewMockAdapter() *MockAdapter {
	return &MockAdapter{
		businessType: "education",
		enabled:      true,
		mockData:     make(map[string]any),
	}
}

// Name returns the provider name.
func (a *MockAdapter) Name() string {
	return "mock"
}

// Initialize sets up the Mock adapter with configuration.
func (a *MockAdapter) Initialize(config *dbpb.DatabaseProviderConfig) error {
	mockProto := config.GetMock()

	if mockProto != nil {
		if mockProto.Name != "" {
			a.businessType = mockProto.Name
		}
		a.simulateFailures = mockProto.SimulateFailures
	}

	a.enabled = config.Enabled
	log.Printf("✅ Mock adapter initialized (businessType: %s)", a.businessType)
	return nil
}

// GetConnection returns the businessType as the "connection" for mock repositories.
func (a *MockAdapter) GetConnection() any {
	return a.businessType
}

// Close cleans up mock adapter resources.
func (a *MockAdapter) Close() error {
	a.mockData = make(map[string]any)
	log.Println("✅ Mock adapter closed")
	return nil
}

// IsHealthy checks if the mock adapter is healthy.
func (a *MockAdapter) IsHealthy(ctx context.Context) error {
	if !a.enabled {
		return fmt.Errorf("mock adapter is disabled")
	}
	if a.simulateFailures {
		return fmt.Errorf("simulated health check failure")
	}
	return nil
}

// IsEnabled returns whether this adapter is currently enabled.
func (a *MockAdapter) IsEnabled() bool {
	return a.enabled
}

// =============================================================================
// RepositoryProvider Implementation - Delegates to Registry
// =============================================================================

// CreateRepository creates a repository by looking up the registered factory.
// This replaces the giant switch statement by delegating to self-registered factories.
func (a *MockAdapter) CreateRepository(entityName string, conn any, tableName string) (any, error) {
	return registry.CreateRepository("mock", entityName, conn, tableName)
}

// GetTransactionManager returns the Mock transaction manager.
func (a *MockAdapter) GetTransactionManager() interfaces.TransactionManager {
	return core.NewMockTransactionManager()
}

// HealthCheck checks if the Mock adapter is healthy.
func (a *MockAdapter) HealthCheck(ctx context.Context) error {
	return a.IsHealthy(ctx)
}

// =============================================================================
// Re-exports from core package for backward compatibility
// =============================================================================

// NewMockTransactionService creates a transaction service using infrastructure mock.
// This re-exports the core package function for backward compatibility with existing imports.
func NewMockTransactionService(supportsTransactions bool) ports.TransactionService {
	return core.NewMockTransactionService(supportsTransactions)
}

// NewFailingMockTransactionService creates a transaction service that will fail RunInTransaction.
// This re-exports the core package function for backward compatibility with existing imports.
func NewFailingMockTransactionService() ports.TransactionService {
	return core.NewFailingMockTransactionService()
}

// Compile-time interface checks
var _ ports.DatabaseProvider = (*MockAdapter)(nil)
var _ ports.RepositoryProvider = (*MockAdapter)(nil)
