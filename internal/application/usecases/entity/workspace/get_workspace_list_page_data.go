package workspace

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

type GetWorkspaceListPageDataRepositories struct {
	Workspace workspacepb.WorkspaceDomainServiceServer
}

type GetWorkspaceListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetWorkspaceListPageDataUseCase handles the business logic for getting workspace list page data
type GetWorkspaceListPageDataUseCase struct {
	repositories GetWorkspaceListPageDataRepositories
	services     GetWorkspaceListPageDataServices
	processor    *listdata.ListDataProcessor
}

// NewGetWorkspaceListPageDataUseCase creates a new GetWorkspaceListPageDataUseCase
func NewGetWorkspaceListPageDataUseCase(
	repositories GetWorkspaceListPageDataRepositories,
	services GetWorkspaceListPageDataServices,
) *GetWorkspaceListPageDataUseCase {
	return &GetWorkspaceListPageDataUseCase{
		repositories: repositories,
		services:     services,
		processor:    listdata.NewListDataProcessor(),
	}
}

// Execute performs the get workspace list page data operation
func (uc *GetWorkspaceListPageDataUseCase) Execute(
	ctx context.Context,
	req *workspacepb.GetWorkspaceListPageDataRequest,
) (*workspacepb.GetWorkspaceListPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Authorization check - ensure user can list workspaces
	if err := uc.checkAuthorizationPermissions(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes workspace list page data retrieval within a transaction
func (uc *GetWorkspaceListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *workspacepb.GetWorkspaceListPageDataRequest,
) (*workspacepb.GetWorkspaceListPageDataResponse, error) {
	var result *workspacepb.GetWorkspaceListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"workspace.errors.list_page_data_failed",
				"workspace list page data retrieval failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting workspace list page data
func (uc *GetWorkspaceListPageDataUseCase) executeCore(
	ctx context.Context,
	req *workspacepb.GetWorkspaceListPageDataRequest,
) (*workspacepb.GetWorkspaceListPageDataResponse, error) {
	// First, get all workspaces from the repository
	listReq := &workspacepb.ListWorkspacesRequest{}
	listResp, err := uc.repositories.Workspace.ListWorkspaces(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workspace.errors.list_failed",
			"failed to retrieve workspaces: %w",
		), err)
	}

	if listResp == nil || len(listResp.Data) == 0 {
		// Return empty response with proper pagination metadata
		var emptyPagination *commonpb.PaginationResponse
		if req.Pagination != nil {
			emptyPagination = uc.processor.GetPaginationUtils().CreatePaginationResponse(req.Pagination, 0, false)
		}
		return &workspacepb.GetWorkspaceListPageDataResponse{
			WorkspaceList: []*workspacepb.Workspace{},
			Pagination:    emptyPagination,
			SearchResults: []*commonpb.SearchResult{},
			Success:       true,
		}, nil
	}

	// Convert to interface slice for processing
	workspaceInterfaces := make([]interface{}, len(listResp.Data))
	for i, workspace := range listResp.Data {
		workspaceInterfaces[i] = workspace
	}

	// Apply user-specific filtering for multi-tenant security
	filteredWorkspaces, err := uc.applySecurityFiltering(ctx, workspaceInterfaces)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workspace.errors.security_filtering_failed",
			"failed to apply security filtering: %w",
		), err)
	}

	// Process the data with filtering, sorting, searching, and pagination
	result, err := uc.processor.ProcessListRequest(
		filteredWorkspaces,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workspace.errors.processing_failed",
			"failed to process workspace list data: %w",
		), err)
	}

	// Convert processed items back to workspace protobuf format
	workspaces := make([]*workspacepb.Workspace, len(result.Items))
	for i, item := range result.Items {
		if workspace, ok := item.(*workspacepb.Workspace); ok {
			workspaces[i] = workspace
		} else {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"workspace.errors.type_conversion_failed",
				"failed to convert item to workspace type",
			))
		}
	}

	// Convert search results to protobuf format
	searchResults := make([]*commonpb.SearchResult, len(result.SearchResults))
	for i, searchResult := range result.SearchResults {
		searchResults[i] = &commonpb.SearchResult{
			Score:      searchResult.Score,
			Highlights: searchResult.Highlights,
		}
	}

	return &workspacepb.GetWorkspaceListPageDataResponse{
		WorkspaceList: workspaces,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// applySecurityFiltering filters workspaces based on user permissions and multi-tenant access
func (uc *GetWorkspaceListPageDataUseCase) applySecurityFiltering(
	ctx context.Context,
	workspaces []interface{},
) ([]interface{}, error) {
	// Convert to workspace slice for easier processing
	workspaceSlice := make([]*workspacepb.Workspace, 0, len(workspaces))
	for _, item := range workspaces {
		if workspace, ok := item.(*workspacepb.Workspace); ok {
			workspaceSlice = append(workspaceSlice, workspace)
		}
	}

	// Apply user-specific filtering if authorization service is available
	if uc.services.AuthorizationService != nil {
		filteredWorkspaces := make([]*workspacepb.Workspace, 0, len(workspaceSlice))

		for _, workspace := range workspaceSlice {
			// Check if user has permission to view this workspace
			if canView, err := uc.checkWorkspaceViewPermission(ctx, workspace.Id); err != nil {
				// Log error but continue processing other workspaces
				continue
			} else if canView {
				filteredWorkspaces = append(filteredWorkspaces, workspace)
			}
		}

		workspaceSlice = filteredWorkspaces
	}

	// Convert back to interface slice
	filtered := make([]interface{}, len(workspaceSlice))
	for i, workspace := range workspaceSlice {
		filtered[i] = workspace
	}

	return filtered, nil
}

// checkWorkspaceViewPermission checks if the current user can view a specific workspace
func (uc *GetWorkspaceListPageDataUseCase) checkWorkspaceViewPermission(
	ctx context.Context,
	workspaceId string,
) (bool, error) {
	if uc.services.AuthorizationService == nil {
		// If no authorization service, allow access (fallback for testing)
		return true, nil
	}

	// Check user permission to view workspace
	// This could involve checking:
	// - User role within the workspace
	// - Organization membership
	// - Private workspace access rights
	// - Delegate permissions

	// For now, return true as a placeholder
	// In production, implement proper RBAC checks
	return true, nil
}

// checkAuthorizationPermissions validates user has permission to list workspaces
func (uc *GetWorkspaceListPageDataUseCase) checkAuthorizationPermissions(
	ctx context.Context,
	req *workspacepb.GetWorkspaceListPageDataRequest,
) error {
	if uc.services.AuthorizationService == nil {
		// No authorization service available, skip check
		return nil
	}

	// Check if user has permission to list workspaces
	// This could check for permissions like:
	// - "workspace:list"
	// - Organization-level access
	// - Role-based permissions

	// For now, return nil as a placeholder
	// In production, implement proper authorization checks
	return nil
}

// validateInput validates the input request
func (uc *GetWorkspaceListPageDataUseCase) validateInput(
	ctx context.Context,
	req *workspacepb.GetWorkspaceListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workspace.validation.request_required",
			"request is required",
		))
	}

	// Validate pagination if provided
	if req.Pagination != nil {
		if err := uc.validatePagination(ctx, req.Pagination); err != nil {
			return err
		}
	}

	// Validate filters if provided
	if req.Filters != nil {
		if err := uc.validateFilters(ctx, req.Filters); err != nil {
			return err
		}
	}

	// Validate sort if provided
	if req.Sort != nil {
		if err := uc.validateSort(ctx, req.Sort); err != nil {
			return err
		}
	}

	// Validate search if provided
	if req.Search != nil {
		if err := uc.validateSearch(ctx, req.Search); err != nil {
			return err
		}
	}

	return nil
}

