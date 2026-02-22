package product_option_value

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	productoptionvaluepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_option_value"
)

// DeleteProductOptionValueRepositories groups all repository dependencies
type DeleteProductOptionValueRepositories struct {
	ProductOptionValue productoptionvaluepb.ProductOptionValueDomainServiceServer // Primary entity repository
}

// DeleteProductOptionValueServices groups all business service dependencies
type DeleteProductOptionValueServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeleteProductOptionValueUseCase handles the business logic for deleting product option values
type DeleteProductOptionValueUseCase struct {
	repositories DeleteProductOptionValueRepositories
	services     DeleteProductOptionValueServices
}

// NewDeleteProductOptionValueUseCase creates a new DeleteProductOptionValueUseCase
func NewDeleteProductOptionValueUseCase(
	repositories DeleteProductOptionValueRepositories,
	services DeleteProductOptionValueServices,
) *DeleteProductOptionValueUseCase {
	return &DeleteProductOptionValueUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete product option value operation
func (uc *DeleteProductOptionValueUseCase) Execute(ctx context.Context, req *productoptionvaluepb.DeleteProductOptionValueRequest) (*productoptionvaluepb.DeleteProductOptionValueResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductOptionValue, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product option value deletion within a transaction
func (uc *DeleteProductOptionValueUseCase) executeWithTransaction(ctx context.Context, req *productoptionvaluepb.DeleteProductOptionValueRequest) (*productoptionvaluepb.DeleteProductOptionValueResponse, error) {
	var result *productoptionvaluepb.DeleteProductOptionValueResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return result, nil
}

// executeCore contains the core business logic for deleting a product option value
func (uc *DeleteProductOptionValueUseCase) executeCore(ctx context.Context, req *productoptionvaluepb.DeleteProductOptionValueRequest) (*productoptionvaluepb.DeleteProductOptionValueResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.errors.authorization_failed", "Authorization failed for product option values [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductOptionValue, ports.ActionDelete)
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

	// Call repository
	resp, err := uc.repositories.ProductOptionValue.DeleteProductOptionValue(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.errors.deletion_failed", "Product option value deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteProductOptionValueUseCase) validateInput(ctx context.Context, req *productoptionvaluepb.DeleteProductOptionValueRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.validation.data_required", "Product option value data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.validation.id_required", "Product option value ID is required [DEFAULT]"))
	}
	return nil
}
