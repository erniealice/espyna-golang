package inventory_attribute

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	inventoryattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_attribute"
)

// UpdateInventoryAttributeRepositories groups all repository dependencies
type UpdateInventoryAttributeRepositories struct {
	InventoryAttribute inventoryattributepb.InventoryAttributeDomainServiceServer
}

// UpdateInventoryAttributeServices groups all business service dependencies
type UpdateInventoryAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateInventoryAttributeUseCase handles the business logic for updating inventory attributes
type UpdateInventoryAttributeUseCase struct {
	repositories UpdateInventoryAttributeRepositories
	services     UpdateInventoryAttributeServices
}

// NewUpdateInventoryAttributeUseCase creates use case with grouped dependencies
func NewUpdateInventoryAttributeUseCase(
	repositories UpdateInventoryAttributeRepositories,
	services UpdateInventoryAttributeServices,
) *UpdateInventoryAttributeUseCase {
	return &UpdateInventoryAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update inventory attribute operation
func (uc *UpdateInventoryAttributeUseCase) Execute(ctx context.Context, req *inventoryattributepb.UpdateInventoryAttributeRequest) (*inventoryattributepb.UpdateInventoryAttributeResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventoryAttribute, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *UpdateInventoryAttributeUseCase) executeWithTransaction(ctx context.Context, req *inventoryattributepb.UpdateInventoryAttributeRequest) (*inventoryattributepb.UpdateInventoryAttributeResponse, error) {
	var result *inventoryattributepb.UpdateInventoryAttributeResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "inventory_attribute.errors.update_failed", "Inventory attribute update failed [DEFAULT]")
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

func (uc *UpdateInventoryAttributeUseCase) executeCore(ctx context.Context, req *inventoryattributepb.UpdateInventoryAttributeRequest) (*inventoryattributepb.UpdateInventoryAttributeResponse, error) {
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.errors.authorization_failed", "Authorization failed for inventory attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventoryAttribute, ports.ActionUpdate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.errors.authorization_failed", "Authorization failed for inventory attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.errors.authorization_failed", "Authorization failed for inventory attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if err := uc.enrichData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	existingResp, err := uc.repositories.InventoryAttribute.ReadInventoryAttribute(ctx, &inventoryattributepb.ReadInventoryAttributeRequest{Data: &inventoryattributepb.InventoryAttribute{Id: req.Data.Id}})
	if err != nil || existingResp == nil || len(existingResp.Data) == 0 {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.errors.not_found", "Inventory attribute not found for update [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	existing := existingResp.Data[0]

	if req.Data.Active == false {
		req.Data.Active = existing.Active
	}

	resp, err := uc.repositories.InventoryAttribute.UpdateInventoryAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.errors.update_failed", "Inventory attribute update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *UpdateInventoryAttributeUseCase) validateInput(ctx context.Context, req *inventoryattributepb.UpdateInventoryAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.validation.data_required", "Inventory attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.validation.id_required", "Inventory attribute ID is required [DEFAULT]"))
	}
	return nil
}

func (uc *UpdateInventoryAttributeUseCase) enrichData(attr *inventoryattributepb.InventoryAttribute) error {
	now := time.Now()
	attr.DateModified = &[]int64{now.UnixMilli()}[0]
	attr.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return nil
}
