package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"

	// Protobuf domain services - Treasury domain
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

// TreasuryRepositories contains all treasury domain repositories
type TreasuryRepositories struct {
	Collection   collectionpb.CollectionDomainServiceServer
	Disbursement disbursementpb.DisbursementDomainServiceServer
}

// NewTreasuryRepositories creates and returns a new set of TreasuryRepositories
func NewTreasuryRepositories(dbProvider contracts.Provider, dbTableConfig *registry.DatabaseTableConfig) (*TreasuryRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	collectionRepo, err := repoCreator.CreateRepository("collection", conn, dbTableConfig.TreasuryCollection)
	if err != nil {
		return nil, fmt.Errorf("failed to create treasury collection repository: %w", err)
	}

	disbursementRepo, err := repoCreator.CreateRepository("disbursement", conn, dbTableConfig.TreasuryDisbursement)
	if err != nil {
		return nil, fmt.Errorf("failed to create treasury disbursement repository: %w", err)
	}

	return &TreasuryRepositories{
		Collection:   collectionRepo.(collectionpb.CollectionDomainServiceServer),
		Disbursement: disbursementRepo.(disbursementpb.DisbursementDomainServiceServer),
	}, nil
}
