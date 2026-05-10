package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - Finance domain
	forexratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/finance/forex_rate"
)

// FinanceRepositories contains all finance domain repositories.
type FinanceRepositories struct {
	ForexRate forexratepb.ForexRateDomainServiceServer
}

// NewFinanceRepositories creates and returns a new set of FinanceRepositories.
// Individual repository failures are logged but do not prevent other repositories
// from being created (graceful degradation per-repository).
func NewFinanceRepositories(dbProvider contracts.Provider, tableConfig *registry.TableConfig) (*FinanceRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()
	repos := &FinanceRepositories{}
	var skipped []string

	// Helper: try to create a repository, log and skip on failure
	tryCreate := func(entity string) interface{} {
		repo, err := repoCreator.CreateRepository(entity, conn, tableConfig.TableName(entity))
		if err != nil {
			skipped = append(skipped, entity)
			return nil
		}
		return repo
	}

	if r := tryCreate(entityid.ForexRate); r != nil {
		repos.ForexRate = r.(forexratepb.ForexRateDomainServiceServer)
	}

	if len(skipped) > 0 {
		fmt.Printf("Finance repos skipped (no adapter registered): %v\n", skipped)
	}

	return repos, nil
}
