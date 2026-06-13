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

// ListInventoryTransactionsRepositories groups all repository dependencies
type ListInventoryTransactionsRepositories struct {
	InventoryTransaction inventorytransactionpb.InventoryTransactionDomainServiceServer
}

// ListInventoryTransactionsServices groups all business service dependencies
type ListInventoryTransactionsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListInventoryTransactionsUseCase handles the business logic for listing inventory transactions
type ListInventoryTransactionsUseCase struct {
	repositories ListInventoryTransactionsRepositories
	services     ListInventoryTransactionsServices
}

// NewListInventoryTransactionsUseCase creates a new ListInventoryTransactionsUseCase
func NewListInventoryTransactionsUseCase(
	repositories ListInventoryTransactionsRepositories,
	services ListInventoryTransactionsServices,
) *ListInventoryTransactionsUseCase {
	return &ListInventoryTransactionsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list inventory transactions operation
func (uc *ListInventoryTransactionsUseCase) Execute(ctx context.Context, req *inventorytransactionpb.ListInventoryTransactionsRequest) (*inventorytransactionpb.ListInventoryTransactionsResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.InventoryTransaction,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_transaction.errors.authorization_failed", "Authorization failed for inventory transactions [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.InventoryTransaction, entityid.ActionList)
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

	resp, err := uc.repositories.InventoryTransaction.ListInventoryTransactions(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_transaction.errors.list_failed", "Failed to retrieve inventory transactions [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *ListInventoryTransactionsUseCase) validateInput(ctx context.Context, req *inventorytransactionpb.ListInventoryTransactionsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_transaction.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
