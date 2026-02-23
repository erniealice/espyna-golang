package revenuelineitem

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
)

// UpdateRevenueLineItemRepositories groups all repository dependencies
type UpdateRevenueLineItemRepositories struct {
	RevenueLineItem pb.RevenueLineItemDomainServiceServer
}

// UpdateRevenueLineItemServices groups all business service dependencies
type UpdateRevenueLineItemServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateRevenueLineItemUseCase handles the business logic for updating revenue line items
type UpdateRevenueLineItemUseCase struct {
	repositories UpdateRevenueLineItemRepositories
	services     UpdateRevenueLineItemServices
}

// NewUpdateRevenueLineItemUseCase creates use case with grouped dependencies
func NewUpdateRevenueLineItemUseCase(
	repositories UpdateRevenueLineItemRepositories,
	services UpdateRevenueLineItemServices,
) *UpdateRevenueLineItemUseCase {
	return &UpdateRevenueLineItemUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update revenue line item operation
func (uc *UpdateRevenueLineItemUseCase) Execute(ctx context.Context, req *pb.UpdateRevenueLineItemRequest) (*pb.UpdateRevenueLineItemResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenueLineItem, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *pb.UpdateRevenueLineItemResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("revenue line item update failed: %w", err)
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

func (uc *UpdateRevenueLineItemUseCase) executeCore(ctx context.Context, req *pb.UpdateRevenueLineItemRequest) (*pb.UpdateRevenueLineItemResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "revenue_line_item.validation.id_required", "Revenue line item ID is required [DEFAULT]"))
	}

	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.RevenueLineItem.UpdateRevenueLineItem(ctx, req)
}
