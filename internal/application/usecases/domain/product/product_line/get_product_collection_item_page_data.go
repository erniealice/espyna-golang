package product_line

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_line"
)

type GetProductLineItemPageDataRepositories struct {
	ProductLine productlinepb.ProductLineDomainServiceServer
}

type GetProductLineItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetProductLineItemPageDataUseCase handles the business logic for getting product line item page data
type GetProductLineItemPageDataUseCase struct {
	repositories GetProductLineItemPageDataRepositories
	services     GetProductLineItemPageDataServices
}

// NewGetProductLineItemPageDataUseCase creates a new GetProductLineItemPageDataUseCase
func NewGetProductLineItemPageDataUseCase(
	repositories GetProductLineItemPageDataRepositories,
	services GetProductLineItemPageDataServices,
) *GetProductLineItemPageDataUseCase {
	return &GetProductLineItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get product line item page data operation
func (uc *GetProductLineItemPageDataUseCase) Execute(
	ctx context.Context,
	req *productlinepb.GetProductLineItemPageDataRequest,
) (*productlinepb.GetProductLineItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityProductLine, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.ProductLineId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product line item page data retrieval within a transaction
func (uc *GetProductLineItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *productlinepb.GetProductLineItemPageDataRequest,
) (*productlinepb.GetProductLineItemPageDataResponse, error) {
	var result *productlinepb.GetProductLineItemPageDataResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.Translator,
				"product_line.errors.item_page_data_failed",
				"product line item page data retrieval failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting product line item page data
func (uc *GetProductLineItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *productlinepb.GetProductLineItemPageDataRequest,
) (*productlinepb.GetProductLineItemPageDataResponse, error) {
	// Create read request for the product line
	readReq := &productlinepb.ReadProductLineRequest{
		Data: &productlinepb.ProductLine{
			Id: req.ProductLineId,
		},
	}

	// Retrieve the product line
	readResp, err := uc.repositories.ProductLine.ReadProductLine(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"product_line.errors.read_failed",
			"failed to retrieve product line: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"product_line.errors.not_found",
			"product line not found",
		))
	}

	// Get the product line (should be only one)
	productLine := readResp.Data[0]

	// Validate that we got the expected product line
	if productLine.Id != req.ProductLineId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"product_line.errors.id_mismatch",
			"retrieved product line ID does not match requested ID",
		))
	}

	return &productlinepb.GetProductLineItemPageDataResponse{
		ProductLine: productLine,
		Success:     true,
	}, nil
}

// validateInput validates the input request
func (uc *GetProductLineItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *productlinepb.GetProductLineItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"product_line.validation.request_required",
			"request is required",
		))
	}

	if req.ProductLineId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"product_line.validation.id_required",
			"product line ID is required",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading product line item page data
func (uc *GetProductLineItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	productLineId string,
) error {
	// Validate product line ID format
	if len(productLineId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"product_line.validation.id_too_short",
			"product line ID is too short",
		))
	}

	return nil
}
