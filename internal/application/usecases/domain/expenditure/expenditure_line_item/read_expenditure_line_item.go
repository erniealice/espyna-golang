package expenditurelineitem

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
)

// ReadExpenditureLineItemRepositories groups all repository dependencies
type ReadExpenditureLineItemRepositories struct {
	ExpenditureLineItem pb.ExpenditureLineItemDomainServiceServer
}

// ReadExpenditureLineItemServices groups all business service dependencies
type ReadExpenditureLineItemServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadExpenditureLineItemUseCase handles the business logic for reading an expenditure line item
type ReadExpenditureLineItemUseCase struct {
	repositories ReadExpenditureLineItemRepositories
	services     ReadExpenditureLineItemServices
}

// NewReadExpenditureLineItemUseCase creates use case with grouped dependencies
func NewReadExpenditureLineItemUseCase(
	repositories ReadExpenditureLineItemRepositories,
	services ReadExpenditureLineItemServices,
) *ReadExpenditureLineItemUseCase {
	return &ReadExpenditureLineItemUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read expenditure line item operation
func (uc *ReadExpenditureLineItemUseCase) Execute(ctx context.Context, req *pb.ReadExpenditureLineItemRequest) (*pb.ReadExpenditureLineItemResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityExpenditureLineItem, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "expenditure_line_item.validation.id_required", "Expenditure line item ID is required [DEFAULT]"))
	}

	return uc.repositories.ExpenditureLineItem.ReadExpenditureLineItem(ctx, req)
}
