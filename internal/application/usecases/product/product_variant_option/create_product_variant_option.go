package product_variant_option

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	productvariantoptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_option"
)

// CreateProductVariantOptionRepositories groups all repository dependencies
type CreateProductVariantOptionRepositories struct {
	ProductVariantOption productvariantoptionpb.ProductVariantOptionDomainServiceServer // Primary entity repository
}

// CreateProductVariantOptionServices groups all business service dependencies
type CreateProductVariantOptionServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateProductVariantOptionUseCase handles the business logic for creating product variant options
type CreateProductVariantOptionUseCase struct {
	repositories CreateProductVariantOptionRepositories
	services     CreateProductVariantOptionServices
}

// NewCreateProductVariantOptionUseCase creates use case with grouped dependencies
func NewCreateProductVariantOptionUseCase(
	repositories CreateProductVariantOptionRepositories,
	services CreateProductVariantOptionServices,
) *CreateProductVariantOptionUseCase {
	return &CreateProductVariantOptionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create product variant option operation
func (uc *CreateProductVariantOptionUseCase) Execute(ctx context.Context, req *productvariantoptionpb.CreateProductVariantOptionRequest) (*productvariantoptionpb.CreateProductVariantOptionResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductVariantOption, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product variant option creation within a transaction
func (uc *CreateProductVariantOptionUseCase) executeWithTransaction(ctx context.Context, req *productvariantoptionpb.CreateProductVariantOptionRequest) (*productvariantoptionpb.CreateProductVariantOptionResponse, error) {
	var result *productvariantoptionpb.CreateProductVariantOptionResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("product variant option creation failed: %w", err)
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
func (uc *CreateProductVariantOptionUseCase) executeCore(ctx context.Context, req *productvariantoptionpb.CreateProductVariantOptionRequest) (*productvariantoptionpb.CreateProductVariantOptionResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.errors.authorization_failed", "Authorization failed for product variant options [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductVariantOption, ports.ActionCreate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.errors.authorization_failed", "Authorization failed for product variant options [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.errors.authorization_failed", "Authorization failed for product variant options [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	return uc.repositories.ProductVariantOption.CreateProductVariantOption(ctx, req)
}

// validateInput validates the input request
func (uc *CreateProductVariantOptionUseCase) validateInput(ctx context.Context, req *productvariantoptionpb.CreateProductVariantOptionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.validation.data_required", "Product variant option data is required [DEFAULT]"))
	}
	if req.Data.ProductVariantId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.validation.product_variant_id_required", "Product variant ID is required [DEFAULT]"))
	}
	if req.Data.ProductOptionValueId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.validation.product_option_value_id_required", "Product option value ID is required [DEFAULT]"))
	}
	return nil
}

// enrichData adds generated fields and audit information
func (uc *CreateProductVariantOptionUseCase) enrichData(data *productvariantoptionpb.ProductVariantOption) error {
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
