package inventory_serial

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	inventoryserialpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_serial"
)

// ListInventorySerialsRepositories groups all repository dependencies
type ListInventorySerialsRepositories struct {
	InventorySerial inventoryserialpb.InventorySerialDomainServiceServer
}

// ListInventorySerialsServices groups all business service dependencies
type ListInventorySerialsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListInventorySerialsUseCase handles the business logic for listing inventory serials
type ListInventorySerialsUseCase struct {
	repositories ListInventorySerialsRepositories
	services     ListInventorySerialsServices
}

// NewListInventorySerialsUseCase creates a new ListInventorySerialsUseCase
func NewListInventorySerialsUseCase(
	repositories ListInventorySerialsRepositories,
	services ListInventorySerialsServices,
) *ListInventorySerialsUseCase {
	return &ListInventorySerialsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list inventory serials operation
func (uc *ListInventorySerialsUseCase) Execute(ctx context.Context, req *inventoryserialpb.ListInventorySerialsRequest) (*inventoryserialpb.ListInventorySerialsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventorySerial, ports.ActionList); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.errors.authorization_failed", "Authorization failed for inventory serials [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventorySerial, ports.ActionList)
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

	resp, err := uc.repositories.InventorySerial.ListInventorySerials(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.errors.list_failed", "Failed to retrieve inventory serials [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *ListInventorySerialsUseCase) validateInput(ctx context.Context, req *inventoryserialpb.ListInventorySerialsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_serial.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
