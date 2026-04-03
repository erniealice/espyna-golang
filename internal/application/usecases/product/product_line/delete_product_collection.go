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

// DeleteProductLineUseCase handles the business logic for deleting product lines
// DeleteProductLineRepositories groups all repository dependencies
type DeleteProductLineRepositories struct {
	ProductLine productlinepb.ProductLineDomainServiceServer // Primary entity repository
}

// DeleteProductLineServices groups all business service dependencies
type DeleteProductLineServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeleteProductLineUseCase handles the business logic for deleting product lines
type DeleteProductLineUseCase struct {
	repositories DeleteProductLineRepositories
	services     DeleteProductLineServices
}

// NewDeleteProductLineUseCase creates a new DeleteProductLineUseCase
func NewDeleteProductLineUseCase(
	repositories DeleteProductLineRepositories,
	services DeleteProductLineServices,
) *DeleteProductLineUseCase {
	return &DeleteProductLineUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete product line operation
func (uc *DeleteProductLineUseCase) Execute(ctx context.Context, req *productlinepb.DeleteProductLineRequest) (*productlinepb.DeleteProductLineResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductLine, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product line deletion within a transaction
func (uc *DeleteProductLineUseCase) executeWithTransaction(ctx context.Context, req *productlinepb.DeleteProductLineRequest) (*productlinepb.DeleteProductLineResponse, error) {
	var result *productlinepb.DeleteProductLineResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return result, nil
}

// executeCore contains the core business logic for deleting a product line
func (uc *DeleteProductLineUseCase) executeCore(ctx context.Context, req *productlinepb.DeleteProductLineRequest) (*productlinepb.DeleteProductLineResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.authorization_failed", "Authorization failed for product lines [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductLine, ports.ActionDelete)
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

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductLine.DeleteProductLine(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteProductLineUseCase) validateInput(ctx context.Context, req *productlinepb.DeleteProductLineRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.validation.data_required", "Product line data is required [DEFAULT]"))
	}
	if req.Data.ProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.validation.product_id_required", "Product ID is required [DEFAULT]"))
	}
	if req.Data.LineId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.validation.line_id_required", "Line ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for product line deletion
func (uc *DeleteProductLineUseCase) validateBusinessRules(ctx context.Context, req *productlinepb.DeleteProductLineRequest) error {
	// Additional business rule validation can be added here
	// For example: check if product line is referenced by other entities
	if uc.isProductLineInUse(ctx, req.Data.ProductId, req.Data.LineId) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.in_use", "Product line is currently in use and cannot be deleted [DEFAULT]"))
	}
	return nil
}

// isProductLineInUse checks if the product line is referenced by other entities
func (uc *DeleteProductLineUseCase) isProductLineInUse(ctx context.Context, productID, lineID string) bool {
	// Placeholder for actual implementation
	// TODO: Implement actual check for product line usage
	return false
}
