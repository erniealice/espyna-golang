package inventory_attribute

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	inventoryattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_attribute"
)

// CreateInventoryAttributeRepositories groups all repository dependencies
type CreateInventoryAttributeRepositories struct {
	InventoryAttribute inventoryattributepb.InventoryAttributeDomainServiceServer
}

// CreateInventoryAttributeServices groups all business service dependencies
type CreateInventoryAttributeServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateInventoryAttributeUseCase handles the business logic for creating inventory attributes
type CreateInventoryAttributeUseCase struct {
	repositories CreateInventoryAttributeRepositories
	services     CreateInventoryAttributeServices
}

// NewCreateInventoryAttributeUseCase creates use case with grouped dependencies
func NewCreateInventoryAttributeUseCase(
	repositories CreateInventoryAttributeRepositories,
	services CreateInventoryAttributeServices,
) *CreateInventoryAttributeUseCase {
	return &CreateInventoryAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create inventory attribute operation
func (uc *CreateInventoryAttributeUseCase) Execute(ctx context.Context, req *inventoryattributepb.CreateInventoryAttributeRequest) (*inventoryattributepb.CreateInventoryAttributeResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.InventoryAttribute,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *CreateInventoryAttributeUseCase) executeWithTransaction(ctx context.Context, req *inventoryattributepb.CreateInventoryAttributeRequest) (*inventoryattributepb.CreateInventoryAttributeResponse, error) {
	var result *inventoryattributepb.CreateInventoryAttributeResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("inventory attribute creation failed: %w", err)
		}
		result = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (uc *CreateInventoryAttributeUseCase) executeCore(ctx context.Context, req *inventoryattributepb.CreateInventoryAttributeRequest) (*inventoryattributepb.CreateInventoryAttributeResponse, error) {
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_attribute.errors.authorization_failed", "Authorization failed for inventory attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.InventoryAttribute, entityid.ActionCreate)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_attribute.errors.authorization_failed", "Authorization failed for inventory attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_attribute.errors.authorization_failed", "Authorization failed for inventory attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if err := uc.enrichData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_attribute.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return uc.repositories.InventoryAttribute.CreateInventoryAttribute(ctx, req)
}

func (uc *CreateInventoryAttributeUseCase) validateInput(ctx context.Context, req *inventoryattributepb.CreateInventoryAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_attribute.validation.data_required", "Inventory attribute data is required [DEFAULT]"))
	}
	if req.Data.InventoryItemId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_attribute.validation.inventory_item_id_required", "Inventory item ID is required [DEFAULT]"))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_attribute.validation.attribute_id_required", "Attribute ID is required [DEFAULT]"))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_attribute.validation.value_required", "Attribute value is required [DEFAULT]"))
	}
	return nil
}

func (uc *CreateInventoryAttributeUseCase) enrichData(attr *inventoryattributepb.InventoryAttribute) error {
	now := time.Now()

	if attr.Id == "" {
		attr.Id = uc.services.IDGenerator.GenerateID()
	}

	attr.DateCreated = &[]int64{now.UnixMilli()}[0]
	attr.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	attr.DateModified = &[]int64{now.UnixMilli()}[0]
	attr.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	attr.Active = true

	return nil
}

func (uc *CreateInventoryAttributeUseCase) validateBusinessRules(ctx context.Context, attr *inventoryattributepb.InventoryAttribute) error {
	// Additional business rule validation can be added here
	return nil
}
