package inventory_serial

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	inventoryserialpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_serial"
)

// CreateInventorySerialRepositories groups all repository dependencies
type CreateInventorySerialRepositories struct {
	InventorySerial inventoryserialpb.InventorySerialDomainServiceServer
}

// CreateInventorySerialServices groups all business service dependencies
type CreateInventorySerialServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateInventorySerialUseCase handles the business logic for creating inventory serials
type CreateInventorySerialUseCase struct {
	repositories CreateInventorySerialRepositories
	services     CreateInventorySerialServices
}

// NewCreateInventorySerialUseCase creates use case with grouped dependencies
func NewCreateInventorySerialUseCase(
	repositories CreateInventorySerialRepositories,
	services CreateInventorySerialServices,
) *CreateInventorySerialUseCase {
	return &CreateInventorySerialUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create inventory serial operation
func (uc *CreateInventorySerialUseCase) Execute(ctx context.Context, req *inventoryserialpb.CreateInventorySerialRequest) (*inventoryserialpb.CreateInventorySerialResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventorySerial, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *CreateInventorySerialUseCase) executeWithTransaction(ctx context.Context, req *inventoryserialpb.CreateInventorySerialRequest) (*inventoryserialpb.CreateInventorySerialResponse, error) {
	var result *inventoryserialpb.CreateInventorySerialResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("inventory serial creation failed: %w", err)
		}
		result = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (uc *CreateInventorySerialUseCase) executeCore(ctx context.Context, req *inventoryserialpb.CreateInventorySerialRequest) (*inventoryserialpb.CreateInventorySerialResponse, error) {
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.errors.authorization_failed", "Authorization failed for inventory serials [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventorySerial, ports.ActionCreate)
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

	if err := uc.enrichData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return uc.repositories.InventorySerial.CreateInventorySerial(ctx, req)
}

func (uc *CreateInventorySerialUseCase) validateInput(ctx context.Context, req *inventoryserialpb.CreateInventorySerialRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.validation.data_required", "Inventory serial data is required [DEFAULT]"))
	}
	if req.Data.SerialNumber == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.validation.serial_number_required", "Serial number is required [DEFAULT]"))
	}
	if req.Data.InventoryItemId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.validation.inventory_item_id_required", "Inventory item ID is required [DEFAULT]"))
	}
	if req.Data.Status == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.validation.status_required", "Status is required [DEFAULT]"))
	}
	return nil
}

func (uc *CreateInventorySerialUseCase) enrichData(serial *inventoryserialpb.InventorySerial) error {
	now := time.Now()

	if serial.Id == "" {
		serial.Id = uc.services.IDService.GenerateID()
	}

	serial.DateCreated = &[]int64{now.UnixMilli()}[0]
	serial.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	serial.DateModified = &[]int64{now.UnixMilli()}[0]
	serial.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	serial.Active = true

	return nil
}

func (uc *CreateInventorySerialUseCase) validateBusinessRules(ctx context.Context, serial *inventoryserialpb.InventorySerial) error {
	// Validate status values
	validStatuses := map[string]bool{"available": true, "reserved": true, "sold": true, "damaged": true, "returned": true}
	if !validStatuses[serial.Status] {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.validation.invalid_status", "Status must be available, reserved, sold, damaged, or returned [DEFAULT]"))
	}
	return nil
}
