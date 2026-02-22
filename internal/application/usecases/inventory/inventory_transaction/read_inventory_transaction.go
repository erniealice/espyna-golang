package inventory_transaction

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	inventorytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_transaction"
)

// ReadInventoryTransactionRepositories groups all repository dependencies
type ReadInventoryTransactionRepositories struct {
	InventoryTransaction inventorytransactionpb.InventoryTransactionDomainServiceServer
}

// ReadInventoryTransactionServices groups all business service dependencies
type ReadInventoryTransactionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadInventoryTransactionUseCase handles the business logic for reading an inventory transaction
type ReadInventoryTransactionUseCase struct {
	repositories ReadInventoryTransactionRepositories
	services     ReadInventoryTransactionServices
}

// NewReadInventoryTransactionUseCase creates use case with grouped dependencies
func NewReadInventoryTransactionUseCase(
	repositories ReadInventoryTransactionRepositories,
	services ReadInventoryTransactionServices,
) *ReadInventoryTransactionUseCase {
	return &ReadInventoryTransactionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read inventory transaction operation
func (uc *ReadInventoryTransactionUseCase) Execute(ctx context.Context, req *inventorytransactionpb.ReadInventoryTransactionRequest) (*inventorytransactionpb.ReadInventoryTransactionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventoryTransaction, ports.ActionRead); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.errors.authorization_failed", "Authorization failed for inventory transactions [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventoryTransaction, ports.ActionRead)
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

	resp, err := uc.repositories.InventoryTransaction.ReadInventoryTransaction(ctx, req)
	if err != nil {
		errorMsg := strings.ToLower(err.Error())
		if strings.Contains(errorMsg, "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.errors.not_found", "Inventory transaction not found [DEFAULT]")
			return nil, errors.New(translatedError)
		}
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_transaction.errors.read_failed", "Failed to retrieve inventory transaction [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *ReadInventoryTransactionUseCase) validateInput(ctx context.Context, req *inventorytransactionpb.ReadInventoryTransactionRequest) error {
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
