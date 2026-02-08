package domain

import (
	"fmt"

	integrationPorts "github.com/erniealice/espyna-golang/internal/application/ports/integration"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// IntegrationPaymentRepository is an alias for the ports interface
type IntegrationPaymentRepository = integrationPorts.IntegrationPaymentRepository

// NewIntegrationPaymentRepository creates the integration payment repository from the database provider
func NewIntegrationPaymentRepository(
	dbProvider contracts.Provider,
	tableConfig *registry.DatabaseTableConfig,
) (IntegrationPaymentRepository, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider is nil")
	}
	if tableConfig == nil {
		return nil, fmt.Errorf("table config is nil")
	}

	// Cast to RepositoryProvider (same pattern as other domain repositories)
	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	// Get the database connection
	conn := repoCreator.GetConnection()

	// Create the repository using the registry
	collectionName := tableConfig.IntegrationPayment
	if collectionName == "" {
		collectionName = "integration_payment"
	}

	repo, err := repoCreator.CreateRepository("integration_payment", conn, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to create integration_payment repository: %w", err)
	}

	integrationPaymentRepo, ok := repo.(IntegrationPaymentRepository)
	if !ok {
		return nil, fmt.Errorf("repository does not implement IntegrationPaymentRepository, got %T", repo)
	}

	return integrationPaymentRepo, nil
}