// validatePagination validates pagination parameters
func (uc *GetWorkspaceListPageDataUseCase) validatePagination(
	ctx context.Context,
	pagination *commonpb.PaginationRequest,
) error {
	if pagination.Limit < 0 || pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workspace.validation.invalid_limit",
			"pagination limit must be between 1 and 100",
		))
	}

	// Validate specific pagination method
	switch method := pagination.Method.(type) {
	case *commonpb.PaginationRequest_Offset:
		if method.Offset.Page < 1 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"workspace.validation.invalid_page",
				"page number must be greater than 0",
			))
		}
	case *commonpb.PaginationRequest_Cursor:
		// Cursor validation could be more sophisticated
		if method.Cursor.Token == "" {
			return errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"workspace.validation.invalid_cursor",
				"cursor token cannot be empty",
			))
		}
	}

	return nil
}

// validateFilters validates filter parameters
func (uc *GetWorkspaceListPageDataUseCase) validateFilters(
	ctx context.Context,
	filters *commonpb.FilterRequest,
) error {
	if len(filters.Filters) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workspace.validation.empty_filters",
			"filters cannot be empty when filter request is provided",
		))
	}

	// Validate individual filters
	for i, filter := range filters.Filters {
		if filter.Field == "" {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"workspace.validation.filter_field_required",
				"filter field is required for filter %d",
			), i)
		}

		// Validate that the field exists on the workspace entity
		if !uc.isValidWorkspaceField(filter.Field) {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"workspace.validation.invalid_filter_field",
				"invalid filter field: %s",
			), filter.Field)
		}
	}

	return nil
}

