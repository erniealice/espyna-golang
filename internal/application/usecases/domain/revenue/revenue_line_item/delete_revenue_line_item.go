package revenuelineitem

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
)

// DeleteRevenueLineItemRepositories groups all repository dependencies
type DeleteRevenueLineItemRepositories struct {
	RevenueLineItem pb.RevenueLineItemDomainServiceServer
}

// DeleteRevenueLineItemServices groups all business service dependencies
type DeleteRevenueLineItemServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// DeleteRevenueLineItemUseCase handles the business logic for deleting revenue line items
type DeleteRevenueLineItemUseCase struct {
	repositories DeleteRevenueLineItemRepositories
	services     DeleteRevenueLineItemServices
}

// NewDeleteRevenueLineItemUseCase creates a new DeleteRevenueLineItemUseCase
func NewDeleteRevenueLineItemUseCase(
	repositories DeleteRevenueLineItemRepositories,
	services DeleteRevenueLineItemServices,
) *DeleteRevenueLineItemUseCase {
	return &DeleteRevenueLineItemUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete revenue line item operation
func (uc *DeleteRevenueLineItemUseCase) Execute(ctx context.Context, req *pb.DeleteRevenueLineItemRequest) (*pb.DeleteRevenueLineItemResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityRevenueLineItem, ports.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_line_item.validation.id_required", "Revenue line item ID is required [DEFAULT]"))
	}

	return uc.repositories.RevenueLineItem.DeleteRevenueLineItem(ctx, req)
}
