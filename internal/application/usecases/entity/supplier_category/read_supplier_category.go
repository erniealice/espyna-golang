package supplier_category

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	suppliercategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_category"
)

// ReadSupplierCategoryRepositories groups all repository dependencies
type ReadSupplierCategoryRepositories struct {
	SupplierCategory suppliercategorypb.SupplierCategoryDomainServiceServer
}

// ReadSupplierCategoryServices groups all business service dependencies
type ReadSupplierCategoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadSupplierCategoryUseCase handles the business logic for reading supplier categories
type ReadSupplierCategoryUseCase struct {
	repositories ReadSupplierCategoryRepositories
	services     ReadSupplierCategoryServices
}

// NewReadSupplierCategoryUseCase creates use case with grouped dependencies
func NewReadSupplierCategoryUseCase(
	repositories ReadSupplierCategoryRepositories,
	services ReadSupplierCategoryServices,
) *ReadSupplierCategoryUseCase {
	return &ReadSupplierCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadSupplierCategoryUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadSupplierCategoryUseCase with grouped parameters instead
func NewReadSupplierCategoryUseCaseUngrouped(supplierCategoryRepo suppliercategorypb.SupplierCategoryDomainServiceServer) *ReadSupplierCategoryUseCase {
	repositories := ReadSupplierCategoryRepositories{
		SupplierCategory: supplierCategoryRepo,
	}

	services := ReadSupplierCategoryServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadSupplierCategoryUseCase(repositories, services)
}

func (uc *ReadSupplierCategoryUseCase) Execute(ctx context.Context, req *suppliercategorypb.ReadSupplierCategoryRequest) (*suppliercategorypb.ReadSupplierCategoryResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"supplier_category", ports.ActionRead); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.SupplierCategory.ReadSupplierCategory(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (uc *ReadSupplierCategoryUseCase) validateInput(ctx context.Context, req *suppliercategorypb.ReadSupplierCategoryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_category.validation.request_required", "Request is required for supplier categories [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_category.validation.data_required", "Supplier category data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_category.validation.id_required", "Supplier category ID is required [DEFAULT]"))
	}
	return nil
}
