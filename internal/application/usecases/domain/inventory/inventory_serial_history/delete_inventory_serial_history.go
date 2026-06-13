package inventory_serial_history

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	serialhistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/serial_history"
)

// DeleteInventorySerialHistoryRepositories groups all repository dependencies
type DeleteInventorySerialHistoryRepositories struct {
	InventorySerialHistory serialhistorypb.InventorySerialHistoryDomainServiceServer
}

// DeleteInventorySerialHistoryServices groups all business service dependencies
type DeleteInventorySerialHistoryServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteInventorySerialHistoryUseCase handles the business logic for deleting inventory serial history entries
type DeleteInventorySerialHistoryUseCase struct {
	repositories DeleteInventorySerialHistoryRepositories
	services     DeleteInventorySerialHistoryServices
}

// NewDeleteInventorySerialHistoryUseCase creates a new DeleteInventorySerialHistoryUseCase
func NewDeleteInventorySerialHistoryUseCase(
	repositories DeleteInventorySerialHistoryRepositories,
	services DeleteInventorySerialHistoryServices,
) *DeleteInventorySerialHistoryUseCase {
	return &DeleteInventorySerialHistoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete inventory serial history operation
func (uc *DeleteInventorySerialHistoryUseCase) Execute(ctx context.Context, req *serialhistorypb.DeleteInventorySerialHistoryRequest) (*serialhistorypb.DeleteInventorySerialHistoryResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.InventorySerialHistory,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *DeleteInventorySerialHistoryUseCase) executeWithTransaction(ctx context.Context, req *serialhistorypb.DeleteInventorySerialHistoryRequest) (*serialhistorypb.DeleteInventorySerialHistoryResponse, error) {
	var result *serialhistorypb.DeleteInventorySerialHistoryResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_serial_history.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return result, nil
}

func (uc *DeleteInventorySerialHistoryUseCase) executeCore(ctx context.Context, req *serialhistorypb.DeleteInventorySerialHistoryRequest) (*serialhistorypb.DeleteInventorySerialHistoryResponse, error) {
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_serial_history.errors.authorization_failed", "Authorization failed for inventory serial history [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.InventorySerialHistory, entityid.ActionDelete)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_serial_history.errors.authorization_failed", "Authorization failed for inventory serial history [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_serial_history.errors.authorization_failed", "Authorization failed for inventory serial history [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_serial_history.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	resp, err := uc.repositories.InventorySerialHistory.DeleteInventorySerialHistory(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_serial_history.errors.deletion_failed", "Inventory serial history deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *DeleteInventorySerialHistoryUseCase) validateInput(ctx context.Context, req *serialhistorypb.DeleteInventorySerialHistoryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_serial_history.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_serial_history.validation.data_required", "Inventory serial history data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_serial_history.validation.id_required", "Inventory serial history ID is required [DEFAULT]"))
	}
	return nil
}
