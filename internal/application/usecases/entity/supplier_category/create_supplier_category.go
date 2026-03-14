package supplier_category

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	suppliercategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_category"
)

// CreateSupplierCategoryRepositories groups all repository dependencies
type CreateSupplierCategoryRepositories struct {
	SupplierCategory suppliercategorypb.SupplierCategoryDomainServiceServer // Primary entity repository
}

// CreateSupplierCategoryServices groups all business service dependencies
type CreateSupplierCategoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateSupplierCategoryUseCase handles the business logic for creating supplier categories
type CreateSupplierCategoryUseCase struct {
	repositories CreateSupplierCategoryRepositories
	services     CreateSupplierCategoryServices
}

// NewCreateSupplierCategoryUseCase creates use case with grouped dependencies
func NewCreateSupplierCategoryUseCase(
	repositories CreateSupplierCategoryRepositories,
	services CreateSupplierCategoryServices,
) *CreateSupplierCategoryUseCase {
	return &CreateSupplierCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateSupplierCategoryUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateSupplierCategoryUseCase with grouped parameters instead
func NewCreateSupplierCategoryUseCaseUngrouped(supplierCategoryRepo suppliercategorypb.SupplierCategoryDomainServiceServer) *CreateSupplierCategoryUseCase {
	repositories := CreateSupplierCategoryRepositories{
		SupplierCategory: supplierCategoryRepo,
	}

	services := CreateSupplierCategoryServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateSupplierCategoryUseCase(repositories, services)
}

// Execute performs the create supplier_category operation
func (uc *CreateSupplierCategoryUseCase) Execute(ctx context.Context, req *suppliercategorypb.CreateSupplierCategoryRequest) (*suppliercategorypb.CreateSupplierCategoryResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"supplier_category", ports.ActionCreate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes supplier_category creation within a transaction
func (uc *CreateSupplierCategoryUseCase) executeWithTransaction(ctx context.Context, req *suppliercategorypb.CreateSupplierCategoryRequest) (*suppliercategorypb.CreateSupplierCategoryResponse, error) {
	var result *suppliercategorypb.CreateSupplierCategoryResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "supplier_category.errors.creation_failed", "Supplier category creation failed [DEFAULT]")
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
func (uc *CreateSupplierCategoryUseCase) executeCore(ctx context.Context, req *suppliercategorypb.CreateSupplierCategoryRequest) (*suppliercategorypb.CreateSupplierCategoryResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichSupplierCategoryData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.SupplierCategory.CreateSupplierCategory(ctx, req)
}

// validateInput validates the input request
func (uc *CreateSupplierCategoryUseCase) validateInput(ctx context.Context, req *suppliercategorypb.CreateSupplierCategoryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_category.validation.request_required", "Request is required for supplier categories [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_category.validation.data_required", "Supplier category data is required [DEFAULT]"))
	}
	if req.Data.SupplierId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_category.validation.supplier_id_required", "Supplier ID is required for supplier categories [DEFAULT]"))
	}
	if req.Data.CategoryId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_category.validation.category_id_required", "Category ID is required for supplier categories [DEFAULT]"))
	}
	return nil
}

// enrichSupplierCategoryData adds generated fields and audit information
func (uc *CreateSupplierCategoryUseCase) enrichSupplierCategoryData(category *suppliercategorypb.SupplierCategory) error {
	now := time.Now()

	// Generate Supplier Category ID if not provided
	if category.Id == "" {
		category.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	category.DateCreated = &[]int64{now.UnixMilli()}[0]
	category.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	category.DateModified = &[]int64{now.UnixMilli()}[0]
	category.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	category.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateSupplierCategoryUseCase) validateBusinessRules(ctx context.Context, category *suppliercategorypb.SupplierCategory) error {
	// Validate supplier ID format if needed
	if len(category.SupplierId) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_category.validation.supplier_id_invalid", "Supplier ID is invalid [DEFAULT]"))
	}

	// Validate category ID format if needed
	if len(category.CategoryId) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_category.validation.category_id_invalid", "Category ID is invalid [DEFAULT]"))
	}

	return nil
}
