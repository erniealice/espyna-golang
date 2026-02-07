package domain

import (
	"fmt"

	"leapfor.xyz/espyna/internal/composition/contracts"
	"leapfor.xyz/espyna/internal/infrastructure/registry"

	// Protobuf domain services - Common domain
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
	categorypb "leapfor.xyz/esqyma/golang/v1/domain/common"
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

	// Create attribute repository using configured table name from dbTableConfig
	attributeRepo, err := repoCreator.CreateRepository("attribute", conn, dbTableConfig.Attribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create attribute repository: %w", err)
	}

	// Create category repository using configured table name from dbTableConfig
	categoryRepo, err := repoCreator.CreateRepository("category", conn, dbTableConfig.Category)
	if err != nil {
		return nil, fmt.Errorf("failed to create category repository: %w", err)
	}

	// Type assert repositories to their interfaces
	return &CommonRepositories{
		Attribute: attributeRepo.(attributepb.AttributeDomainServiceServer),
		Category:  categoryRepo.(categorypb.CategoryDomainServiceServer),
	}, nil
}
