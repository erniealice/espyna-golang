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

const entityExpenditureLineItem = "expenditure_line_item"

// CreateExpenditureLineItemRepositories groups all repository dependencies
type CreateExpenditureLineItemRepositories struct {
	ExpenditureLineItem pb.ExpenditureLineItemDomainServiceServer
}

// CreateExpenditureLineItemServices groups all business service dependencies
type CreateExpenditureLineItemServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateExpenditureLineItemUseCase handles the business logic for creating expenditure line items
type CreateExpenditureLineItemUseCase struct {
	repositories CreateExpenditureLineItemRepositories
	services     CreateExpenditureLineItemServices
}

// NewCreateExpenditureLineItemUseCase creates use case with grouped dependencies
func NewCreateExpenditureLineItemUseCase(
	repositories CreateExpenditureLineItemRepositories,
	services CreateExpenditureLineItemServices,
) *CreateExpenditureLineItemUseCase {
	return &CreateExpenditureLineItemUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create expenditure line item operation
func (uc *CreateExpenditureLineItemUseCase) Execute(ctx context.Context, req *pb.CreateExpenditureLineItemRequest) (*pb.CreateExpenditureLineItemResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenditureLineItem, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *pb.CreateExpenditureLineItemResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("expenditure line item creation failed: %w", err)
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

func (uc *CreateExpenditureLineItemUseCase) executeCore(ctx context.Context, req *pb.CreateExpenditureLineItemRequest) (*pb.CreateExpenditureLineItemResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "expenditure_line_item.validation.data_required", "Expenditure line item data is required [DEFAULT]"))
	}

	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDService.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true

	return uc.repositories.ExpenditureLineItem.CreateExpenditureLineItem(ctx, req)
}
