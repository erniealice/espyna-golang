package inventory_transaction

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	inventorytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_transaction"
)

// DeleteInventoryTransactionRepositories groups all repository dependencies
type DeleteInventoryTransactionRepositories struct {
	InventoryTransaction inventorytransactionpb.InventoryTransactionDomainServiceServer
}

// DeleteInventoryTransactionServices groups all business service dependencies
type DeleteInventoryTransactionServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteInventoryTransactionUseCase handles the business logic for deleting inventory transactions
type DeleteInventoryTransactionUseCase struct {
	repositories DeleteInventoryTransactionRepositories
	services     DeleteInventoryTransactionServices
}

// NewDeleteInventoryTransactionUseCase creates a new DeleteInventoryTransactionUseCase
func NewDeleteInventoryTransactionUseCase(
	repositories DeleteInventoryTransactionRepositories,
	services DeleteInventoryTransactionServices,
) *DeleteInventoryTransactionUseCase {
	return &DeleteInventoryTransactionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete inventory transaction operation
func (uc *DeleteInventoryTransactionUseCase) Execute(ctx context.Context, req *inventorytransactionpb.DeleteInventoryTransactionRequest) (*inventorytransactionpb.DeleteInventoryTransactionResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.InventoryTransaction,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *DeleteInventoryTransactionUseCase) executeWithTransaction(ctx context.Context, req *inventorytransactionpb.DeleteInventoryTransactionRequest) (*inventorytransactionpb.DeleteInventoryTransactionResponse, error) {
	var result *inventorytransactionpb.DeleteInventoryTransactionResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_transaction.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return result, nil
}

func (uc *DeleteInventoryTransactionUseCase) executeCore(ctx context.Context, req *inventorytransactionpb.DeleteInventoryTransactionRequest) (*inventorytransactionpb.DeleteInventoryTransactionResponse, error) {
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_transaction.errors.authorization_failed", "Authorization failed for inventory transactions [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.InventoryTransaction, entityid.ActionDelete)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_transaction.errors.authorization_failed", "Authorization failed for inventory transactions [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_transaction.errors.authorization_failed", "Authorization failed for inventory transactions [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_transaction.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	resp, err := uc.repositories.InventoryTransaction.DeleteInventoryTransaction(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_transaction.errors.deletion_failed", "Inventory transaction deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *DeleteInventoryTransactionUseCase) validateInput(ctx context.Context, req *inventorytransactionpb.DeleteInventoryTransactionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_transaction.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_transaction.validation.data_required", "Inventory transaction data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_transaction.validation.id_required", "Inventory transaction ID is required [DEFAULT]"))
	}
	return nil
}
