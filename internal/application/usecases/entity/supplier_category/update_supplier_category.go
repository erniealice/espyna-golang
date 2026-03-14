package supplier_category

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	suppliercategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_category"
)

// UpdateSupplierCategoryRepositories groups all repository dependencies
type UpdateSupplierCategoryRepositories struct {
	SupplierCategory suppliercategorypb.SupplierCategoryDomainServiceServer
}

// UpdateSupplierCategoryServices groups all business service dependencies
type UpdateSupplierCategoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UpdateSupplierCategoryUseCase handles the business logic for updating supplier categories
type UpdateSupplierCategoryUseCase struct {
	repositories UpdateSupplierCategoryRepositories
	services     UpdateSupplierCategoryServices
}

// NewUpdateSupplierCategoryUseCase creates use case with grouped dependencies
func NewUpdateSupplierCategoryUseCase(
	repositories UpdateSupplierCategoryRepositories,
	services UpdateSupplierCategoryServices,
) *UpdateSupplierCategoryUseCase {
	return &UpdateSupplierCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateSupplierCategoryUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateSupplierCategoryUseCase with grouped parameters instead
func NewUpdateSupplierCategoryUseCaseUngrouped(supplierCategoryRepo suppliercategorypb.SupplierCategoryDomainServiceServer) *UpdateSupplierCategoryUseCase {
	repositories := UpdateSupplierCategoryRepositories{
		SupplierCategory: supplierCategoryRepo,
	}

	services := UpdateSupplierCategoryServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewUpdateSupplierCategoryUseCase(repositories, services)
}

func (uc *UpdateSupplierCategoryUseCase) Execute(ctx context.Context, req *suppliercategorypb.UpdateSupplierCategoryRequest) (*suppliercategorypb.UpdateSupplierCategoryResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"supplier_category", ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *UpdateSupplierCategoryUseCase) executeWithTransaction(ctx context.Context, req *suppliercategorypb.UpdateSupplierCategoryRequest) (*suppliercategorypb.UpdateSupplierCategoryResponse, error) {
	var result *suppliercategorypb.UpdateSupplierCategoryResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "supplier_category.errors.update_failed", "Supplier category update failed [DEFAULT]")
			return errors.New(translatedError + ": " + err.Error())
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (uc *UpdateSupplierCategoryUseCase) executeCore(ctx context.Context, req *suppliercategorypb.UpdateSupplierCategoryRequest) (*suppliercategorypb.UpdateSupplierCategoryResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	return uc.repositories.SupplierCategory.UpdateSupplierCategory(ctx, req)
}

func (uc *UpdateSupplierCategoryUseCase) validateInput(ctx context.Context, req *suppliercategorypb.UpdateSupplierCategoryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_category.validation.request_required", "Request is required for supplier categories [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_category.validation.data_required", "Supplier category data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_category.validation.id_required", "Supplier category ID is required for update [DEFAULT]"))
	}
	return nil
}
