package inventory_serial

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	inventoryserialpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_serial"
)

// ReadInventorySerialRepositories groups all repository dependencies
type ReadInventorySerialRepositories struct {
	InventorySerial inventoryserialpb.InventorySerialDomainServiceServer
}

// ReadInventorySerialServices groups all business service dependencies
type ReadInventorySerialServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadInventorySerialUseCase handles the business logic for reading an inventory serial
type ReadInventorySerialUseCase struct {
	repositories ReadInventorySerialRepositories
	services     ReadInventorySerialServices
}

// NewReadInventorySerialUseCase creates use case with grouped dependencies
func NewReadInventorySerialUseCase(
	repositories ReadInventorySerialRepositories,
	services ReadInventorySerialServices,
) *ReadInventorySerialUseCase {
	return &ReadInventorySerialUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read inventory serial operation
func (uc *ReadInventorySerialUseCase) Execute(ctx context.Context, req *inventoryserialpb.ReadInventorySerialRequest) (*inventoryserialpb.ReadInventorySerialResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventorySerial, ports.ActionRead); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.errors.authorization_failed", "Authorization failed for inventory serials [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventorySerial, ports.ActionRead)
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

	resp, err := uc.repositories.InventorySerial.ReadInventorySerial(ctx, req)
	if err != nil {
		errorMsg := strings.ToLower(err.Error())
		if strings.Contains(errorMsg, "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.errors.not_found", "Inventory serial not found [DEFAULT]")
			return nil, errors.New(translatedError)
		}
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.errors.read_failed", "Failed to retrieve inventory serial [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

func (uc *ReadInventorySerialUseCase) validateInput(ctx context.Context, req *inventoryserialpb.ReadInventorySerialRequest) error {
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
