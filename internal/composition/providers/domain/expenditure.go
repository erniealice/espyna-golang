package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"

	// Protobuf domain services - Expenditure domain
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
	expenditureattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_attribute"
	expenditurecategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_category"
	expenditurelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
)

// ExpenditureRepositories contains all expenditure domain repositories
type ExpenditureRepositories struct {
	Expenditure          expenditurepb.ExpenditureDomainServiceServer
	ExpenditureLineItem  expenditurelineitempb.ExpenditureLineItemDomainServiceServer
	ExpenditureCategory  expenditurecategorypb.ExpenditureCategoryDomainServiceServer
	ExpenditureAttribute expenditureattributepb.ExpenditureAttributeDomainServiceServer
}

// NewExpenditureRepositories creates and returns a new set of ExpenditureRepositories
func NewExpenditureRepositories(dbProvider contracts.Provider, dbTableConfig *registry.DatabaseTableConfig) (*ExpenditureRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	expenditureRepo, err := repoCreator.CreateRepository("expenditure", conn, dbTableConfig.Expenditure)
	if err != nil {
		return nil, fmt.Errorf("failed to create expenditure repository: %w", err)
	}

	expenditureLineItemRepo, err := repoCreator.CreateRepository("expenditure_line_item", conn, dbTableConfig.ExpenditureLineItem)
	if err != nil {
		return nil, fmt.Errorf("failed to create expenditure_line_item repository: %w", err)
	}

	expenditureCategoryRepo, err := repoCreator.CreateRepository("expenditure_category", conn, dbTableConfig.ExpenditureCategory)
	if err != nil {
		return nil, fmt.Errorf("failed to create expenditure_category repository: %w", err)
	}

	expenditureAttributeRepo, err := repoCreator.CreateRepository("expenditure_attribute", conn, dbTableConfig.ExpenditureAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create expenditure_attribute repository: %w", err)
	}

	return &ExpenditureRepositories{
		Expenditure:          expenditureRepo.(expenditurepb.ExpenditureDomainServiceServer),
		ExpenditureLineItem:  expenditureLineItemRepo.(expenditurelineitempb.ExpenditureLineItemDomainServiceServer),
		ExpenditureCategory:  expenditureCategoryRepo.(expenditurecategorypb.ExpenditureCategoryDomainServiceServer),
		ExpenditureAttribute: expenditureAttributeRepo.(expenditureattributepb.ExpenditureAttributeDomainServiceServer),
	}, nil
}
