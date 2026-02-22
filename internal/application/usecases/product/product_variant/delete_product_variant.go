package product_variant

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	productvariantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant"
)

// DeleteProductVariantRepositories groups all repository dependencies
type DeleteProductVariantRepositories struct {
	ProductVariant productvariantpb.ProductVariantDomainServiceServer // Primary entity repository
}

// DeleteProductVariantServices groups all business service dependencies
type DeleteProductVariantServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeleteProductVariantUseCase handles the business logic for deleting product variants
type DeleteProductVariantUseCase struct {
	repositories DeleteProductVariantRepositories
	services     DeleteProductVariantServices
}

// NewDeleteProductVariantUseCase creates a new DeleteProductVariantUseCase
func NewDeleteProductVariantUseCase(
	repositories DeleteProductVariantRepositories,
	services DeleteProductVariantServices,
) *DeleteProductVariantUseCase {
	return &DeleteProductVariantUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete product variant operation
func (uc *DeleteProductVariantUseCase) Execute(ctx context.Context, req *productvariantpb.DeleteProductVariantRequest) (*productvariantpb.DeleteProductVariantResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductVariant, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product variant deletion within a transaction
func (uc *DeleteProductVariantUseCase) executeWithTransaction(ctx context.Context, req *productvariantpb.DeleteProductVariantRequest) (*productvariantpb.DeleteProductVariantResponse, error) {
	var result *productvariantpb.DeleteProductVariantResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return result, nil
}

// executeCore contains the core business logic for deleting a product variant
func (uc *DeleteProductVariantUseCase) executeCore(ctx context.Context, req *productvariantpb.DeleteProductVariantRequest) (*productvariantpb.DeleteProductVariantResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant.errors.authorization_failed", "Authorization failed for product variants [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductVariant, ports.ActionDelete)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant.errors.authorization_failed", "Authorization failed for product variants [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant.errors.authorization_failed", "Authorization failed for product variants [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductVariant.DeleteProductVariant(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant.errors.deletion_failed", "Product variant deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteProductVariantUseCase) validateInput(ctx context.Context, req *productvariantpb.DeleteProductVariantRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant.validation.data_required", "Product variant data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant.validation.id_required", "Product variant ID is required [DEFAULT]"))
	}
	return nil
}
