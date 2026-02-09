package staff_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	staffattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff_attribute"
)

// GetStaffAttributeListPageDataUseCase handles the business logic for getting staff attribute list page data
// GetStaffAttributeListPageDataRepositories groups all repository dependencies
type GetStaffAttributeListPageDataRepositories struct {
	StaffAttribute staffattributepb.StaffAttributeDomainServiceServer // Primary entity repository
}

// GetStaffAttributeListPageDataServices groups all business service dependencies
type GetStaffAttributeListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetStaffAttributeListPageDataUseCase handles the business logic for getting staff attribute list page data
type GetStaffAttributeListPageDataUseCase struct {
	repositories GetStaffAttributeListPageDataRepositories
	services     GetStaffAttributeListPageDataServices
}

// NewGetStaffAttributeListPageDataUseCase creates use case with grouped dependencies
func NewGetStaffAttributeListPageDataUseCase(
	repositories GetStaffAttributeListPageDataRepositories,
	services GetStaffAttributeListPageDataServices,
) *GetStaffAttributeListPageDataUseCase {
	return &GetStaffAttributeListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetStaffAttributeListPageDataUseCaseUngrouped creates a new GetStaffAttributeListPageDataUseCase
// Deprecated: Use NewGetStaffAttributeListPageDataUseCase with grouped parameters instead
func NewGetStaffAttributeListPageDataUseCaseUngrouped(staffAttributeRepo staffattributepb.StaffAttributeDomainServiceServer) *GetStaffAttributeListPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetStaffAttributeListPageDataRepositories{
		StaffAttribute: staffAttributeRepo,
	}

	services := GetStaffAttributeListPageDataServices{
		AuthorizationService: nil,
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewGetStaffAttributeListPageDataUseCase(repositories, services)
}

// Execute performs the get staff attribute list page data operation
func (uc *GetStaffAttributeListPageDataUseCase) Execute(ctx context.Context, req *staffattributepb.GetStaffAttributeListPageDataRequest) (*staffattributepb.GetStaffAttributeListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityStaffAttribute, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Apply default values for pagination, filtering, sorting, and search
	if err := uc.applyDefaults(req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.apply_defaults_failed", "Failed to apply default values [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.StaffAttribute.GetStaffAttributeListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.list_page_data_failed", "Failed to retrieve staff attribute list page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetStaffAttributeListPageDataUseCase) validateInput(ctx context.Context, req *staffattributepb.GetStaffAttributeListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.request_required", "Request is required for staff attributes [DEFAULT]"))
	}

	// Validate pagination if provided
	if req.Pagination != nil {
		if req.Pagination.Limit < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.pagination_limit_invalid", "Pagination limit must be non-negative [DEFAULT]"))
		}
		if req.Pagination.Limit > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.pagination_limit_exceeded", "Pagination limit cannot exceed 1000 [DEFAULT]"))
		}
	}

	// Validate search if provided
	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) < 2 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.search_query_too_short", "Search query must be at least 2 characters [DEFAULT]"))
		}
	}

	return nil
}

// applyDefaults sets default values for optional parameters
func (uc *GetStaffAttributeListPageDataUseCase) applyDefaults(req *staffattributepb.GetStaffAttributeListPageDataRequest) error {
	// Apply default pagination if not provided
	if req.Pagination == nil {
		req.Pagination = &commonpb.PaginationRequest{
			Limit: 10, // Default page size
			Method: &commonpb.PaginationRequest_Offset{
				Offset: &commonpb.OffsetPagination{
					Page: 1, // Default to first page
				},
			},
		}
	} else if req.Pagination.Limit == 0 {
		req.Pagination.Limit = 10 // Default page size if not specified
	}

	return nil
}
