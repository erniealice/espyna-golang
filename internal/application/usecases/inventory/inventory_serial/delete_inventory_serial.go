package inventory_serial

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	inventoryserialpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_serial"
)

// DeleteInventorySerialRepositories groups all repository dependencies
type DeleteInventorySerialRepositories struct {
	InventorySerial inventoryserialpb.InventorySerialDomainServiceServer
}

// DeleteInventorySerialServices groups all business service dependencies
type DeleteInventorySerialServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteInventorySerialUseCase handles the business logic for deleting inventory serials
type DeleteInventorySerialUseCase struct {
	repositories DeleteInventorySerialRepositories
	services     DeleteInventorySerialServices
}

// NewDeleteInventorySerialUseCase creates a new DeleteInventorySerialUseCase
func NewDeleteInventorySerialUseCase(
	repositories DeleteInventorySerialRepositories,
	services DeleteInventorySerialServices,
) *DeleteInventorySerialUseCase {
	return &DeleteInventorySerialUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete inventory serial operation
func (uc *DeleteInventorySerialUseCase) Execute(ctx context.Context, req *inventoryserialpb.DeleteInventorySerialRequest) (*inventoryserialpb.DeleteInventorySerialResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventorySerial, ports.ActionDelete); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *DeleteInventorySerialUseCase) executeWithTransaction(ctx context.Context, req *inventoryserialpb.DeleteInventorySerialRequest) (*inventoryserialpb.DeleteInventorySerialResponse, error) {
	var result *inventoryserialpb.DeleteInventorySerialResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return result, nil
}

func (uc *DeleteInventorySerialUseCase) executeCore(ctx context.Context, req *inventoryserialpb.DeleteInventorySerialRequest) (*inventoryserialpb.DeleteInventorySerialResponse, error) {
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.errors.authorization_failed", "Authorization failed for inventory serials [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventorySerial, ports.ActionDelete)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.errors.authorization_failed", "Authorization failed for inventory serials [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.errors.authorization_failed", "Authorization failed for inventory serials [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	resp, err := uc.repositories.InventorySerial.DeleteInventorySerial(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.errors.deletion_failed", "Inventory serial deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *DeleteInventorySerialUseCase) validateInput(ctx context.Context, req *inventoryserialpb.DeleteInventorySerialRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.validation.data_required", "Inventory serial data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.validation.id_required", "Inventory serial ID is required [DEFAULT]"))
	}
	return nil
}
