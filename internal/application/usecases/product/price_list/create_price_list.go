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

// CreatePriceListRepositories groups all repository dependencies
type CreatePriceListRepositories struct {
	PriceList pricelistpb.PriceListDomainServiceServer // Primary entity repository
}

// CreatePriceListServices groups all business service dependencies
type CreatePriceListServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Transaction management
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreatePriceListUseCase handles the business logic for creating price lists
type CreatePriceListUseCase struct {
	repositories CreatePriceListRepositories
	services     CreatePriceListServices
}

// NewCreatePriceListUseCase creates use case with grouped dependencies
func NewCreatePriceListUseCase(
	repositories CreatePriceListRepositories,
	services CreatePriceListServices,
) *CreatePriceListUseCase {
	return &CreatePriceListUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create price list operation
func (uc *CreatePriceListUseCase) Execute(ctx context.Context, req *pricelistpb.CreatePriceListRequest) (*pricelistpb.CreatePriceListResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPriceList, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.errors.authorization_failed", "Authorization failed for price list [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityPriceList, ports.ActionCreate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.errors.authorization_failed", "Authorization failed for price list [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.errors.authorization_failed", "Authorization failed for price list [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichPriceListData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Determine if we should use transactions
	if uc.shouldUseTransaction(ctx) {
		return uc.executeWithTransaction(ctx, req)
	}

	// Execute without transaction (backward compatibility)
	return uc.executeWithoutTransaction(ctx, req)
}

// shouldUseTransaction determines if this operation should use a transaction
func (uc *CreatePriceListUseCase) shouldUseTransaction(ctx context.Context) bool {
	if uc.services.TransactionService == nil || !uc.services.TransactionService.SupportsTransactions() {
		return false
	}

	if uc.services.TransactionService.IsTransactionActive(ctx) {
		return false
	}

	return true
}

// executeWithTransaction performs the operation within a transaction
func (uc *CreatePriceListUseCase) executeWithTransaction(ctx context.Context, req *pricelistpb.CreatePriceListRequest) (*pricelistpb.CreatePriceListResponse, error) {
	var response *pricelistpb.CreatePriceListResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// Business rule validation
		if err := uc.validateBusinessRules(txCtx, req.Data); err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "price_list.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}

		// Create PriceList
		createResponse, err := uc.repositories.PriceList.CreatePriceList(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "price_list.errors.creation_failed", "Failed to create price list [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}

		response = createResponse
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("transaction execution failed: %w", err)
	}

	return response, nil
}

// executeWithoutTransaction performs the operation without transaction
func (uc *CreatePriceListUseCase) executeWithoutTransaction(ctx context.Context, req *pricelistpb.CreatePriceListRequest) (*pricelistpb.CreatePriceListResponse, error) {
	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.PriceList.CreatePriceList(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.errors.creation_failed", "Failed to create price list [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *CreatePriceListUseCase) validateInput(ctx context.Context, req *pricelistpb.CreatePriceListRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.validation.request_required", "Request is required for price list [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.validation.data_required", "Price list data is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.validation.name_required", "Price list name is required [DEFAULT]"))
	}
	return nil
}

// enrichPriceListData adds generated fields and audit information
func (uc *CreatePriceListUseCase) enrichPriceListData(priceList *pricelistpb.PriceList) error {
	now := time.Now()

	// Generate PriceList ID if not provided
	if priceList.Id == "" {
		priceList.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	priceList.DateCreated = &[]int64{now.UnixMilli()}[0]
	priceList.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	priceList.DateModified = &[]int64{now.UnixMilli()}[0]
	priceList.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	priceList.Active = true

	return nil
}

// validateBusinessRules enforces business constraints for price lists
func (uc *CreatePriceListUseCase) validateBusinessRules(ctx context.Context, priceList *pricelistpb.PriceList) error {
	// Validate price list name length
	if len(priceList.Name) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.validation.name_min_length", "Price list name must be at least 3 characters long [DEFAULT]"))
	}

	if len(priceList.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.validation.name_max_length", "Price list name cannot exceed 100 characters [DEFAULT]"))
	}

	return nil
}
