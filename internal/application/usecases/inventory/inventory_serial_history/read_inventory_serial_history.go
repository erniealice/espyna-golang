package inventory_serial_history

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	serialhistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/serial_history"
)

// ReadInventorySerialHistoryRepositories groups all repository dependencies
type ReadInventorySerialHistoryRepositories struct {
	InventorySerialHistory serialhistorypb.InventorySerialHistoryDomainServiceServer
}

// ReadInventorySerialHistoryServices groups all business service dependencies
type ReadInventorySerialHistoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadInventorySerialHistoryUseCase handles the business logic for reading an inventory serial history entry
type ReadInventorySerialHistoryUseCase struct {
	repositories ReadInventorySerialHistoryRepositories
	services     ReadInventorySerialHistoryServices
}

// NewReadInventorySerialHistoryUseCase creates use case with grouped dependencies
func NewReadInventorySerialHistoryUseCase(
	repositories ReadInventorySerialHistoryRepositories,
	services ReadInventorySerialHistoryServices,
) *ReadInventorySerialHistoryUseCase {
	return &ReadInventorySerialHistoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read inventory serial history operation
func (uc *ReadInventorySerialHistoryUseCase) Execute(ctx context.Context, req *serialhistorypb.ReadInventorySerialHistoryRequest) (*serialhistorypb.ReadInventorySerialHistoryResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventorySerialHistory, ports.ActionRead); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.errors.authorization_failed", "Authorization failed for inventory serial history [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventorySerialHistory, ports.ActionRead)
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

	resp, err := uc.repositories.InventorySerialHistory.ReadInventorySerialHistory(ctx, req)
	if err != nil {
		errorMsg := strings.ToLower(err.Error())
		if strings.Contains(errorMsg, "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.errors.not_found", "Inventory serial history entry not found [DEFAULT]")
			return nil, errors.New(translatedError)
		}
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.errors.read_failed", "Failed to retrieve inventory serial history [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *ReadInventorySerialHistoryUseCase) validateInput(ctx context.Context, req *serialhistorypb.ReadInventorySerialHistoryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.validation.data_required", "Inventory serial history data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.validation.id_required", "Inventory serial history ID is required [DEFAULT]"))
	}
	return nil
}
