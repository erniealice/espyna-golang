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

// UpdateInventorySerialRepositories groups all repository dependencies
type UpdateInventorySerialRepositories struct {
	InventorySerial inventoryserialpb.InventorySerialDomainServiceServer
}

// UpdateInventorySerialServices groups all business service dependencies
type UpdateInventorySerialServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateInventorySerialUseCase handles the business logic for updating inventory serials
type UpdateInventorySerialUseCase struct {
	repositories UpdateInventorySerialRepositories
	services     UpdateInventorySerialServices
}

// NewUpdateInventorySerialUseCase creates use case with grouped dependencies
func NewUpdateInventorySerialUseCase(
	repositories UpdateInventorySerialRepositories,
	services UpdateInventorySerialServices,
) *UpdateInventorySerialUseCase {
	return &UpdateInventorySerialUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update inventory serial operation
func (uc *UpdateInventorySerialUseCase) Execute(ctx context.Context, req *inventoryserialpb.UpdateInventorySerialRequest) (*inventoryserialpb.UpdateInventorySerialResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventorySerial, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *UpdateInventorySerialUseCase) executeWithTransaction(ctx context.Context, req *inventoryserialpb.UpdateInventorySerialRequest) (*inventoryserialpb.UpdateInventorySerialResponse, error) {
	var result *inventoryserialpb.UpdateInventorySerialResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "inventory_serial.errors.update_failed", "Inventory serial update failed [DEFAULT]")
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

func (uc *UpdateInventorySerialUseCase) executeCore(ctx context.Context, req *inventoryserialpb.UpdateInventorySerialRequest) (*inventoryserialpb.UpdateInventorySerialResponse, error) {
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.errors.authorization_failed", "Authorization failed for inventory serials [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventorySerial, ports.ActionUpdate)
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

	existingResp, err := uc.repositories.InventorySerial.ReadInventorySerial(ctx, &inventoryserialpb.ReadInventorySerialRequest{Data: &inventoryserialpb.InventorySerial{Id: req.Data.Id}})
	if err != nil || existingResp == nil || len(existingResp.Data) == 0 {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.errors.not_found", "Inventory serial not found for update [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	existingSerial := existingResp.Data[0]

	if req.Data.Active == false {
		req.Data.Active = existingSerial.Active
	}

	resp, err := uc.repositories.InventorySerial.UpdateInventorySerial(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.errors.update_failed", "Inventory serial update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *UpdateInventorySerialUseCase) validateInput(ctx context.Context, req *inventoryserialpb.UpdateInventorySerialRequest) error {
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

func (uc *UpdateInventorySerialUseCase) enrichData(serial *inventoryserialpb.InventorySerial) error {
	now := time.Now()
	serial.DateModified = &[]int64{now.UnixMilli()}[0]
	serial.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return nil
}
