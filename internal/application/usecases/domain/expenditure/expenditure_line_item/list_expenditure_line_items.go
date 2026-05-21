package expenditurelineitem

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
)

// ListExpenditureLineItemsRepositories groups all repository dependencies
type ListExpenditureLineItemsRepositories struct {
	ExpenditureLineItem pb.ExpenditureLineItemDomainServiceServer
}

// ListExpenditureLineItemsServices groups all business service dependencies
type ListExpenditureLineItemsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityExpenditureLineItem, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "expenditure_line_item.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.ExpenditureLineItem.ListExpenditureLineItems(ctx, req)
}
