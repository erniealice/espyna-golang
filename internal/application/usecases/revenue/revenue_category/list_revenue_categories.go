package revenuecategory

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_category"
)

// ListRevenueCategoriesRepositories groups all repository dependencies
type ListRevenueCategoriesRepositories struct {
	RevenueCategory pb.RevenueCategoryDomainServiceServer
}

// ListRevenueCategoriesServices groups all business service dependencies
type ListRevenueCategoriesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenueCategory, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "revenue_category.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.RevenueCategory.ListRevenueCategories(ctx, req)
}
