package product_option_value

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	productoptionvaluepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_option_value"
)

// UpdateProductOptionValueRepositories groups all repository dependencies
type UpdateProductOptionValueRepositories struct {
	ProductOptionValue productoptionvaluepb.ProductOptionValueDomainServiceServer // Primary entity repository
}

// UpdateProductOptionValueServices groups all business service dependencies
type UpdateProductOptionValueServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateProductOptionValueUseCase handles the business logic for updating product option values
type UpdateProductOptionValueUseCase struct {
	repositories UpdateProductOptionValueRepositories
	services     UpdateProductOptionValueServices
}

// NewUpdateProductOptionValueUseCase creates use case with grouped dependencies
func NewUpdateProductOptionValueUseCase(
	repositories UpdateProductOptionValueRepositories,
	services UpdateProductOptionValueServices,
) *UpdateProductOptionValueUseCase {
	return &UpdateProductOptionValueUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update product option value operation
func (uc *UpdateProductOptionValueUseCase) Execute(ctx context.Context, req *productoptionvaluepb.UpdateProductOptionValueRequest) (*productoptionvaluepb.UpdateProductOptionValueResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductOptionValue, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product option value update within a transaction
func (uc *UpdateProductOptionValueUseCase) executeWithTransaction(ctx context.Context, req *productoptionvaluepb.UpdateProductOptionValueRequest) (*productoptionvaluepb.UpdateProductOptionValueResponse, error) {
	var result *productoptionvaluepb.UpdateProductOptionValueResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "product_option_value.errors.update_failed", "Product option value update failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic
func (uc *UpdateProductOptionValueUseCase) executeCore(ctx context.Context, req *productoptionvaluepb.UpdateProductOptionValueRequest) (*productoptionvaluepb.UpdateProductOptionValueResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.errors.authorization_failed", "Authorization failed for product option values [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductOptionValue, ports.ActionUpdate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.errors.authorization_failed", "Authorization failed for product option values [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.errors.authorization_failed", "Authorization failed for product option values [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductOptionValue.UpdateProductOptionValue(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.errors.update_failed", "Product option value update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateProductOptionValueUseCase) validateInput(ctx context.Context, req *productoptionvaluepb.UpdateProductOptionValueRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.validation.data_required", "Product option value data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.validation.id_required", "Product option value ID is required [DEFAULT]"))
	}
	if req.Data.Label == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.validation.label_required", "Label is required [DEFAULT]"))
	}
	return nil
}

// enrichData adds generated fields and audit information
func (uc *UpdateProductOptionValueUseCase) enrichData(data *productoptionvaluepb.ProductOptionValue) error {
	now := time.Now()

	// Update audit fields
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}
