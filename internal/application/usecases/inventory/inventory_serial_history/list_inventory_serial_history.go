package inventory_serial_history

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	serialhistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/serial_history"
)

// ListInventorySerialHistoryRepositories groups all repository dependencies
type ListInventorySerialHistoryRepositories struct {
	InventorySerialHistory serialhistorypb.InventorySerialHistoryDomainServiceServer
}

// ListInventorySerialHistoryServices groups all business service dependencies
type ListInventorySerialHistoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListInventorySerialHistoryUseCase handles the business logic for listing inventory serial history
type ListInventorySerialHistoryUseCase struct {
	repositories ListInventorySerialHistoryRepositories
	services     ListInventorySerialHistoryServices
}

// NewListInventorySerialHistoryUseCase creates a new ListInventorySerialHistoryUseCase
func NewListInventorySerialHistoryUseCase(
	repositories ListInventorySerialHistoryRepositories,
	services ListInventorySerialHistoryServices,
) *ListInventorySerialHistoryUseCase {
	return &ListInventorySerialHistoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list inventory serial history operation
func (uc *ListInventorySerialHistoryUseCase) Execute(ctx context.Context, req *serialhistorypb.ListInventorySerialHistoryRequest) (*serialhistorypb.ListInventorySerialHistoryResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventorySerialHistory, ports.ActionList); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.errors.authorization_failed", "Authorization failed for inventory serial history [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventorySerialHistory, ports.ActionList)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.errors.authorization_failed", "Authorization failed for inventory serial history [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.errors.authorization_failed", "Authorization failed for inventory serial history [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	resp, err := uc.repositories.InventorySerialHistory.ListInventorySerialHistory(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.errors.list_failed", "Failed to retrieve inventory serial history [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *ListInventorySerialHistoryUseCase) validateInput(ctx context.Context, req *serialhistorypb.ListInventorySerialHistoryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
