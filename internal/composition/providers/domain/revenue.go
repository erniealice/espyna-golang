package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - Revenue domain
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenueattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_attribute"
	revenuecategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_category"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
)

// RevenueRepositories contains all revenue domain repositories
type RevenueRepositories struct {
	Revenue          revenuepb.RevenueDomainServiceServer
	RevenueLineItem  revenuelineitempb.RevenueLineItemDomainServiceServer
	RevenueCategory  revenuecategorypb.RevenueCategoryDomainServiceServer
	RevenueAttribute revenueattributepb.RevenueAttributeDomainServiceServer
}

// NewRevenueRepositories creates and returns a new set of RevenueRepositories
func NewRevenueRepositories(dbProvider contracts.Provider, dbTableConfig *registry.DatabaseTableConfig) (*RevenueRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	revenueRepo, err := repoCreator.CreateRepository(entityid.Revenue, conn, dbTableConfig.Revenue)
	if err != nil {
		return nil, fmt.Errorf("failed to create revenue repository: %w", err)
	}

	revenueLineItemRepo, err := repoCreator.CreateRepository(entityid.RevenueLineItem, conn, dbTableConfig.RevenueLineItem)
	if err != nil {
		return nil, fmt.Errorf("failed to create revenue_line_item repository: %w", err)
	}

	revenueCategoryRepo, err := repoCreator.CreateRepository(entityid.RevenueCategory, conn, dbTableConfig.RevenueCategory)
	if err != nil {
		return nil, fmt.Errorf("failed to create revenue_category repository: %w", err)
	}

	revenueAttributeRepo, err := repoCreator.CreateRepository(entityid.RevenueAttribute, conn, dbTableConfig.RevenueAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create revenue_attribute repository: %w", err)
	}

	return &RevenueRepositories{
		Revenue:          revenueRepo.(revenuepb.RevenueDomainServiceServer),
		RevenueLineItem:  revenueLineItemRepo.(revenuelineitempb.RevenueLineItemDomainServiceServer),
		RevenueCategory:  revenueCategoryRepo.(revenuecategorypb.RevenueCategoryDomainServiceServer),
		RevenueAttribute: revenueAttributeRepo.(revenueattributepb.RevenueAttributeDomainServiceServer),
	}, nil
}
