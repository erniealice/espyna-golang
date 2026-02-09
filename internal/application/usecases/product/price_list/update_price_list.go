package price_list

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pricelistpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_list"
)

// UpdatePriceListRepositories groups all repository dependencies
type UpdatePriceListRepositories struct {
	PriceList pricelistpb.PriceListDomainServiceServer // Primary entity repository
}

// UpdatePriceListServices groups all business service dependencies
type UpdatePriceListServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// UpdatePriceListUseCase handles the business logic for updating price lists
type UpdatePriceListUseCase struct {
	repositories UpdatePriceListRepositories
	services     UpdatePriceListServices
}

// NewUpdatePriceListUseCase creates a new UpdatePriceListUseCase
func NewUpdatePriceListUseCase(
	repositories UpdatePriceListRepositories,
	services UpdatePriceListServices,
) *UpdatePriceListUseCase {
	return &UpdatePriceListUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update price list operation
func (uc *UpdatePriceListUseCase) Execute(ctx context.Context, req *pricelistpb.UpdatePriceListRequest) (*pricelistpb.UpdatePriceListResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPriceList, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes price list update within a transaction
func (uc *UpdatePriceListUseCase) executeWithTransaction(ctx context.Context, req *pricelistpb.UpdatePriceListRequest) (*pricelistpb.UpdatePriceListResponse, error) {
	var result *pricelistpb.UpdatePriceListResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "price_list.errors.update_failed", "Price List update failed [DEFAULT]")
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

// executeCore contains the core business logic
func (uc *UpdatePriceListUseCase) executeCore(ctx context.Context, req *pricelistpb.UpdatePriceListRequest) (*pricelistpb.UpdatePriceListResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.errors.authorization_failed", "Authorization failed for price lists [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityPriceList, ports.ActionUpdate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.errors.authorization_failed", "Authorization failed for price lists [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.errors.authorization_failed", "Authorization failed for price lists [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.PriceList.UpdatePriceList(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.errors.update_failed", "Price List update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched price list
func (uc *UpdatePriceListUseCase) applyBusinessLogic(priceList *pricelistpb.PriceList) *pricelistpb.PriceList {
	now := time.Now()

	// Business logic: Update modification audit fields
	priceList.DateModified = &[]int64{now.UnixMilli()}[0]
	priceList.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return priceList
}

// validateInput validates the input request
func (uc *UpdatePriceListUseCase) validateInput(ctx context.Context, req *pricelistpb.UpdatePriceListRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.validation.data_required", "Price List data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.validation.id_required", "Price List ID is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.validation.name_required", "Price List name is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for price lists
func (uc *UpdatePriceListUseCase) validateBusinessRules(ctx context.Context, priceList *pricelistpb.PriceList) error {
	// Validate price list name length
	if len(priceList.Name) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.validation.name_min_length", "Price list name must be at least 3 characters long [DEFAULT]"))
	}

	if len(priceList.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.validation.name_max_length", "Price list name cannot exceed 100 characters [DEFAULT]"))
	}

	return nil
}
