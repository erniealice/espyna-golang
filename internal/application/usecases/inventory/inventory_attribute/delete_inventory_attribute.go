package inventory_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	inventoryattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_attribute"
)

// DeleteInventoryAttributeRepositories groups all repository dependencies
type DeleteInventoryAttributeRepositories struct {
	InventoryAttribute inventoryattributepb.InventoryAttributeDomainServiceServer
}

// DeleteInventoryAttributeServices groups all business service dependencies
type DeleteInventoryAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteInventoryAttributeUseCase handles the business logic for deleting inventory attributes
type DeleteInventoryAttributeUseCase struct {
	repositories DeleteInventoryAttributeRepositories
	services     DeleteInventoryAttributeServices
}

// NewDeleteInventoryAttributeUseCase creates a new DeleteInventoryAttributeUseCase
func NewDeleteInventoryAttributeUseCase(
	repositories DeleteInventoryAttributeRepositories,
	services DeleteInventoryAttributeServices,
) *DeleteInventoryAttributeUseCase {
	return &DeleteInventoryAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete inventory attribute operation
func (uc *DeleteInventoryAttributeUseCase) Execute(ctx context.Context, req *inventoryattributepb.DeleteInventoryAttributeRequest) (*inventoryattributepb.DeleteInventoryAttributeResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventoryAttribute, ports.ActionDelete); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *DeleteInventoryAttributeUseCase) executeWithTransaction(ctx context.Context, req *inventoryattributepb.DeleteInventoryAttributeRequest) (*inventoryattributepb.DeleteInventoryAttributeResponse, error) {
	var result *inventoryattributepb.DeleteInventoryAttributeResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return result, nil
}

func (uc *DeleteInventoryAttributeUseCase) executeCore(ctx context.Context, req *inventoryattributepb.DeleteInventoryAttributeRequest) (*inventoryattributepb.DeleteInventoryAttributeResponse, error) {
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.errors.authorization_failed", "Authorization failed for inventory attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventoryAttribute, ports.ActionDelete)
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

	resp, err := uc.repositories.InventoryAttribute.DeleteInventoryAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.errors.deletion_failed", "Inventory attribute deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *DeleteInventoryAttributeUseCase) validateInput(ctx context.Context, req *inventoryattributepb.DeleteInventoryAttributeRequest) error {
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
