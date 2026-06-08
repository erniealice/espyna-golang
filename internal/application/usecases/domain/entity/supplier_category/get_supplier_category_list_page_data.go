package supplier_category

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	suppliercategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_category"
)

// GetSupplierCategoryListPageDataRepositories groups all repository dependencies
type GetSupplierCategoryListPageDataRepositories struct {
	SupplierCategory suppliercategorypb.SupplierCategoryDomainServiceServer
}

// GetSupplierCategoryListPageDataServices groups all business service dependencies
type GetSupplierCategoryListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetSupplierCategoryListPageDataUseCase handles the business logic for getting supplier category list page data
type GetSupplierCategoryListPageDataUseCase struct {
	repositories GetSupplierCategoryListPageDataRepositories
	services     GetSupplierCategoryListPageDataServices
}

// NewGetSupplierCategoryListPageDataUseCase creates use case with grouped dependencies
func NewGetSupplierCategoryListPageDataUseCase(
	repositories GetSupplierCategoryListPageDataRepositories,
	services GetSupplierCategoryListPageDataServices,
) *GetSupplierCategoryListPageDataUseCase {
	return &GetSupplierCategoryListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetSupplierCategoryListPageDataUseCaseUngrouped creates a new GetSupplierCategoryListPageDataUseCase
// Deprecated: Use NewGetSupplierCategoryListPageDataUseCase with grouped parameters instead
func NewGetSupplierCategoryListPageDataUseCaseUngrouped(supplierCategoryRepo suppliercategorypb.SupplierCategoryDomainServiceServer) *GetSupplierCategoryListPageDataUseCase {
	repositories := GetSupplierCategoryListPageDataRepositories{
		SupplierCategory: supplierCategoryRepo,
	}

	services := GetSupplierCategoryListPageDataServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewGetSupplierCategoryListPageDataUseCase(repositories, services)
}

func (uc *GetSupplierCategoryListPageDataUseCase) Execute(ctx context.Context, req *suppliercategorypb.GetSupplierCategoryListPageDataRequest) (*suppliercategorypb.GetSupplierCategoryListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		"supplier_category", entityid.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_category.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if err := uc.applyDefaults(req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_category.errors.apply_defaults_failed", "Failed to apply default values [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	resp, err := uc.repositories.SupplierCategory.GetSupplierCategoryListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_category.errors.list_page_data_failed", "Failed to retrieve supplier category list page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

func (uc *GetSupplierCategoryListPageDataUseCase) validateInput(ctx context.Context, req *suppliercategorypb.GetSupplierCategoryListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_category.validation.request_required", "Request is required for supplier categories [DEFAULT]"))
	}

	if req.Pagination != nil {
		if req.Pagination.Limit < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_category.validation.pagination_limit_invalid", "Pagination limit must be non-negative [DEFAULT]"))
		}
		if req.Pagination.Limit > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_category.validation.pagination_limit_exceeded", "Pagination limit cannot exceed 1000 [DEFAULT]"))
		}
	}

	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) < 2 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_category.validation.search_query_too_short", "Search query must be at least 2 characters [DEFAULT]"))
		}
	}

	return nil
}

func (uc *GetSupplierCategoryListPageDataUseCase) applyDefaults(req *suppliercategorypb.GetSupplierCategoryListPageDataRequest) error {
	if req.Pagination == nil {
		req.Pagination = &commonpb.PaginationRequest{
			Limit: 10,
			Method: &commonpb.PaginationRequest_Offset{
				Offset: &commonpb.OffsetPagination{
					Page: 1,
				},
			},
		}
	} else if req.Pagination.Limit == 0 {
		req.Pagination.Limit = 10
	}

	return nil
}
