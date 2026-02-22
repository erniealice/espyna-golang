package inventory_item

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	inventoryitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_item"
)

// UpdateInventoryItemRepositories groups all repository dependencies
type UpdateInventoryItemRepositories struct {
	InventoryItem inventoryitempb.InventoryItemDomainServiceServer // Primary entity repository
}

// UpdateInventoryItemServices groups all business service dependencies
type UpdateInventoryItemServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateInventoryItemUseCase handles the business logic for updating inventory items
type UpdateInventoryItemUseCase struct {
	repositories UpdateInventoryItemRepositories
	services     UpdateInventoryItemServices
}

// NewUpdateInventoryItemUseCase creates use case with grouped dependencies
func NewUpdateInventoryItemUseCase(
	repositories UpdateInventoryItemRepositories,
	services UpdateInventoryItemServices,
) *UpdateInventoryItemUseCase {
	return &UpdateInventoryItemUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update inventory item operation
func (uc *UpdateInventoryItemUseCase) Execute(ctx context.Context, req *inventoryitempb.UpdateInventoryItemRequest) (*inventoryitempb.UpdateInventoryItemResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventoryItem, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes inventory item update within a transaction
func (uc *UpdateInventoryItemUseCase) executeWithTransaction(ctx context.Context, req *inventoryitempb.UpdateInventoryItemRequest) (*inventoryitempb.UpdateInventoryItemResponse, error) {
	var result *inventoryitempb.UpdateInventoryItemResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "inventory_item.errors.update_failed", "Inventory item update failed [DEFAULT]")
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
func (uc *UpdateInventoryItemUseCase) executeCore(ctx context.Context, req *inventoryitempb.UpdateInventoryItemRequest) (*inventoryitempb.UpdateInventoryItemResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.errors.authorization_failed", "Authorization failed for inventory items [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventoryItem, ports.ActionUpdate)
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

	// Business logic and enrichment
	if err := uc.enrichData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Get existing item to preserve fields
	existingResp, err := uc.repositories.InventoryItem.ReadInventoryItem(ctx, &inventoryitempb.ReadInventoryItemRequest{Data: &inventoryitempb.InventoryItem{Id: req.Data.Id}})
	if err != nil || existingResp == nil || len(existingResp.Data) == 0 {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.errors.not_found", "Inventory item not found for update [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	existingItem := existingResp.Data[0]

	// Preserve the active status if not provided in the request
	if req.Data.Active == false {
		req.Data.Active = existingItem.Active
	}

	// Call repository
	resp, err := uc.repositories.InventoryItem.UpdateInventoryItem(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.errors.update_failed", "Inventory item update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateInventoryItemUseCase) validateInput(ctx context.Context, req *inventoryitempb.UpdateInventoryItemRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.validation.data_required", "Inventory item data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.validation.id_required", "Inventory item ID is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.validation.name_required", "Inventory item name is required [DEFAULT]"))
	}
	return nil
}

// enrichData adds generated fields and audit information
func (uc *UpdateInventoryItemUseCase) enrichData(item *inventoryitempb.InventoryItem) error {
	now := time.Now()

	// Update audit fields
	item.DateModified = &[]int64{now.UnixMilli()}[0]
	item.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for inventory items
func (uc *UpdateInventoryItemUseCase) validateBusinessRules(ctx context.Context, item *inventoryitempb.InventoryItem) error {
	// Validate name length
	name := strings.TrimSpace(item.Name)
	if len(name) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.validation.name_min_length", "Inventory item name must be at least 2 characters long [DEFAULT]"))
	}

	if len(name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.validation.name_max_length", "Inventory item name cannot exceed 100 characters [DEFAULT]"))
	}

	return nil
}
