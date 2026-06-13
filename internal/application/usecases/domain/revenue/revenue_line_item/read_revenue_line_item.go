package revenuelineitem

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
)

// ReadRevenueLineItemRepositories groups all repository dependencies
type ReadRevenueLineItemRepositories struct {
	RevenueLineItem pb.RevenueLineItemDomainServiceServer
}

// ReadRevenueLineItemServices groups all business service dependencies
type ReadRevenueLineItemServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadRevenueLineItemUseCase handles the business logic for reading a revenue line item
type ReadRevenueLineItemUseCase struct {
	repositories ReadRevenueLineItemRepositories
	services     ReadRevenueLineItemServices
}

// NewReadRevenueLineItemUseCase creates use case with grouped dependencies
func NewReadRevenueLineItemUseCase(
	repositories ReadRevenueLineItemRepositories,
	services ReadRevenueLineItemServices,
) *ReadRevenueLineItemUseCase {
	return &ReadRevenueLineItemUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read revenue line item operation
func (uc *ReadRevenueLineItemUseCase) Execute(ctx context.Context, req *pb.ReadRevenueLineItemRequest) (*pb.ReadRevenueLineItemResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityRevenueLineItem,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_line_item.validation.id_required", "Revenue line item ID is required [DEFAULT]"))
	}

	return uc.repositories.RevenueLineItem.ReadRevenueLineItem(ctx, req)
}
