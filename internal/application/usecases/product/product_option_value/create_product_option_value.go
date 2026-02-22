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

// CreateProductOptionValueRepositories groups all repository dependencies
type CreateProductOptionValueRepositories struct {
	ProductOptionValue productoptionvaluepb.ProductOptionValueDomainServiceServer // Primary entity repository
}

// CreateProductOptionValueServices groups all business service dependencies
type CreateProductOptionValueServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateProductOptionValueUseCase handles the business logic for creating product option values
type CreateProductOptionValueUseCase struct {
	repositories CreateProductOptionValueRepositories
	services     CreateProductOptionValueServices
}

// NewCreateProductOptionValueUseCase creates use case with grouped dependencies
func NewCreateProductOptionValueUseCase(
	repositories CreateProductOptionValueRepositories,
	services CreateProductOptionValueServices,
) *CreateProductOptionValueUseCase {
	return &CreateProductOptionValueUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create product option value operation
func (uc *CreateProductOptionValueUseCase) Execute(ctx context.Context, req *productoptionvaluepb.CreateProductOptionValueRequest) (*productoptionvaluepb.CreateProductOptionValueResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductOptionValue, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product option value creation within a transaction
func (uc *CreateProductOptionValueUseCase) executeWithTransaction(ctx context.Context, req *productoptionvaluepb.CreateProductOptionValueRequest) (*productoptionvaluepb.CreateProductOptionValueResponse, error) {
	var result *productoptionvaluepb.CreateProductOptionValueResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("product option value creation failed: %w", err)
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
func (uc *CreateProductOptionValueUseCase) executeCore(ctx context.Context, req *productoptionvaluepb.CreateProductOptionValueRequest) (*productoptionvaluepb.CreateProductOptionValueResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.errors.authorization_failed", "Authorization failed for product option values [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductOptionValue, ports.ActionCreate)
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
	return uc.repositories.ProductOptionValue.CreateProductOptionValue(ctx, req)
}

// validateInput validates the input request
func (uc *CreateProductOptionValueUseCase) validateInput(ctx context.Context, req *productoptionvaluepb.CreateProductOptionValueRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.validation.data_required", "Product option value data is required [DEFAULT]"))
	}
	if req.Data.ProductOptionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.validation.product_option_id_required", "Product option ID is required [DEFAULT]"))
	}
	if req.Data.Label == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.validation.label_required", "Label is required [DEFAULT]"))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.validation.value_required", "Value is required [DEFAULT]"))
	}
	return nil
}

// enrichData adds generated fields and audit information
func (uc *CreateProductOptionValueUseCase) enrichData(data *productoptionvaluepb.ProductOptionValue) error {
	now := time.Now()

	// Generate ID if not provided
	if data.Id == "" {
		data.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	data.DateCreated = &[]int64{now.UnixMilli()}[0]
	data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	data.Active = true

	return nil
}
