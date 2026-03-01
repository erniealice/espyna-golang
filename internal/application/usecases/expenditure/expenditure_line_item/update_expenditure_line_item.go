package expenditurelineitem

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
)

// UpdateExpenditureLineItemRepositories groups all repository dependencies
type UpdateExpenditureLineItemRepositories struct {
	ExpenditureLineItem pb.ExpenditureLineItemDomainServiceServer
}

// UpdateExpenditureLineItemServices groups all business service dependencies
type UpdateExpenditureLineItemServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateExpenditureLineItemUseCase handles the business logic for updating expenditure line items
type UpdateExpenditureLineItemUseCase struct {
	repositories UpdateExpenditureLineItemRepositories
	services     UpdateExpenditureLineItemServices
}

// NewUpdateExpenditureLineItemUseCase creates use case with grouped dependencies
func NewUpdateExpenditureLineItemUseCase(
	repositories UpdateExpenditureLineItemRepositories,
	services UpdateExpenditureLineItemServices,
) *UpdateExpenditureLineItemUseCase {
	return &UpdateExpenditureLineItemUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update expenditure line item operation
func (uc *UpdateExpenditureLineItemUseCase) Execute(ctx context.Context, req *pb.UpdateExpenditureLineItemRequest) (*pb.UpdateExpenditureLineItemResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenditureLineItem, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *pb.UpdateExpenditureLineItemResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("expenditure line item update failed: %w", err)
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

func (uc *UpdateExpenditureLineItemUseCase) executeCore(ctx context.Context, req *pb.UpdateExpenditureLineItemRequest) (*pb.UpdateExpenditureLineItemResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "expenditure_line_item.validation.id_required", "Expenditure line item ID is required [DEFAULT]"))
	}

	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.ExpenditureLineItem.UpdateExpenditureLineItem(ctx, req)
}
