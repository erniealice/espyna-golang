package inventory_depreciation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	inventorydepreciationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_depreciation"
)

// UpdateInventoryDepreciationRepositories groups all repository dependencies
type UpdateInventoryDepreciationRepositories struct {
	InventoryDepreciation inventorydepreciationpb.InventoryDepreciationDomainServiceServer
}

// UpdateInventoryDepreciationServices groups all business service dependencies
type UpdateInventoryDepreciationServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateInventoryDepreciationUseCase handles the business logic for updating inventory depreciations
type UpdateInventoryDepreciationUseCase struct {
	repositories UpdateInventoryDepreciationRepositories
	services     UpdateInventoryDepreciationServices
}

// NewUpdateInventoryDepreciationUseCase creates use case with grouped dependencies
func NewUpdateInventoryDepreciationUseCase(
	repositories UpdateInventoryDepreciationRepositories,
	services UpdateInventoryDepreciationServices,
) *UpdateInventoryDepreciationUseCase {
	return &UpdateInventoryDepreciationUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update inventory depreciation operation
func (uc *UpdateInventoryDepreciationUseCase) Execute(ctx context.Context, req *inventorydepreciationpb.UpdateInventoryDepreciationRequest) (*inventorydepreciationpb.UpdateInventoryDepreciationResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventoryDepreciation, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *UpdateInventoryDepreciationUseCase) executeWithTransaction(ctx context.Context, req *inventorydepreciationpb.UpdateInventoryDepreciationRequest) (*inventorydepreciationpb.UpdateInventoryDepreciationResponse, error) {
	var result *inventorydepreciationpb.UpdateInventoryDepreciationResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "inventory_depreciation.errors.update_failed", "Inventory depreciation update failed [DEFAULT]")
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

func (uc *UpdateInventoryDepreciationUseCase) executeCore(ctx context.Context, req *inventorydepreciationpb.UpdateInventoryDepreciationRequest) (*inventorydepreciationpb.UpdateInventoryDepreciationResponse, error) {
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.errors.authorization_failed", "Authorization failed for inventory depreciations [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventoryDepreciation, ports.ActionUpdate)
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

	if err := uc.enrichData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	existingResp, err := uc.repositories.InventoryDepreciation.ReadInventoryDepreciation(ctx, &inventorydepreciationpb.ReadInventoryDepreciationRequest{Data: &inventorydepreciationpb.InventoryDepreciation{Id: req.Data.Id}})
	if err != nil || existingResp == nil || len(existingResp.Data) == 0 {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.errors.not_found", "Inventory depreciation not found for update [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	existing := existingResp.Data[0]

	if req.Data.Active == false {
		req.Data.Active = existing.Active
	}

	resp, err := uc.repositories.InventoryDepreciation.UpdateInventoryDepreciation(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.errors.update_failed", "Inventory depreciation update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *UpdateInventoryDepreciationUseCase) validateInput(ctx context.Context, req *inventorydepreciationpb.UpdateInventoryDepreciationRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.validation.data_required", "Inventory depreciation data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.validation.id_required", "Inventory depreciation ID is required [DEFAULT]"))
	}
	return nil
}

func (uc *UpdateInventoryDepreciationUseCase) enrichData(depreciation *inventorydepreciationpb.InventoryDepreciation) error {
	now := time.Now()
	depreciation.DateModified = &[]int64{now.UnixMilli()}[0]
	depreciation.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return nil
}
