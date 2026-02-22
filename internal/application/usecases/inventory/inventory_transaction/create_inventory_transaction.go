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

// CreateInventoryTransactionRepositories groups all repository dependencies
type CreateInventoryTransactionRepositories struct {
	InventoryTransaction inventorytransactionpb.InventoryTransactionDomainServiceServer
}

// CreateInventoryTransactionServices groups all business service dependencies
type CreateInventoryTransactionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateInventoryTransactionUseCase handles the business logic for creating inventory transactions
type CreateInventoryTransactionUseCase struct {
	repositories CreateInventoryTransactionRepositories
	services     CreateInventoryTransactionServices
}

// NewCreateInventoryTransactionUseCase creates use case with grouped dependencies
func NewCreateInventoryTransactionUseCase(
	repositories CreateInventoryTransactionRepositories,
	services CreateInventoryTransactionServices,
) *CreateInventoryTransactionUseCase {
	return &CreateInventoryTransactionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create inventory transaction operation
func (uc *CreateInventoryTransactionUseCase) Execute(ctx context.Context, req *inventorytransactionpb.CreateInventoryTransactionRequest) (*inventorytransactionpb.CreateInventoryTransactionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventoryTransaction, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *CreateInventoryTransactionUseCase) executeWithTransaction(ctx context.Context, req *inventorytransactionpb.CreateInventoryTransactionRequest) (*inventorytransactionpb.CreateInventoryTransactionResponse, error) {
	var result *inventorytransactionpb.CreateInventoryTransactionResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("inventory transaction creation failed: %w", err)
		}
		result = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (uc *CreateInventoryTransactionUseCase) executeCore(ctx context.Context, req *inventorytransactionpb.CreateInventoryTransactionRequest) (*inventorytransactionpb.CreateInventoryTransactionResponse, error) {
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.errors.authorization_failed", "Authorization failed for inventory transactions [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventoryTransaction, ports.ActionCreate)
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

	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return uc.repositories.InventoryTransaction.CreateInventoryTransaction(ctx, req)
}

func (uc *CreateInventoryTransactionUseCase) validateInput(ctx context.Context, req *inventorytransactionpb.CreateInventoryTransactionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.validation.data_required", "Inventory transaction data is required [DEFAULT]"))
	}
	if req.Data.InventoryItemId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.validation.inventory_item_id_required", "Inventory item ID is required [DEFAULT]"))
	}
	if req.Data.TransactionType == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.validation.transaction_type_required", "Transaction type is required [DEFAULT]"))
	}
	if req.Data.Quantity == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.validation.quantity_non_zero", "Quantity must not be zero [DEFAULT]"))
	}
	return nil
}

func (uc *CreateInventoryTransactionUseCase) enrichData(transaction *inventorytransactionpb.InventoryTransaction) error {
	now := time.Now()

	if transaction.Id == "" {
		transaction.Id = uc.services.IDService.GenerateID()
	}

	transaction.DateCreated = &[]int64{now.UnixMilli()}[0]
	transaction.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	transaction.DateModified = &[]int64{now.UnixMilli()}[0]
	transaction.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	transaction.Active = true

	// Set transaction date if not provided
	if transaction.TransactionDate == nil {
		transaction.TransactionDate = &[]int64{now.UnixMilli()}[0]
		transaction.TransactionDateString = &[]string{now.Format(time.RFC3339)}[0]
	}

	return nil
}

func (uc *CreateInventoryTransactionUseCase) validateBusinessRules(ctx context.Context, transaction *inventorytransactionpb.InventoryTransaction) error {
	// Validate transaction type
	validTypes := map[string]bool{"receipt": true, "issue": true, "transfer": true, "adjustment": true, "return": true}
	if !validTypes[transaction.TransactionType] {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.validation.invalid_transaction_type", "Transaction type must be receipt, issue, transfer, adjustment, or return [DEFAULT]"))
	}
	return nil
}
