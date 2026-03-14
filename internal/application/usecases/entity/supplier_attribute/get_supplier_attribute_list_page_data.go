package supplier_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	supplierattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_attribute"
)

// GetSupplierAttributeListPageDataRepositories groups all repository dependencies
type GetSupplierAttributeListPageDataRepositories struct {
	SupplierAttribute supplierattributepb.SupplierAttributeDomainServiceServer // Primary entity repository
}

// GetSupplierAttributeListPageDataServices groups all business service dependencies
type GetSupplierAttributeListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetSupplierAttributeListPageDataUseCase handles the business logic for getting supplier attribute list page data
type GetSupplierAttributeListPageDataUseCase struct {
	repositories GetSupplierAttributeListPageDataRepositories
	services     GetSupplierAttributeListPageDataServices
}

// NewGetSupplierAttributeListPageDataUseCase creates use case with grouped dependencies
func NewGetSupplierAttributeListPageDataUseCase(
	repositories GetSupplierAttributeListPageDataRepositories,
	services GetSupplierAttributeListPageDataServices,
) *GetSupplierAttributeListPageDataUseCase {
	return &GetSupplierAttributeListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetSupplierAttributeListPageDataUseCaseUngrouped creates a new GetSupplierAttributeListPageDataUseCase
// Deprecated: Use NewGetSupplierAttributeListPageDataUseCase with grouped parameters instead
func NewGetSupplierAttributeListPageDataUseCaseUngrouped(supplierAttributeRepo supplierattributepb.SupplierAttributeDomainServiceServer) *GetSupplierAttributeListPageDataUseCase {
	repositories := GetSupplierAttributeListPageDataRepositories{
		SupplierAttribute: supplierAttributeRepo,
	}

	services := GetSupplierAttributeListPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewGetSupplierAttributeListPageDataUseCase(repositories, services)
}

// Execute performs the get supplier attribute list page data operation
func (uc *GetSupplierAttributeListPageDataUseCase) Execute(ctx context.Context, req *supplierattributepb.GetSupplierAttributeListPageDataRequest) (*supplierattributepb.GetSupplierAttributeListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"supplier_attribute", ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Apply default values
	if err := uc.applyDefaults(req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.errors.apply_defaults_failed", "Failed to apply default values [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.SupplierAttribute.GetSupplierAttributeListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.errors.list_page_data_failed", "Failed to retrieve supplier attribute list page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetSupplierAttributeListPageDataUseCase) validateInput(ctx context.Context, req *supplierattributepb.GetSupplierAttributeListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.request_required", "Request is required for supplier attributes [DEFAULT]"))
	}

	if req.Pagination != nil {
		if req.Pagination.Limit < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.pagination_limit_invalid", "Pagination limit must be non-negative [DEFAULT]"))
		}
		if req.Pagination.Limit > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.pagination_limit_exceeded", "Pagination limit cannot exceed 1000 [DEFAULT]"))
		}
	}

	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) < 2 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.search_query_too_short", "Search query must be at least 2 characters [DEFAULT]"))
		}
	}

	return nil
}

// applyDefaults sets default values for optional parameters
func (uc *GetSupplierAttributeListPageDataUseCase) applyDefaults(req *supplierattributepb.GetSupplierAttributeListPageDataRequest) error {
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
