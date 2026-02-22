package inventory_transaction

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	inventorytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_transaction"
)

// UpdateInventoryTransactionRepositories groups all repository dependencies
type UpdateInventoryTransactionRepositories struct {
	InventoryTransaction inventorytransactionpb.InventoryTransactionDomainServiceServer
}

// UpdateInventoryTransactionServices groups all business service dependencies
type UpdateInventoryTransactionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateInventoryTransactionUseCase handles the business logic for updating inventory transactions
type UpdateInventoryTransactionUseCase struct {
	repositories UpdateInventoryTransactionRepositories
	services     UpdateInventoryTransactionServices
}

// NewUpdateInventoryTransactionUseCase creates use case with grouped dependencies
func NewUpdateInventoryTransactionUseCase(
	repositories UpdateInventoryTransactionRepositories,
	services UpdateInventoryTransactionServices,
) *UpdateInventoryTransactionUseCase {
	return &UpdateInventoryTransactionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update inventory transaction operation
func (uc *UpdateInventoryTransactionUseCase) Execute(ctx context.Context, req *inventorytransactionpb.UpdateInventoryTransactionRequest) (*inventorytransactionpb.UpdateInventoryTransactionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventoryTransaction, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *UpdateInventoryTransactionUseCase) executeWithTransaction(ctx context.Context, req *inventorytransactionpb.UpdateInventoryTransactionRequest) (*inventorytransactionpb.UpdateInventoryTransactionResponse, error) {
	var result *inventorytransactionpb.UpdateInventoryTransactionResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "inventory_transaction.errors.update_failed", "Inventory transaction update failed [DEFAULT]")
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

func (uc *UpdateInventoryTransactionUseCase) executeCore(ctx context.Context, req *inventorytransactionpb.UpdateInventoryTransactionRequest) (*inventorytransactionpb.UpdateInventoryTransactionResponse, error) {
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.errors.authorization_failed", "Authorization failed for inventory transactions [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventoryTransaction, ports.ActionUpdate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.errors.authorization_failed", "Authorization failed for inventory transactions [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.errors.authorization_failed", "Authorization failed for inventory transactions [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if err := uc.enrichData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	existingResp, err := uc.repositories.InventoryTransaction.ReadInventoryTransaction(ctx, &inventorytransactionpb.ReadInventoryTransactionRequest{Data: &inventorytransactionpb.InventoryTransaction{Id: req.Data.Id}})
	if err != nil || existingResp == nil || len(existingResp.Data) == 0 {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.errors.not_found", "Inventory transaction not found for update [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	existing := existingResp.Data[0]

	if req.Data.Active == false {
		req.Data.Active = existing.Active
	}

	resp, err := uc.repositories.InventoryTransaction.UpdateInventoryTransaction(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.errors.update_failed", "Inventory transaction update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *UpdateInventoryTransactionUseCase) validateInput(ctx context.Context, req *inventorytransactionpb.UpdateInventoryTransactionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.validation.data_required", "Inventory transaction data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.validation.id_required", "Inventory transaction ID is required [DEFAULT]"))
	}
	return nil
}

func (uc *UpdateInventoryTransactionUseCase) enrichData(transaction *inventorytransactionpb.InventoryTransaction) error {
	now := time.Now()
	transaction.DateModified = &[]int64{now.UnixMilli()}[0]
	transaction.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return nil
}
