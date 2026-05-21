package revenuecategory

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_category"
)

// ListRevenueCategoriesRepositories groups all repository dependencies
type ListRevenueCategoriesRepositories struct {
	RevenueCategory pb.RevenueCategoryDomainServiceServer
}

// ListRevenueCategoriesServices groups all business service dependencies
type ListRevenueCategoriesServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListRevenueCategoriesUseCase handles the business logic for listing revenue categories
type ListRevenueCategoriesUseCase struct {
	repositories ListRevenueCategoriesRepositories
	services     ListRevenueCategoriesServices
}

// NewListRevenueCategoriesUseCase creates a new ListRevenueCategoriesUseCase
func NewListRevenueCategoriesUseCase(
	repositories ListRevenueCategoriesRepositories,
	services ListRevenueCategoriesServices,
) *ListRevenueCategoriesUseCase {
	return &ListRevenueCategoriesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list revenue categories operation
func (uc *ListRevenueCategoriesUseCase) Execute(ctx context.Context, req *pb.ListRevenueCategoriesRequest) (*pb.ListRevenueCategoriesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityRevenueCategory, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_category.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.RevenueCategory.ListRevenueCategories(ctx, req)
}
