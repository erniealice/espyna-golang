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
//   - "postgresql" → PostgreSQL provider
//   - "firestore" → Firestore provider
//   - "mock_db"   → Mock provider
//
// Legacy aliases ("postgres", "mock", "") are no longer accepted and cause a
// startup error. Set the canonical token explicitly.
func CreateDatabaseProvider() (contracts.Provider, error) {
	providerName := strings.ToLower(os.Getenv("CONFIG_DATABASE_PROVIDER"))

	// Canonical tokens only — no aliases, no empty-string fallback.
	switch providerName {
	case "postgresql", "firestore", "mock_db":
		// accepted
	case "postgres":
		return nil, fmt.Errorf("CONFIG_DATABASE_PROVIDER=%q is a retired alias — use \"postgresql\"", providerName)
	case "mock":
		return nil, fmt.Errorf("CONFIG_DATABASE_PROVIDER=%q is a retired alias — use \"mock_db\"", providerName)
	case "":
		return nil, fmt.Errorf("CONFIG_DATABASE_PROVIDER is empty — set it to one of: postgresql, firestore, mock_db")
	default:
		return nil, fmt.Errorf("CONFIG_DATABASE_PROVIDER=%q is not a recognized database provider (valid: postgresql, firestore, mock_db)", providerName)
	}

	// Let the provider build and configure itself from environment
	providerInstance, err := registry.BuildDatabaseProviderFromEnv(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create database provider '%s': %w", providerName, err)
	}

	// Wrap to satisfy composition contracts
	return NewProviderWrapper(providerInstance, contracts.ProviderTypeDatabase), nil
}
