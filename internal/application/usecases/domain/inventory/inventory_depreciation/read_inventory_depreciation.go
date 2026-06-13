package inventory_depreciation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	inventorydepreciationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_depreciation"
)

// ReadInventoryDepreciationRepositories groups all repository dependencies
type ReadInventoryDepreciationRepositories struct {
	InventoryDepreciation inventorydepreciationpb.InventoryDepreciationDomainServiceServer
}

// ReadInventoryDepreciationServices groups all business service dependencies
type ReadInventoryDepreciationServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadInventoryDepreciationUseCase handles the business logic for reading an inventory depreciation
type ReadInventoryDepreciationUseCase struct {
	repositories ReadInventoryDepreciationRepositories
	services     ReadInventoryDepreciationServices
}

// NewReadInventoryDepreciationUseCase creates use case with grouped dependencies
func NewReadInventoryDepreciationUseCase(
	repositories ReadInventoryDepreciationRepositories,
	services ReadInventoryDepreciationServices,
) *ReadInventoryDepreciationUseCase {
	return &ReadInventoryDepreciationUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read inventory depreciation operation
func (uc *ReadInventoryDepreciationUseCase) Execute(ctx context.Context, req *inventorydepreciationpb.ReadInventoryDepreciationRequest) (*inventorydepreciationpb.ReadInventoryDepreciationResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.InventoryDepreciation,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_depreciation.errors.authorization_failed", "Authorization failed for inventory depreciations [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.InventoryDepreciation, entityid.ActionRead)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_depreciation.errors.authorization_failed", "Authorization failed for inventory depreciations [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_depreciation.errors.authorization_failed", "Authorization failed for inventory depreciations [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_depreciation.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	resp, err := uc.repositories.InventoryDepreciation.ReadInventoryDepreciation(ctx, req)
	if err != nil {
		errorMsg := strings.ToLower(err.Error())
		if strings.Contains(errorMsg, "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_depreciation.errors.not_found", "Inventory depreciation not found [DEFAULT]")
			return nil, errors.New(translatedError)
		}
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_depreciation.errors.read_failed", "Failed to retrieve inventory depreciation [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *ReadInventoryDepreciationUseCase) validateInput(ctx context.Context, req *inventorydepreciationpb.ReadInventoryDepreciationRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_depreciation.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_depreciation.validation.data_required", "Inventory depreciation data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "inventory_depreciation.validation.id_required", "Inventory depreciation ID is required [DEFAULT]"))
	}
	return nil
}
