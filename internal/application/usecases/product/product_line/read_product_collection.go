package product_line

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	productlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_line"
)

// ReadProductLineUseCase handles the business logic for reading a product line
// ReadProductLineRepositories groups all repository dependencies
type ReadProductLineRepositories struct {
	ProductLine productlinepb.ProductLineDomainServiceServer // Primary entity repository
}

// ReadProductLineServices groups all business service dependencies
type ReadProductLineServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ReadProductLineUseCase handles the business logic for reading a product line
type ReadProductLineUseCase struct {
	repositories ReadProductLineRepositories
	services     ReadProductLineServices
}

// NewReadProductLineUseCase creates a new ReadProductLineUseCase
func NewReadProductLineUseCase(
	repositories ReadProductLineRepositories,
	services ReadProductLineServices,
) *ReadProductLineUseCase {
	return &ReadProductLineUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read product line operation
func (uc *ReadProductLineUseCase) Execute(ctx context.Context, req *productlinepb.ReadProductLineRequest) (*productlinepb.ReadProductLineResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductLine, ports.ActionRead); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.authorization_failed", "Authorization failed for product lines [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductLine, ports.ActionRead)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.authorization_failed", "Authorization failed for product lines [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.authorization_failed", "Authorization failed for product lines [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductLine.ReadProductLine(ctx, req)
	if err != nil {
		return nil, err
	}

	// Not found error
	if resp == nil || resp.Data == nil || len(resp.Data) == 0 {
		translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "product_line.errors.not_found", map[string]interface{}{"productLineId": req.Data.Id}, "Product line not found")
		return nil, errors.New(translatedError)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadProductLineUseCase) validateInput(ctx context.Context, req *productlinepb.ReadProductLineRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.validation.data_required", "Product line data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.validation.id_required", "Product line ID is required [DEFAULT]"))
	}
	return nil
}
