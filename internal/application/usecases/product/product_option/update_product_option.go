package product_option

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	productoptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_option"
)

// UpdateProductOptionRepositories groups all repository dependencies
type UpdateProductOptionRepositories struct {
	ProductOption productoptionpb.ProductOptionDomainServiceServer // Primary entity repository
}

// UpdateProductOptionServices groups all business service dependencies
type UpdateProductOptionServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateProductOptionUseCase handles the business logic for updating product options
type UpdateProductOptionUseCase struct {
	repositories UpdateProductOptionRepositories
	services     UpdateProductOptionServices
}

// NewUpdateProductOptionUseCase creates use case with grouped dependencies
func NewUpdateProductOptionUseCase(
	repositories UpdateProductOptionRepositories,
	services UpdateProductOptionServices,
) *UpdateProductOptionUseCase {
	return &UpdateProductOptionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update product option operation
func (uc *UpdateProductOptionUseCase) Execute(ctx context.Context, req *productoptionpb.UpdateProductOptionRequest) (*productoptionpb.UpdateProductOptionResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductOption, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product option update within a transaction
func (uc *UpdateProductOptionUseCase) executeWithTransaction(ctx context.Context, req *productoptionpb.UpdateProductOptionRequest) (*productoptionpb.UpdateProductOptionResponse, error) {
	var result *productoptionpb.UpdateProductOptionResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "product_option.errors.update_failed", "Product option update failed [DEFAULT]")
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
func (uc *UpdateProductOptionUseCase) executeCore(ctx context.Context, req *productoptionpb.UpdateProductOptionRequest) (*productoptionpb.UpdateProductOptionResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.errors.authorization_failed", "Authorization failed for product options [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductOption, ports.ActionUpdate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.errors.authorization_failed", "Authorization failed for product options [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.errors.authorization_failed", "Authorization failed for product options [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductOption.UpdateProductOption(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.errors.update_failed", "Product option update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateProductOptionUseCase) validateInput(ctx context.Context, req *productoptionpb.UpdateProductOptionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.validation.data_required", "Product option data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.validation.id_required", "Product option ID is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.validation.name_required", "Product option name is required [DEFAULT]"))
	}
	return nil
}

// enrichData adds generated fields and audit information
func (uc *UpdateProductOptionUseCase) enrichData(data *productoptionpb.ProductOption) error {
	now := time.Now()

	// Update audit fields
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}
