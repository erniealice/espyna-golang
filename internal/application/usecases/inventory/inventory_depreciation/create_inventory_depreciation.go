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

// CreateInventoryDepreciationRepositories groups all repository dependencies
type CreateInventoryDepreciationRepositories struct {
	InventoryDepreciation inventorydepreciationpb.InventoryDepreciationDomainServiceServer
}

// CreateInventoryDepreciationServices groups all business service dependencies
type CreateInventoryDepreciationServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateInventoryDepreciationUseCase handles the business logic for creating inventory depreciations
type CreateInventoryDepreciationUseCase struct {
	repositories CreateInventoryDepreciationRepositories
	services     CreateInventoryDepreciationServices
}

// NewCreateInventoryDepreciationUseCase creates use case with grouped dependencies
func NewCreateInventoryDepreciationUseCase(
	repositories CreateInventoryDepreciationRepositories,
	services CreateInventoryDepreciationServices,
) *CreateInventoryDepreciationUseCase {
	return &CreateInventoryDepreciationUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create inventory depreciation operation
func (uc *CreateInventoryDepreciationUseCase) Execute(ctx context.Context, req *inventorydepreciationpb.CreateInventoryDepreciationRequest) (*inventorydepreciationpb.CreateInventoryDepreciationResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInventoryDepreciation, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *CreateInventoryDepreciationUseCase) executeWithTransaction(ctx context.Context, req *inventorydepreciationpb.CreateInventoryDepreciationRequest) (*inventorydepreciationpb.CreateInventoryDepreciationResponse, error) {
	var result *inventorydepreciationpb.CreateInventoryDepreciationResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("inventory depreciation creation failed: %w", err)
		}
		result = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (uc *CreateInventoryDepreciationUseCase) executeCore(ctx context.Context, req *inventorydepreciationpb.CreateInventoryDepreciationRequest) (*inventorydepreciationpb.CreateInventoryDepreciationResponse, error) {
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.errors.authorization_failed", "Authorization failed for inventory depreciations [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInventoryDepreciation, ports.ActionCreate)
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

	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return uc.repositories.InventoryDepreciation.CreateInventoryDepreciation(ctx, req)
}

func (uc *CreateInventoryDepreciationUseCase) validateInput(ctx context.Context, req *inventorydepreciationpb.CreateInventoryDepreciationRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.validation.data_required", "Inventory depreciation data is required [DEFAULT]"))
	}
	if req.Data.InventoryItemId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.validation.inventory_item_id_required", "Inventory item ID is required [DEFAULT]"))
	}
	if req.Data.Method == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.validation.method_required", "Depreciation method is required [DEFAULT]"))
	}
	if req.Data.CostBasis <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.validation.cost_basis_positive", "Cost basis must be greater than zero [DEFAULT]"))
	}
	if req.Data.UsefulLifeMonths <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.validation.useful_life_months_positive", "Useful life months must be greater than zero [DEFAULT]"))
	}
	if req.Data.StartDate == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.validation.start_date_required", "Start date is required [DEFAULT]"))
	}
	return nil
}

func (uc *CreateInventoryDepreciationUseCase) enrichData(depreciation *inventorydepreciationpb.InventoryDepreciation) error {
	now := time.Now()

	if depreciation.Id == "" {
		depreciation.Id = uc.services.IDService.GenerateID()
	}

	depreciation.DateCreated = &[]int64{now.UnixMilli()}[0]
	depreciation.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	depreciation.DateModified = &[]int64{now.UnixMilli()}[0]
	depreciation.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	depreciation.Active = true

	// Initialize book value to cost basis minus salvage value if not set
	if depreciation.BookValue == 0 {
		depreciation.BookValue = depreciation.CostBasis - depreciation.SalvageValue
	}

	return nil
}

func (uc *CreateInventoryDepreciationUseCase) validateBusinessRules(ctx context.Context, depreciation *inventorydepreciationpb.InventoryDepreciation) error {
	// Validate depreciation method
	validMethods := map[string]bool{"straight_line": true, "declining_balance": true, "sum_of_years": true, "units_of_production": true}
	if !validMethods[depreciation.Method] {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.validation.invalid_method", "Depreciation method must be straight_line, declining_balance, sum_of_years, or units_of_production [DEFAULT]"))
	}

	// Validate salvage value is not greater than cost basis
	if depreciation.SalvageValue > depreciation.CostBasis {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_depreciation.validation.salvage_exceeds_cost", "Salvage value cannot exceed cost basis [DEFAULT]"))
	}

	return nil
}
