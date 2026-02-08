package infrastructure

import (
	"fmt"
	"os"
	"strings"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// CreateDatabaseProvider creates a database provider using provider self-configuration.
// The provider reads its own environment variables - composition layer is provider-agnostic.
//
// Uses CONFIG_DATABASE_PROVIDER environment variable to select which provider to use:
//   - "postgres" or "postgresql" → PostgreSQL provider
//   - "firestore" → Firestore provider
//   - "mock_db" or "mock" → Mock provider (default)
func CreateDatabaseProvider() (contracts.Provider, error) {
	providerName := strings.ToLower(os.Getenv("CONFIG_DATABASE_PROVIDER"))

	// Normalize provider names
	switch providerName {
	case "postgres":
		providerName = "postgresql"
	case "mock_db", "":
		providerName = "mock"
	}

	// Let the provider build and configure itself from environment
	providerInstance, err := registry.BuildDatabaseProviderFromEnv(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create database provider '%s': %w", providerName, err)
	}

	// Wrap to satisfy composition contracts
	return NewProviderWrapper(providerInstance, contracts.ProviderTypeDatabase), nil
}
