package inventory_item

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	inventoryitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_item"
)

// DeleteInventoryItemRepositories groups all repository dependencies
type DeleteInventoryItemRepositories struct {
	InventoryItem inventoryitempb.InventoryItemDomainServiceServer // Primary entity repository
}

// DeleteInventoryItemServices groups all business service dependencies
type DeleteInventoryItemServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeleteInventoryItemUseCase handles the business logic for deleting inventory items
type DeleteInventoryItemUseCase struct {
	repositories DeleteInventoryItemRepositories
	services     DeleteInventoryItemServices
}

// NewDeleteInventoryItemUseCase creates a new DeleteInventoryItemUseCase
func NewDeleteInventoryItemUseCase(
	repositories DeleteInventoryItemRepositories,
	services DeleteInventoryItemServices,
) *DeleteInventoryItemUseCase {
	return &DeleteInventoryItemUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete inventory item operation
func (uc *DeleteInventoryItemUseCase) Execute(ctx context.Context, req *inventoryitempb.DeleteInventoryItemRequest) (*inventoryitempb.DeleteInventoryItemResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventoryItem, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes inventory item deletion within a transaction
func (uc *DeleteInventoryItemUseCase) executeWithTransaction(ctx context.Context, req *inventoryitempb.DeleteInventoryItemRequest) (*inventoryitempb.DeleteInventoryItemResponse, error) {
	var result *inventoryitempb.DeleteInventoryItemResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return result, nil
}

// executeCore contains the core business logic for deleting an inventory item
func (uc *DeleteInventoryItemUseCase) executeCore(ctx context.Context, req *inventoryitempb.DeleteInventoryItemRequest) (*inventoryitempb.DeleteInventoryItemResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.errors.authorization_failed", "Authorization failed for inventory items [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventoryItem, ports.ActionDelete)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.errors.authorization_failed", "Authorization failed for inventory items [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.errors.authorization_failed", "Authorization failed for inventory items [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.InventoryItem.DeleteInventoryItem(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.errors.deletion_failed", "Inventory item deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteInventoryItemUseCase) validateInput(ctx context.Context, req *inventoryitempb.DeleteInventoryItemRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.validation.data_required", "Inventory item data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.validation.id_required", "Inventory item ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for inventory item deletion
func (uc *DeleteInventoryItemUseCase) validateBusinessRules(ctx context.Context, req *inventoryitempb.DeleteInventoryItemRequest) error {
	// Check if inventory item is in use
	if uc.isInventoryItemInUse(ctx, req.Data.Id) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.errors.in_use", "Inventory item is currently in use and cannot be deleted [DEFAULT]"))
	}
	return nil
}

// isInventoryItemInUse checks if the inventory item is referenced by other entities
func (uc *DeleteInventoryItemUseCase) isInventoryItemInUse(ctx context.Context, itemID string) bool {
	// Placeholder for actual implementation
	// TODO: Implement actual check for inventory item usage (serials, transactions, etc.)
	return false
}
