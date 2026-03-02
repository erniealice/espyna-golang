package supplier_category

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	suppliercategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_category"
)

// ListSupplierCategoriesRepositories groups all repository dependencies
type ListSupplierCategoriesRepositories struct {
	SupplierCategory suppliercategorypb.SupplierCategoryDomainServiceServer
}

// ListSupplierCategoriesServices groups all business service dependencies
type ListSupplierCategoriesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListSupplierCategoriesUseCase handles the business logic for listing supplier categories
type ListSupplierCategoriesUseCase struct {
	repositories ListSupplierCategoriesRepositories
	services     ListSupplierCategoriesServices
}

// NewListSupplierCategoriesUseCase creates use case with grouped dependencies
func NewListSupplierCategoriesUseCase(
	repositories ListSupplierCategoriesRepositories,
	services ListSupplierCategoriesServices,
) *ListSupplierCategoriesUseCase {
	return &ListSupplierCategoriesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListSupplierCategoriesUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListSupplierCategoriesUseCase with grouped parameters instead
func NewListSupplierCategoriesUseCaseUngrouped(supplierCategoryRepo suppliercategorypb.SupplierCategoryDomainServiceServer) *ListSupplierCategoriesUseCase {
	repositories := ListSupplierCategoriesRepositories{
		SupplierCategory: supplierCategoryRepo,
	}

	services := ListSupplierCategoriesServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListSupplierCategoriesUseCase(repositories, services)
}

func (uc *ListSupplierCategoriesUseCase) Execute(ctx context.Context, req *suppliercategorypb.ListSupplierCategoriesRequest) (*suppliercategorypb.ListSupplierCategoriesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"supplier_category", ports.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.SupplierCategory.ListSupplierCategories(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (uc *ListSupplierCategoriesUseCase) validateInput(ctx context.Context, req *suppliercategorypb.ListSupplierCategoriesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_category.validation.request_required", "Request is required for supplier categories [DEFAULT]"))
	}
	return nil
}
