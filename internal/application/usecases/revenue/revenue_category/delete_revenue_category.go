package revenuecategory

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_category"
)

// DeleteRevenueCategoryRepositories groups all repository dependencies
type DeleteRevenueCategoryRepositories struct {
	RevenueCategory pb.RevenueCategoryDomainServiceServer
}

// DeleteRevenueCategoryServices groups all business service dependencies
type DeleteRevenueCategoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteRevenueCategoryUseCase handles the business logic for deleting revenue categories
type DeleteRevenueCategoryUseCase struct {
	repositories DeleteRevenueCategoryRepositories
	services     DeleteRevenueCategoryServices
}

// NewDeleteRevenueCategoryUseCase creates a new DeleteRevenueCategoryUseCase
func NewDeleteRevenueCategoryUseCase(
	repositories DeleteRevenueCategoryRepositories,
	services DeleteRevenueCategoryServices,
) *DeleteRevenueCategoryUseCase {
	return &DeleteRevenueCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete revenue category operation
func (uc *DeleteRevenueCategoryUseCase) Execute(ctx context.Context, req *pb.DeleteRevenueCategoryRequest) (*pb.DeleteRevenueCategoryResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenueCategory, ports.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "revenue_category.validation.id_required", "Revenue category ID is required [DEFAULT]"))
	}

	return uc.repositories.RevenueCategory.DeleteRevenueCategory(ctx, req)
}
