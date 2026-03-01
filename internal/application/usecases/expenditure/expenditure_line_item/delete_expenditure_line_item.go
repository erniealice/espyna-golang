package expenditurelineitem

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
)

// DeleteExpenditureLineItemRepositories groups all repository dependencies
type DeleteExpenditureLineItemRepositories struct {
	ExpenditureLineItem pb.ExpenditureLineItemDomainServiceServer
}

// DeleteExpenditureLineItemServices groups all business service dependencies
type DeleteExpenditureLineItemServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteExpenditureLineItemUseCase handles the business logic for deleting expenditure line items
type DeleteExpenditureLineItemUseCase struct {
	repositories DeleteExpenditureLineItemRepositories
	services     DeleteExpenditureLineItemServices
}

// NewDeleteExpenditureLineItemUseCase creates a new DeleteExpenditureLineItemUseCase
func NewDeleteExpenditureLineItemUseCase(
	repositories DeleteExpenditureLineItemRepositories,
	services DeleteExpenditureLineItemServices,
) *DeleteExpenditureLineItemUseCase {
	return &DeleteExpenditureLineItemUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete expenditure line item operation
func (uc *DeleteExpenditureLineItemUseCase) Execute(ctx context.Context, req *pb.DeleteExpenditureLineItemRequest) (*pb.DeleteExpenditureLineItemResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenditureLineItem, ports.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "expenditure_line_item.validation.id_required", "Expenditure line item ID is required [DEFAULT]"))
	}

	return uc.repositories.ExpenditureLineItem.DeleteExpenditureLineItem(ctx, req)
}
