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

// DeleteInventoryDepreciationRepositories groups all repository dependencies
type DeleteInventoryDepreciationRepositories struct {
	InventoryDepreciation inventorydepreciationpb.InventoryDepreciationDomainServiceServer
}

// DeleteInventoryDepreciationServices groups all business service dependencies
type DeleteInventoryDepreciationServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteInventoryDepreciationUseCase handles the business logic for deleting inventory depreciations
type DeleteInventoryDepreciationUseCase struct {
	repositories DeleteInventoryDepreciationRepositories
	services     DeleteInventoryDepreciationServices
}

// NewDeleteInventoryDepreciationUseCase creates a new DeleteInventoryDepreciationUseCase
func NewDeleteInventoryDepreciationUseCase(
	repositories DeleteInventoryDepreciationRepositories,
	services DeleteInventoryDepreciationServices,
) *DeleteInventoryDepreciationUseCase {
	return &DeleteInventoryDepreciationUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete inventory depreciation operation
func (uc *DeleteInventoryDepreciationUseCase) Execute(ctx context.Context, req *inventorydepreciationpb.DeleteInventoryDepreciationRequest) (*inventorydepreciationpb.DeleteInventoryDepreciationResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventoryDepreciation, ports.ActionDelete); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *DeleteInventoryDepreciationUseCase) executeWithTransaction(ctx context.Context, req *inventorydepreciationpb.DeleteInventoryDepreciationRequest) (*inventorydepreciationpb.DeleteInventoryDepreciationResponse, error) {
	var result *inventorydepreciationpb.DeleteInventoryDepreciationResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return result, nil
}

func (uc *DeleteInventoryDepreciationUseCase) executeCore(ctx context.Context, req *inventorydepreciationpb.DeleteInventoryDepreciationRequest) (*inventorydepreciationpb.DeleteInventoryDepreciationResponse, error) {
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.errors.authorization_failed", "Authorization failed for inventory depreciations [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventoryDepreciation, ports.ActionDelete)
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

	resp, err := uc.repositories.InventoryDepreciation.DeleteInventoryDepreciation(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.errors.deletion_failed", "Inventory depreciation deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *DeleteInventoryDepreciationUseCase) validateInput(ctx context.Context, req *inventorydepreciationpb.DeleteInventoryDepreciationRequest) error {
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
