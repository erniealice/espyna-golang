package inventory_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	inventoryattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_attribute"
)

// ReadInventoryAttributeRepositories groups all repository dependencies
type ReadInventoryAttributeRepositories struct {
	InventoryAttribute inventoryattributepb.InventoryAttributeDomainServiceServer
}

// ReadInventoryAttributeServices groups all business service dependencies
type ReadInventoryAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadInventoryAttributeUseCase handles the business logic for reading an inventory attribute
type ReadInventoryAttributeUseCase struct {
	repositories ReadInventoryAttributeRepositories
	services     ReadInventoryAttributeServices
}

// NewReadInventoryAttributeUseCase creates use case with grouped dependencies
func NewReadInventoryAttributeUseCase(
	repositories ReadInventoryAttributeRepositories,
	services ReadInventoryAttributeServices,
) *ReadInventoryAttributeUseCase {
	return &ReadInventoryAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read inventory attribute operation
func (uc *ReadInventoryAttributeUseCase) Execute(ctx context.Context, req *inventoryattributepb.ReadInventoryAttributeRequest) (*inventoryattributepb.ReadInventoryAttributeResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventoryAttribute, ports.ActionRead); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.errors.authorization_failed", "Authorization failed for inventory attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventoryAttribute, ports.ActionRead)
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

	resp, err := uc.repositories.InventoryAttribute.ReadInventoryAttribute(ctx, req)
	if err != nil {
		errorMsg := strings.ToLower(err.Error())
		if strings.Contains(errorMsg, "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.errors.not_found", "Inventory attribute not found [DEFAULT]")
			return nil, errors.New(translatedError)
		}
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.errors.read_failed", "Failed to retrieve inventory attribute [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *ReadInventoryAttributeUseCase) validateInput(ctx context.Context, req *inventoryattributepb.ReadInventoryAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.validation.data_required", "Inventory attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_attribute.validation.id_required", "Inventory attribute ID is required [DEFAULT]"))
	}
	return nil
}
