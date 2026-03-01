package expenditurelineitem

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
)

// ListExpenditureLineItemsRepositories groups all repository dependencies
type ListExpenditureLineItemsRepositories struct {
	ExpenditureLineItem pb.ExpenditureLineItemDomainServiceServer
}

// ListExpenditureLineItemsServices groups all business service dependencies
type ListExpenditureLineItemsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListExpenditureLineItemsUseCase handles the business logic for listing expenditure line items
type ListExpenditureLineItemsUseCase struct {
	repositories ListExpenditureLineItemsRepositories
	services     ListExpenditureLineItemsServices
}

// NewListExpenditureLineItemsUseCase creates a new ListExpenditureLineItemsUseCase
func NewListExpenditureLineItemsUseCase(
	repositories ListExpenditureLineItemsRepositories,
	services ListExpenditureLineItemsServices,
) *ListExpenditureLineItemsUseCase {
	return &ListExpenditureLineItemsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list expenditure line items operation
func (uc *ListExpenditureLineItemsUseCase) Execute(ctx context.Context, req *pb.ListExpenditureLineItemsRequest) (*pb.ListExpenditureLineItemsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenditureLineItem, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "expenditure_line_item.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.ExpenditureLineItem.ListExpenditureLineItems(ctx, req)
}
