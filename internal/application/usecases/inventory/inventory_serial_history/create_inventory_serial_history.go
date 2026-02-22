package inventory_serial_history

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	serialhistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/serial_history"
)

// CreateInventorySerialHistoryRepositories groups all repository dependencies
type CreateInventorySerialHistoryRepositories struct {
	InventorySerialHistory serialhistorypb.InventorySerialHistoryDomainServiceServer
}

// CreateInventorySerialHistoryServices groups all business service dependencies
type CreateInventorySerialHistoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateInventorySerialHistoryUseCase handles the business logic for creating inventory serial history entries
type CreateInventorySerialHistoryUseCase struct {
	repositories CreateInventorySerialHistoryRepositories
	services     CreateInventorySerialHistoryServices
}

// NewCreateInventorySerialHistoryUseCase creates use case with grouped dependencies
func NewCreateInventorySerialHistoryUseCase(
	repositories CreateInventorySerialHistoryRepositories,
	services CreateInventorySerialHistoryServices,
) *CreateInventorySerialHistoryUseCase {
	return &CreateInventorySerialHistoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create inventory serial history operation
func (uc *CreateInventorySerialHistoryUseCase) Execute(ctx context.Context, req *serialhistorypb.CreateInventorySerialHistoryRequest) (*serialhistorypb.CreateInventorySerialHistoryResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventorySerialHistory, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *CreateInventorySerialHistoryUseCase) executeWithTransaction(ctx context.Context, req *serialhistorypb.CreateInventorySerialHistoryRequest) (*serialhistorypb.CreateInventorySerialHistoryResponse, error) {
	var result *serialhistorypb.CreateInventorySerialHistoryResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("inventory serial history creation failed: %w", err)
		}
		result = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (uc *CreateInventorySerialHistoryUseCase) executeCore(ctx context.Context, req *serialhistorypb.CreateInventorySerialHistoryRequest) (*serialhistorypb.CreateInventorySerialHistoryResponse, error) {
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.errors.authorization_failed", "Authorization failed for inventory serial history [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventorySerialHistory, ports.ActionCreate)
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

	if err := uc.enrichData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return uc.repositories.InventorySerialHistory.CreateInventorySerialHistory(ctx, req)
}

func (uc *CreateInventorySerialHistoryUseCase) validateInput(ctx context.Context, req *serialhistorypb.CreateInventorySerialHistoryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.validation.data_required", "Inventory serial history data is required [DEFAULT]"))
	}
	if req.Data.InventorySerialId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.validation.inventory_serial_id_required", "Inventory serial ID is required [DEFAULT]"))
	}
	if req.Data.InventoryItemId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.validation.inventory_item_id_required", "Inventory item ID is required [DEFAULT]"))
	}
	if req.Data.FromStatus == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.validation.from_status_required", "From status is required [DEFAULT]"))
	}
	if req.Data.ToStatus == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.validation.to_status_required", "To status is required [DEFAULT]"))
	}
	if req.Data.ReferenceType == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.validation.reference_type_required", "Reference type is required [DEFAULT]"))
	}
	return nil
}

// enrichData adds generated fields — date_created ONLY (immutable audit trail, no date_modified)
func (uc *CreateInventorySerialHistoryUseCase) enrichData(history *serialhistorypb.InventorySerialHistory) error {
	now := time.Now()

	if history.Id == "" {
		history.Id = uc.services.IDService.GenerateID()
	}

	// Immutable audit trail: set date_created ONLY (no date_modified)
	history.DateCreated = &[]int64{now.UnixMilli()}[0]
	history.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

func (uc *CreateInventorySerialHistoryUseCase) validateBusinessRules(ctx context.Context, history *serialhistorypb.InventorySerialHistory) error {
	// Validate that from_status and to_status are different
	if history.FromStatus == history.ToStatus {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.validation.status_must_change", "From status and to status must be different [DEFAULT]"))
	}

	// Validate reference type
	validReferenceTypes := map[string]bool{"sale": true, "manual": true, "return": true, "damage": true, "repair": true, "transfer": true}
	if !validReferenceTypes[history.ReferenceType] {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial_history.validation.invalid_reference_type", "Reference type must be sale, manual, return, damage, repair, or transfer [DEFAULT]"))
	}

	return nil
}
