package revenuecategory

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_category"
)

// ReadRevenueCategoryRepositories groups all repository dependencies
type ReadRevenueCategoryRepositories struct {
	RevenueCategory pb.RevenueCategoryDomainServiceServer
}

// ReadRevenueCategoryServices groups all business service dependencies
type ReadRevenueCategoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadRevenueCategoryUseCase handles the business logic for reading a revenue category
type ReadRevenueCategoryUseCase struct {
	repositories ReadRevenueCategoryRepositories
	services     ReadRevenueCategoryServices
}

// NewReadRevenueCategoryUseCase creates use case with grouped dependencies
func NewReadRevenueCategoryUseCase(
	repositories ReadRevenueCategoryRepositories,
	services ReadRevenueCategoryServices,
) *ReadRevenueCategoryUseCase {
	return &ReadRevenueCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read revenue category operation
func (uc *ReadRevenueCategoryUseCase) Execute(ctx context.Context, req *pb.ReadRevenueCategoryRequest) (*pb.ReadRevenueCategoryResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenueCategory, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "revenue_category.validation.id_required", "Revenue category ID is required [DEFAULT]"))
	}

	return uc.repositories.RevenueCategory.ReadRevenueCategory(ctx, req)
}
