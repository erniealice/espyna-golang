package group

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	grouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/group"
)

// GetGroupListPageDataUseCase handles the business logic for getting group list page data
// GetGroupListPageDataRepositories groups all repository dependencies
type GetGroupListPageDataRepositories struct {
	Group grouppb.GroupDomainServiceServer // Primary entity repository
}

// GetGroupListPageDataServices groups all business service dependencies
type GetGroupListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetGroupListPageDataUseCase handles the business logic for getting group list page data
type GetGroupListPageDataUseCase struct {
	repositories GetGroupListPageDataRepositories
	services     GetGroupListPageDataServices
}

// NewGetGroupListPageDataUseCase creates use case with grouped dependencies
func NewGetGroupListPageDataUseCase(
	repositories GetGroupListPageDataRepositories,
	services GetGroupListPageDataServices,
) *GetGroupListPageDataUseCase {
	return &GetGroupListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetGroupListPageDataUseCaseUngrouped creates a new GetGroupListPageDataUseCase
// Deprecated: Use NewGetGroupListPageDataUseCase with grouped parameters instead
func NewGetGroupListPageDataUseCaseUngrouped(groupRepo grouppb.GroupDomainServiceServer) *GetGroupListPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetGroupListPageDataRepositories{
		Group: groupRepo,
	}

	services := GetGroupListPageDataServices{
		AuthorizationService: nil,
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewGetGroupListPageDataUseCase(repositories, services)
}

// Execute performs the get group list page data operation
func (uc *GetGroupListPageDataUseCase) Execute(ctx context.Context, req *grouppb.GetGroupListPageDataRequest) (*grouppb.GetGroupListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityGroup, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Apply default values for pagination, filtering, sorting, and search
	if err := uc.applyDefaults(req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.errors.apply_defaults_failed", "Failed to apply default values [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Group.GetGroupListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.errors.list_page_data_failed", "Failed to retrieve group list page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetGroupListPageDataUseCase) validateInput(ctx context.Context, req *grouppb.GetGroupListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.request_required", "Request is required for groups [DEFAULT]"))
	}

	// Validate pagination if provided
	if req.Pagination != nil {
		if req.Pagination.Limit < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.pagination_limit_invalid", "Pagination limit must be non-negative [DEFAULT]"))
		}
		if req.Pagination.Limit > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.pagination_limit_exceeded", "Pagination limit cannot exceed 1000 [DEFAULT]"))
		}
	}

	// Validate search if provided
	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) < 2 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.search_query_too_short", "Search query must be at least 2 characters [DEFAULT]"))
		}
	}

	return nil
}

// applyDefaults sets default values for optional parameters
func (uc *GetGroupListPageDataUseCase) applyDefaults(req *grouppb.GetGroupListPageDataRequest) error {
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
