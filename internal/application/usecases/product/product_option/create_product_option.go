package product_option

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	productoptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_option"
)

// CreateProductOptionRepositories groups all repository dependencies
type CreateProductOptionRepositories struct {
	ProductOption productoptionpb.ProductOptionDomainServiceServer // Primary entity repository
}

// CreateProductOptionServices groups all business service dependencies
type CreateProductOptionServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateProductOptionUseCase handles the business logic for creating product options
type CreateProductOptionUseCase struct {
	repositories CreateProductOptionRepositories
	services     CreateProductOptionServices
}

// NewCreateProductOptionUseCase creates use case with grouped dependencies
func NewCreateProductOptionUseCase(
	repositories CreateProductOptionRepositories,
	services CreateProductOptionServices,
) *CreateProductOptionUseCase {
	return &CreateProductOptionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create product option operation
func (uc *CreateProductOptionUseCase) Execute(ctx context.Context, req *productoptionpb.CreateProductOptionRequest) (*productoptionpb.CreateProductOptionResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductOption, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product option creation within a transaction
func (uc *CreateProductOptionUseCase) executeWithTransaction(ctx context.Context, req *productoptionpb.CreateProductOptionRequest) (*productoptionpb.CreateProductOptionResponse, error) {
	var result *productoptionpb.CreateProductOptionResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("product option creation failed: %w", err)
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
func (uc *CreateProductOptionUseCase) executeCore(ctx context.Context, req *productoptionpb.CreateProductOptionRequest) (*productoptionpb.CreateProductOptionResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.errors.authorization_failed", "Authorization failed for product options [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductOption, ports.ActionCreate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.errors.authorization_failed", "Authorization failed for product options [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.errors.authorization_failed", "Authorization failed for product options [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	return uc.repositories.ProductOption.CreateProductOption(ctx, req)
}

// validateInput validates the input request
func (uc *CreateProductOptionUseCase) validateInput(ctx context.Context, req *productoptionpb.CreateProductOptionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.validation.data_required", "Product option data is required [DEFAULT]"))
	}
	if req.Data.ProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.validation.product_id_required", "Product ID is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.validation.name_required", "Product option name is required [DEFAULT]"))
	}
	if req.Data.Code == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option.validation.code_required", "Product option code is required [DEFAULT]"))
	}
	return nil
}

// enrichData adds generated fields and audit information
func (uc *CreateProductOptionUseCase) enrichData(data *productoptionpb.ProductOption) error {
	now := time.Now()

	// Generate ID if not provided
	if data.Id == "" {
		data.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	data.DateCreated = &[]int64{now.UnixMilli()}[0]
	data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	data.Active = true

	return nil
}
