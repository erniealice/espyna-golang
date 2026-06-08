package inventory_serial

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	inventoryserialpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_serial"
)

// ReadInventorySerialRepositories groups all repository dependencies
type ReadInventorySerialRepositories struct {
	InventorySerial inventoryserialpb.InventorySerialDomainServiceServer
}

// ReadInventorySerialServices groups all business service dependencies
type ReadInventorySerialServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.InventorySerial, entityid.ActionRead); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_serial.errors.authorization_failed", "Authorization failed for inventory serials [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.InventorySerial, entityid.ActionRead)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_serial.errors.authorization_failed", "Authorization failed for inventory serials [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_serial.errors.authorization_failed", "Authorization failed for inventory serials [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_serial.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	resp, err := uc.repositories.InventorySerial.ReadInventorySerial(ctx, req)
	if err != nil {
		errorMsg := strings.ToLower(err.Error())
		if strings.Contains(errorMsg, "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_serial.errors.not_found", "Inventory serial not found [DEFAULT]")
			return nil, errors.New(translatedError)
		}
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_serial.errors.read_failed", "Failed to retrieve inventory serial [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

func (uc *ReadInventorySerialUseCase) validateInput(ctx context.Context, req *inventoryserialpb.ReadInventorySerialRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_serial.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_serial.validation.data_required", "Inventory serial data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_serial.validation.id_required", "Inventory serial ID is required [DEFAULT]"))
	}
	return nil
}
