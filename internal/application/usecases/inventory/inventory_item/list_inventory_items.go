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

// ListInventoryItemsRepositories groups all repository dependencies
type ListInventoryItemsRepositories struct {
	InventoryItem inventoryitempb.InventoryItemDomainServiceServer // Primary entity repository
}

// ListInventoryItemsServices groups all business service dependencies
type ListInventoryItemsServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListInventoryItemsUseCase handles the business logic for listing inventory items
type ListInventoryItemsUseCase struct {
	repositories ListInventoryItemsRepositories
	services     ListInventoryItemsServices
}

// NewListInventoryItemsUseCase creates a new ListInventoryItemsUseCase
func NewListInventoryItemsUseCase(
	repositories ListInventoryItemsRepositories,
	services ListInventoryItemsServices,
) *ListInventoryItemsUseCase {
	return &ListInventoryItemsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list inventory items operation
func (uc *ListInventoryItemsUseCase) Execute(ctx context.Context, req *inventoryitempb.ListInventoryItemsRequest) (*inventoryitempb.ListInventoryItemsResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventoryItem, ports.ActionList); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.errors.authorization_failed", "Authorization failed for inventory items [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventoryItem, ports.ActionList)
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

	// Call repository
	resp, err := uc.repositories.InventoryItem.ListInventoryItems(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.errors.list_failed", "Failed to retrieve inventory items [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListInventoryItemsUseCase) validateInput(ctx context.Context, req *inventoryitempb.ListInventoryItemsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_item.validation.request_required", "Request is required [DEFAULT]"))
	}
	// Additional validation can be added here if needed
	return nil
}
