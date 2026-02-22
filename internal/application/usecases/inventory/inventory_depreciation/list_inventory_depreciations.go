package inventory_depreciation

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	inventorydepreciationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_depreciation"
)

// ListInventoryDepreciationsRepositories groups all repository dependencies
type ListInventoryDepreciationsRepositories struct {
	InventoryDepreciation inventorydepreciationpb.InventoryDepreciationDomainServiceServer
}

// ListInventoryDepreciationsServices groups all business service dependencies
type ListInventoryDepreciationsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListInventoryDepreciationsUseCase handles the business logic for listing inventory depreciations
type ListInventoryDepreciationsUseCase struct {
	repositories ListInventoryDepreciationsRepositories
	services     ListInventoryDepreciationsServices
}

// NewListInventoryDepreciationsUseCase creates a new ListInventoryDepreciationsUseCase
func NewListInventoryDepreciationsUseCase(
	repositories ListInventoryDepreciationsRepositories,
	services ListInventoryDepreciationsServices,
) *ListInventoryDepreciationsUseCase {
	return &ListInventoryDepreciationsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list inventory depreciations operation
func (uc *ListInventoryDepreciationsUseCase) Execute(ctx context.Context, req *inventorydepreciationpb.ListInventoryDepreciationsRequest) (*inventorydepreciationpb.ListInventoryDepreciationsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventoryDepreciation, ports.ActionList); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.errors.authorization_failed", "Authorization failed for inventory depreciations [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventoryDepreciation, ports.ActionList)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.errors.authorization_failed", "Authorization failed for inventory depreciations [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.errors.authorization_failed", "Authorization failed for inventory depreciations [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	resp, err := uc.repositories.InventoryDepreciation.ListInventoryDepreciations(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.errors.list_failed", "Failed to retrieve inventory depreciations [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *ListInventoryDepreciationsUseCase) validateInput(ctx context.Context, req *inventorydepreciationpb.ListInventoryDepreciationsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
