package revenuelineitem

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
)

const entityRevenueLineItem = "revenue_line_item"

// CreateRevenueLineItemRepositories groups all repository dependencies
type CreateRevenueLineItemRepositories struct {
	RevenueLineItem pb.RevenueLineItemDomainServiceServer
}

// CreateRevenueLineItemServices groups all business service dependencies
type CreateRevenueLineItemServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateRevenueLineItemUseCase handles the business logic for creating revenue line items
type CreateRevenueLineItemUseCase struct {
	repositories CreateRevenueLineItemRepositories
	services     CreateRevenueLineItemServices
}

// NewCreateRevenueLineItemUseCase creates use case with grouped dependencies
func NewCreateRevenueLineItemUseCase(
	repositories CreateRevenueLineItemRepositories,
	services CreateRevenueLineItemServices,
) *CreateRevenueLineItemUseCase {
	return &CreateRevenueLineItemUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create revenue line item operation
func (uc *CreateRevenueLineItemUseCase) Execute(ctx context.Context, req *pb.CreateRevenueLineItemRequest) (*pb.CreateRevenueLineItemResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenueLineItem, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *pb.CreateRevenueLineItemResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("revenue line item creation failed: %w", err)
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

func (uc *CreateRevenueLineItemUseCase) executeCore(ctx context.Context, req *pb.CreateRevenueLineItemRequest) (*pb.CreateRevenueLineItemResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "revenue_line_item.validation.data_required", "Revenue line item data is required [DEFAULT]"))
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

	return uc.repositories.RevenueLineItem.CreateRevenueLineItem(ctx, req)
}
