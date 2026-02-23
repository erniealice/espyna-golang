package revenuelineitem

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
)

// ReadRevenueLineItemRepositories groups all repository dependencies
type ReadRevenueLineItemRepositories struct {
	RevenueLineItem pb.RevenueLineItemDomainServiceServer
}

// ReadRevenueLineItemServices groups all business service dependencies
type ReadRevenueLineItemServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenueLineItem, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "revenue_line_item.validation.id_required", "Revenue line item ID is required [DEFAULT]"))
	}

	return uc.repositories.RevenueLineItem.ReadRevenueLineItem(ctx, req)
}
