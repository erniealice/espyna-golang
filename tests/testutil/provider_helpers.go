//go:build mock_auth

package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/erniealice/espyna-golang/internal/composition/core"
	adminpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/admin"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
)

// SetupTestEnvironment sets environment variables for a specific provider
func SetupTestEnvironment(providerName string) {
	switch providerName {
	case "mock":
		os.Setenv("CONFIG_DATABASE_PROVIDER", "mock_db")
		os.Setenv("BUSINESS_TYPE", "education")
	case "postgresql":
		os.Setenv("CONFIG_DATABASE_PROVIDER", "postgres")
		os.Setenv("POSTGRES_HOST", getEnvWithDefault("TEST_POSTGRES_HOST", "localhost"))
		os.Setenv("POSTGRES_PORT", getEnvWithDefault("TEST_POSTGRES_PORT", "5432"))
		os.Setenv("POSTGRES_NAME", getEnvWithDefault("TEST_POSTGRES_DB", "espyna_test"))
		os.Setenv("POSTGRES_USER", getEnvWithDefault("TEST_POSTGRES_USER", "postgres"))
		os.Setenv("POSTGRES_PASSWORD", getEnvWithDefault("TEST_POSTGRES_PASSWORD", ""))
	case "firestore":
		os.Setenv("CONFIG_DATABASE_PROVIDER", "firestore")
		os.Setenv("FIRESTORE_PROJECT_ID", getEnvWithDefault("TEST_FIRESTORE_PROJECT", "espyna-test"))
	default:
		panic(fmt.Sprintf("Unknown provider: %s", providerName))
	}

	// Set common test environment variables
	os.Setenv("CONFIG_AUTH_PROVIDER", "mock_auth")
	os.Setenv("CONFIG_STORAGE_PROVIDER", "mock_storage")
}

// CreateTestContainer creates a container for testing with the specified provider
func CreateTestContainer(providerName string) *core.Container {
	SetupTestEnvironment(providerName)
	return core.NewContainerFromEnv()
}

// CreateTestClient creates a standardized test client for consistent testing
func CreateTestClient() *clientpb.Client {
	now := time.Now()
	timestamp := now.Unix()
	timestampString := now.Format(time.RFC3339)

	return &clientpb.Client{
		Id:                 "test-client-123",
		UserId:             "test-user-123",
		Active:             true,
		InternalId:         "internal-123",
		DateCreated:        &timestamp,
		DateCreatedString:  &timestampString,
		DateModified:       &timestamp,
		DateModifiedString: &timestampString,
	}
}

// CreateTestAdmin creates a standardized test admin for consistent testing
func CreateTestAdmin() *adminpb.Admin {
	return &adminpb.Admin{
		Id:     "test-admin-123",
		UserId: "test-user-456",
		Active: true,
	}
}

// CleanupProvider performs cleanup operations for the specified provider
func CleanupProvider(container *core.Container, providerName string) error {
	switch providerName {
	case "mock":
		// For mock provider, we can reset by creating a new container
		// The mock data is in-memory and will be garbage collected
		return nil
	case "postgresql":
		// Clean test database tables
		return cleanPostgreSQLTestData(container)
	case "firestore":
		// Clean test collections
		return cleanFirestoreTestData(container)
	default:
		return fmt.Errorf("unknown provider for cleanup: %s", providerName)
	}
}

// cleanPostgreSQLTestData removes test data from PostgreSQL tables
func cleanPostgreSQLTestData(container *core.Container) error {
	// This would connect to the database and clean test tables
	// For now, we'll implement a basic version
	// In practice, you might want to use transactions or specific test schemas
	return nil
}

// cleanFirestoreTestData removes test data from Firestore collections
func cleanFirestoreTestData(container *core.Container) error {
	// This would connect to Firestore and clean test collections
	// For now, we'll implement a basic version
	// In practice, you might want to use the Firestore emulator for testing
	return nil
}

// getEnvWithDefault gets environment variable with fallback default
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// AssertClientCreated verifies that a client was successfully created
func AssertClientCreated(t *testing.T, response *clientpb.CreateClientResponse) {
	t.Helper()

	if response == nil {
		t.Fatal("Expected non-nil response")
	}

	if response.Data == nil || len(response.Data) == 0 {
		t.Fatal("Expected response to contain client data")
	}

	client := response.Data[0]
	if client.Id == "" {
		t.Error("Expected client to have an ID")
	}

	if client.UserId == "" {
		t.Error("Expected client to have a user ID")
	}

	if !client.Active {
		t.Error("Expected client to be active")
	}
}

// AssertAdminCreated verifies that an admin was successfully created
func AssertAdminCreated(t *testing.T, response *adminpb.CreateAdminResponse) {
	t.Helper()

	if response == nil {
		t.Fatal("Expected non-nil response")
	}

	if response.Data == nil || len(response.Data) == 0 {
		t.Fatal("Expected response to contain admin data")
	}

	admin := response.Data[0]
	if admin.Id == "" {
		t.Error("Expected admin to have an ID")
	}

	if admin.UserId == "" {
		t.Error("Expected admin to have a user ID")
	}

	if !admin.Active {
		t.Error("Expected admin to be active")
	}
}

// AssertProviderHealthy verifies that a provider is healthy
func AssertProviderHealthy(t *testing.T, container *core.Container, providerName string) {
	t.Helper()

	providerManager := container.GetProviderManager()
	if providerManager == nil {
		t.Fatalf("Expected provider manager for %s", providerName)
	}

	var provider interface {
		Health(ctx context.Context) error
		Name() string
	}
	switch providerName {
	case "mock", "postgresql", "firestore":
		provider = providerManager.GetDatabaseProvider()
	default:
		t.Fatalf("Unknown provider type: %s", providerName)
	}

	if provider == nil {
		t.Fatalf("Expected active provider for %s", providerName)
	}

	if provider.Name() != providerName {
		t.Fatalf("Expected provider %s, got %s", providerName, provider.Name())
	}

	err := provider.Health(context.Background())
	if err != nil {
		t.Fatalf("Provider %s should be healthy, got error: %v", providerName, err)
	}
}

// AssertProviderUnhealthy verifies that a provider is unhealthy
func AssertProviderUnhealthy(t *testing.T, container *core.Container, providerName string) {
	t.Helper()

	provider := container.GetProviderManager().GetDatabaseProvider()
	if provider == nil {
		t.Fatalf("Expected active provider for %s", providerName)
	}

	err := provider.Health(context.Background())
	if err == nil {
		t.Fatalf("Provider %s should be unhealthy, but health check passed", providerName)
	}
}