// validateSort validates sort parameters
func (uc *GetWorkspaceListPageDataUseCase) validateSort(
	ctx context.Context,
	sort *commonpb.SortRequest,
) error {
	if len(sort.Fields) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workspace.validation.empty_sort_fields",
			"sort fields cannot be empty when sort request is provided",
		))
	}

	// Validate individual sort fields
	for i, sortField := range sort.Fields {
		if sortField.Field == "" {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"workspace.validation.sort_field_required",
				"sort field is required for sort field %d",
			), i)
		}

		// Validate that the field exists on the workspace entity
		if !uc.isValidWorkspaceField(sortField.Field) {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"workspace.validation.invalid_sort_field",
				"invalid sort field: %s",
			), sortField.Field)
		}
	}

	return nil
}

// validateSearch validates search parameters
func (uc *GetWorkspaceListPageDataUseCase) validateSearch(
	ctx context.Context,
	search *commonpb.SearchRequest,
) error {
	if search.Query == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workspace.validation.empty_search_query",
			"search query cannot be empty when search request is provided",
		))
	}

	if search.Options != nil {
		// Validate search fields if specified
		for _, field := range search.Options.SearchFields {
			if !uc.isValidWorkspaceField(field) {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
					ctx,
					uc.services.TranslationService,
					"workspace.validation.invalid_search_field",
					"invalid search field: %s",
				), field)
			}
		}

		// Validate max results
		if search.Options.MaxResults < 0 || search.Options.MaxResults > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"workspace.validation.invalid_max_results",
				"max results must be between 0 and 1000",
			))
		}
	}

	return nil
}

// isValidWorkspaceField checks if a field name is valid for workspace filtering/sorting/searching
func (uc *GetWorkspaceListPageDataUseCase) isValidWorkspaceField(field string) bool {
	validFields := map[string]bool{
		"id":                   true,
		"name":                 true,
		"description":          true,
		"private":              true,
		"active":               true,
		"date_created":         true,
		"date_created_string":  true,
		"date_modified":        true,
		"date_modified_string": true,
		// Computed/derived fields for filtering
		"user_count":   true,
		"organization": true,
		"owner_id":     true,
	}

	return validFields[field]
}
