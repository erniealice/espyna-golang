package revenuelineitem

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
)

// ListRevenueLineItemsRepositories groups all repository dependencies
type ListRevenueLineItemsRepositories struct {
	RevenueLineItem pb.RevenueLineItemDomainServiceServer
}

// ListRevenueLineItemsServices groups all business service dependencies
type ListRevenueLineItemsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenueLineItem, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "revenue_line_item.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.RevenueLineItem.ListRevenueLineItems(ctx, req)
}
