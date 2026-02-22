package product_option

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	productoptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_option"
)

// DeleteProductOptionRepositories groups all repository dependencies
type DeleteProductOptionRepositories struct {
	ProductOption productoptionpb.ProductOptionDomainServiceServer // Primary entity repository
}

// DeleteProductOptionServices groups all business service dependencies
type DeleteProductOptionServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeleteProductOptionUseCase handles the business logic for deleting product options
type DeleteProductOptionUseCase struct {
	repositories DeleteProductOptionRepositories
	services     DeleteProductOptionServices
}

// NewDeleteProductOptionUseCase creates a new DeleteProductOptionUseCase
func NewDeleteProductOptionUseCase(
	repositories DeleteProductOptionRepositories,
	services DeleteProductOptionServices,
) *DeleteProductOptionUseCase {
	return &DeleteProductOptionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete product option operation
func (uc *DeleteProductOptionUseCase) Execute(ctx context.Context, req *productoptionpb.DeleteProductOptionRequest) (*productoptionpb.DeleteProductOptionResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductOption, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product option deletion within a transaction
func (uc *DeleteProductOptionUseCase) executeWithTransaction(ctx context.Context, req *productoptionpb.DeleteProductOptionRequest) (*productoptionpb.DeleteProductOptionResponse, error) {
	var result *productoptionpb.DeleteProductOptionResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return result, nil
}

// executeCore contains the core business logic for deleting a product option
func (uc *DeleteProductOptionUseCase) executeCore(ctx context.Context, req *productoptionpb.DeleteProductOptionRequest) (*productoptionpb.DeleteProductOptionResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.errors.authorization_failed", "Authorization failed for product options [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductOption, ports.ActionDelete)
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

	// Call repository
	resp, err := uc.repositories.ProductOption.DeleteProductOption(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.errors.deletion_failed", "Product option deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteProductOptionUseCase) validateInput(ctx context.Context, req *productoptionpb.DeleteProductOptionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.validation.data_required", "Product option data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.validation.id_required", "Product option ID is required [DEFAULT]"))
	}
	return nil
}
