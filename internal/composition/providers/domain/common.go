package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"

	// Protobuf domain services - Common domain
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	categorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// CommonRepositories contains all common domain repositories
type CommonRepositories struct {
	Attribute attributepb.AttributeDomainServiceServer
	Category  categorypb.CategoryDomainServiceServer
}

// NewCommonRepositories creates and returns a new set of CommonRepositories
func NewCommonRepositories(dbProvider contracts.Provider, dbTableConfig *registry.DatabaseTableConfig) (*CommonRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	repos := &CommonRepositories{}

	// Create attribute repository (optional — may not have a factory registered)
	attributeRepo, err := repoCreator.CreateRepository("attribute", conn, dbTableConfig.Attribute)
	if err != nil {
		fmt.Printf("⚠️  Attribute repository not available: %v\n", err)
	} else {
		repos.Attribute = attributeRepo.(attributepb.AttributeDomainServiceServer)
	}

	// Create category repository using configured table name from dbTableConfig
	categoryRepo, err := repoCreator.CreateRepository("category", conn, dbTableConfig.Category)
	if err != nil {
		fmt.Printf("⚠️  Category repository not available: %v\n", err)
	} else {
		repos.Category = categoryRepo.(categorypb.CategoryDomainServiceServer)
	}

	// Return error only if no repositories were created at all
	if repos.Attribute == nil && repos.Category == nil {
		return nil, fmt.Errorf("no common repositories could be created")
	}

	return repos, nil
}
