package revenuecategory

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_category"
)

// UpdateRevenueCategoryRepositories groups all repository dependencies
type UpdateRevenueCategoryRepositories struct {
	RevenueCategory pb.RevenueCategoryDomainServiceServer
}

// UpdateRevenueCategoryServices groups all business service dependencies
type UpdateRevenueCategoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateRevenueCategoryUseCase handles the business logic for updating revenue categories
type UpdateRevenueCategoryUseCase struct {
	repositories UpdateRevenueCategoryRepositories
	services     UpdateRevenueCategoryServices
}

// NewUpdateRevenueCategoryUseCase creates use case with grouped dependencies
func NewUpdateRevenueCategoryUseCase(
	repositories UpdateRevenueCategoryRepositories,
	services UpdateRevenueCategoryServices,
) *UpdateRevenueCategoryUseCase {
	return &UpdateRevenueCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update revenue category operation
func (uc *UpdateRevenueCategoryUseCase) Execute(ctx context.Context, req *pb.UpdateRevenueCategoryRequest) (*pb.UpdateRevenueCategoryResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenueCategory, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *pb.UpdateRevenueCategoryResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("revenue category update failed: %w", err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return uc.executeCore(ctx, req)
}

func (uc *UpdateRevenueCategoryUseCase) executeCore(ctx context.Context, req *pb.UpdateRevenueCategoryRequest) (*pb.UpdateRevenueCategoryResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "revenue_category.validation.id_required", "Revenue category ID is required [DEFAULT]"))
	}

	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.RevenueCategory.UpdateRevenueCategory(ctx, req)
}
