package revenuecategory

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_category"
)

// ReadRevenueCategoryRepositories groups all repository dependencies
type ReadRevenueCategoryRepositories struct {
	RevenueCategory pb.RevenueCategoryDomainServiceServer
}

// ReadRevenueCategoryServices groups all business service dependencies
type ReadRevenueCategoryServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityRevenueCategory, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_category.validation.id_required", "Revenue category ID is required [DEFAULT]"))
	}

	return uc.repositories.RevenueCategory.ReadRevenueCategory(ctx, req)
}
