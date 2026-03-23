package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - Fulfillment domain
	fulfillmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

// FulfillmentRepositories contains all fulfillment domain repositories.
type FulfillmentRepositories struct {
	Fulfillment fulfillmentpb.FulfillmentDomainServiceServer
}

// NewFulfillmentRepositories creates and returns a new set of FulfillmentRepositories.
// Individual repository failures are logged but do not prevent other repositories
// from being created (graceful degradation per-repository).
func NewFulfillmentRepositories(dbProvider contracts.Provider, tableConfig *registry.TableConfig) (*FulfillmentRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()
	repos := &FulfillmentRepositories{}
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

	if r := tryCreate(entityid.Fulfillment); r != nil {
		repos.Fulfillment = r.(fulfillmentpb.FulfillmentDomainServiceServer)
	}

	if len(skipped) > 0 {
		fmt.Printf("⚠️  Fulfillment repos skipped (no adapter registered): %v\n", skipped)
	}

	return repos, nil
}
