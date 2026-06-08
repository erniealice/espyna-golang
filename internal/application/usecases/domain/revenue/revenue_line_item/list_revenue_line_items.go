package revenuelineitem

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
)

// ListRevenueLineItemsRepositories groups all repository dependencies
type ListRevenueLineItemsRepositories struct {
	RevenueLineItem pb.RevenueLineItemDomainServiceServer
}

// ListRevenueLineItemsServices groups all business service dependencies
type ListRevenueLineItemsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListRevenueLineItemsUseCase handles the business logic for listing revenue line items
type ListRevenueLineItemsUseCase struct {
	repositories ListRevenueLineItemsRepositories
	services     ListRevenueLineItemsServices
}

// NewListRevenueLineItemsUseCase creates a new ListRevenueLineItemsUseCase
func NewListRevenueLineItemsUseCase(
	repositories ListRevenueLineItemsRepositories,
	services ListRevenueLineItemsServices,
) *ListRevenueLineItemsUseCase {
	return &ListRevenueLineItemsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list revenue line items operation
func (uc *ListRevenueLineItemsUseCase) Execute(ctx context.Context, req *pb.ListRevenueLineItemsRequest) (*pb.ListRevenueLineItemsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityRevenueLineItem, entityid.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_line_item.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.RevenueLineItem.ListRevenueLineItems(ctx, req)
}
