package inventory_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	inventoryattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_attribute"
)

// ListInventoryAttributesRepositories groups all repository dependencies
type ListInventoryAttributesRepositories struct {
	InventoryAttribute inventoryattributepb.InventoryAttributeDomainServiceServer
}

// ListInventoryAttributesServices groups all business service dependencies
type ListInventoryAttributesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListInventoryAttributesUseCase handles the business logic for listing inventory attributes
type ListInventoryAttributesUseCase struct {
	repositories ListInventoryAttributesRepositories
	services     ListInventoryAttributesServices
}

// NewListInventoryAttributesUseCase creates a new ListInventoryAttributesUseCase
func NewListInventoryAttributesUseCase(
	repositories ListInventoryAttributesRepositories,
	services ListInventoryAttributesServices,
) *ListInventoryAttributesUseCase {
	return &ListInventoryAttributesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list inventory attributes operation
func (uc *ListInventoryAttributesUseCase) Execute(ctx context.Context, req *inventoryattributepb.ListInventoryAttributesRequest) (*inventoryattributepb.ListInventoryAttributesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventoryAttribute, ports.ActionList); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.errors.authorization_failed", "Authorization failed for inventory attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventoryAttribute, ports.ActionList)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.errors.authorization_failed", "Authorization failed for inventory attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.errors.authorization_failed", "Authorization failed for inventory attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	resp, err := uc.repositories.InventoryAttribute.ListInventoryAttributes(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.errors.list_failed", "Failed to retrieve inventory attributes [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *ListInventoryAttributesUseCase) validateInput(ctx context.Context, req *inventoryattributepb.ListInventoryAttributesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
