package inventory_item

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	inventoryitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_item"
)

// CreateInventoryItemRepositories groups all repository dependencies
type CreateInventoryItemRepositories struct {
	InventoryItem inventoryitempb.InventoryItemDomainServiceServer // Primary entity repository
}

// CreateInventoryItemServices groups all business service dependencies
type CreateInventoryItemServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateInventoryItemUseCase handles the business logic for creating inventory items
type CreateInventoryItemUseCase struct {
	repositories CreateInventoryItemRepositories
	services     CreateInventoryItemServices
}

// NewCreateInventoryItemUseCase creates use case with grouped dependencies
func NewCreateInventoryItemUseCase(
	repositories CreateInventoryItemRepositories,
	services CreateInventoryItemServices,
) *CreateInventoryItemUseCase {
	return &CreateInventoryItemUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create inventory item operation
func (uc *CreateInventoryItemUseCase) Execute(ctx context.Context, req *inventoryitempb.CreateInventoryItemRequest) (*inventoryitempb.CreateInventoryItemResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.InventoryItem,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes inventory item creation within a transaction
func (uc *CreateInventoryItemUseCase) executeWithTransaction(ctx context.Context, req *inventoryitempb.CreateInventoryItemRequest) (*inventoryitempb.CreateInventoryItemResponse, error) {
	var result *inventoryitempb.CreateInventoryItemResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("inventory item creation failed: %w", err)
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
func (uc *CreateInventoryItemUseCase) executeCore(ctx context.Context, req *inventoryitempb.CreateInventoryItemRequest) (*inventoryitempb.CreateInventoryItemResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_item.errors.authorization_failed", "Authorization failed for inventory items [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.InventoryItem, entityid.ActionCreate)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_item.errors.authorization_failed", "Authorization failed for inventory items [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_item.errors.authorization_failed", "Authorization failed for inventory items [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_item.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_item.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_item.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	return uc.repositories.InventoryItem.CreateInventoryItem(ctx, req)
}

// validateInput validates the input request
func (uc *CreateInventoryItemUseCase) validateInput(ctx context.Context, req *inventoryitempb.CreateInventoryItemRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_item.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_item.validation.data_required", "Inventory item data is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_item.validation.name_required", "Inventory item name is required [DEFAULT]"))
	}
	if req.Data.QuantityOnHand < 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_item.validation.quantity_on_hand_non_negative", "Quantity on hand must be zero or greater [DEFAULT]"))
	}
	if req.Data.UnitOfMeasure == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_item.validation.unit_of_measure_required", "Unit of measure is required [DEFAULT]"))
	}
	return nil
}

// enrichData adds generated fields and audit information
func (uc *CreateInventoryItemUseCase) enrichData(item *inventoryitempb.InventoryItem) error {
	now := time.Now()

	// Generate ID if not provided
	if item.Id == "" {
		item.Id = uc.services.IDGenerator.GenerateID()
	}

	// Set audit fields
	item.DateCreated = &[]int64{now.UnixMilli()}[0]
	item.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	item.DateModified = &[]int64{now.UnixMilli()}[0]
	item.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	item.Active = true

	return nil
}

// validateBusinessRules enforces business constraints for inventory items
func (uc *CreateInventoryItemUseCase) validateBusinessRules(ctx context.Context, item *inventoryitempb.InventoryItem) error {
	// Validate name length
	name := strings.TrimSpace(item.Name)
	if len(name) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_item.validation.name_min_length", "Inventory item name must be at least 2 characters long [DEFAULT]"))
	}

	if len(name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_item.validation.name_max_length", "Inventory item name cannot exceed 100 characters [DEFAULT]"))
	}

	return nil
}
