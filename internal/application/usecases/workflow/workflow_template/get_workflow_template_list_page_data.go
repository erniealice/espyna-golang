package workflow_template

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	workflow_templatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow_template"
)

type GetWorkflowTemplateListPageDataRepositories struct {
	WorkflowTemplate workflow_templatepb.WorkflowTemplateDomainServiceServer
}

type GetWorkflowTemplateListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetWorkflowTemplateListPageDataUseCase handles the business logic for getting workflow template list page data
type GetWorkflowTemplateListPageDataUseCase struct {
	repositories GetWorkflowTemplateListPageDataRepositories
	services     GetWorkflowTemplateListPageDataServices
	processor    *listdata.ListDataProcessor
}

// NewGetWorkflowTemplateListPageDataUseCase creates a new GetWorkflowTemplateListPageDataUseCase
func NewGetWorkflowTemplateListPageDataUseCase(
	repositories GetWorkflowTemplateListPageDataRepositories,
	services GetWorkflowTemplateListPageDataServices,
) *GetWorkflowTemplateListPageDataUseCase {
	return &GetWorkflowTemplateListPageDataUseCase{
		repositories: repositories,
		services:     services,
		processor:    listdata.NewListDataProcessor(),
	}
}

// Execute performs the get workflow template list page data operation
func (uc *GetWorkflowTemplateListPageDataUseCase) Execute(
	ctx context.Context,
	req *workflow_templatepb.GetWorkflowTemplateListPageDataRequest,
) (*workflow_templatepb.GetWorkflowTemplateListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"workflow_template", ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes workflow template list page data retrieval within a transaction
func (uc *GetWorkflowTemplateListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *workflow_templatepb.GetWorkflowTemplateListPageDataRequest,
) (*workflow_templatepb.GetWorkflowTemplateListPageDataResponse, error) {
	var result *workflow_templatepb.GetWorkflowTemplateListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"workflow_template.errors.list_page_data_failed",
				"workflow template list page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting workflow template list page data
func (uc *GetWorkflowTemplateListPageDataUseCase) executeCore(
	ctx context.Context,
	req *workflow_templatepb.GetWorkflowTemplateListPageDataRequest,
) (*workflow_templatepb.GetWorkflowTemplateListPageDataResponse, error) {
	// First, get all workflow templates from the repository
	listReq := &workflow_templatepb.ListWorkflowTemplatesRequest{}
	listResp, err := uc.repositories.WorkflowTemplate.ListWorkflowTemplates(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workflow_template.errors.list_failed",
			"failed to retrieve workflow templates: %w",
		), err)
	}

	if listResp == nil || len(listResp.Data) == 0 {
		// Return empty response with proper pagination metadata
		emptyPagination := uc.processor.GetPaginationUtils().CreatePaginationResponse(req.Pagination, 0, false)
		return &workflow_templatepb.GetWorkflowTemplateListPageDataResponse{
			WorkflowTemplateList: []*workflow_templatepb.WorkflowTemplate{},
			Pagination:           emptyPagination,
			SearchResults:        []*commonpb.SearchResult{},
			Success:              true,
		}, nil
	}

	// Convert to slice for processing
	workflowTemplates := make([]interface{}, len(listResp.Data))
	for i, item := range listResp.Data {
		workflowTemplates[i] = item
	}

	// Process the data with filtering, sorting, searching, and pagination
	result, err := uc.processor.ProcessListRequest(
		workflowTemplates,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workflow_template.errors.processing_failed",
			"failed to process workflow template list data: %w",
		), err)
	}

	// Convert processed items back to workflow template protobuf format
	processedWorkflowTemplates := make([]*workflow_templatepb.WorkflowTemplate, len(result.Items))
	for i, item := range result.Items {
		if workflowTemplate, ok := item.(*workflow_templatepb.WorkflowTemplate); ok {
			processedWorkflowTemplates[i] = workflowTemplate
		} else {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"workflow_template.errors.type_conversion_failed",
				"failed to convert item to workflow template type",
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

	return &workflow_templatepb.GetWorkflowTemplateListPageDataResponse{
		WorkflowTemplateList: processedWorkflowTemplates,
		Pagination:           result.PaginationResponse,
		SearchResults:        searchResults,
		Success:              true,
	}, nil
}

// validateInput validates the input request
func (uc *GetWorkflowTemplateListPageDataUseCase) validateInput(
	ctx context.Context,
	req *workflow_templatepb.GetWorkflowTemplateListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workflow_template.validation.request_required",
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
func (uc *GetWorkflowTemplateListPageDataUseCase) validatePagination(
	ctx context.Context,
	pagination *commonpb.PaginationRequest,
) error {
	if pagination.Limit < 0 || pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workflow_template.validation.invalid_limit",
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
				"workflow_template.validation.invalid_page",
				"page number must be greater than 0",
			))
		}
	case *commonpb.PaginationRequest_Cursor:
		// Cursor validation could be more sophisticated
		if method.Cursor.Token == "" {
			return errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"workflow_template.validation.invalid_cursor",
				"cursor token cannot be empty",
			))
		}
	}

	return nil
}

// validateFilters validates filter parameters
func (uc *GetWorkflowTemplateListPageDataUseCase) validateFilters(
	ctx context.Context,
	filters *commonpb.FilterRequest,
) error {
	if len(filters.Filters) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workflow_template.validation.empty_filters",
			"filters cannot be empty when filter request is provided",
		))
	}

	// Validate individual filters
	for i, filter := range filters.Filters {
		if filter.Field == "" {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"workflow_template.validation.filter_field_required",
				"filter field is required for filter %d",
			), i)
		}

		// Validate that the field exists on the workflow template entity
		if !uc.isValidWorkflowTemplateField(filter.Field) {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"workflow_template.validation.invalid_filter_field",
				"invalid filter field: %s",
			), filter.Field)
		}
	}

	return nil
}

// validateSort validates sort parameters
func (uc *GetWorkflowTemplateListPageDataUseCase) validateSort(
	ctx context.Context,
	sort *commonpb.SortRequest,
) error {
	if len(sort.Fields) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workflow_template.validation.empty_sort_fields",
			"sort fields cannot be empty when sort request is provided",
		))
	}

	// Validate individual sort fields
	for i, sortField := range sort.Fields {
		if sortField.Field == "" {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"workflow_template.validation.sort_field_required",
				"sort field is required for sort field %d",
			), i)
		}

		// Validate that the field exists on the workflow template entity
		if !uc.isValidWorkflowTemplateField(sortField.Field) {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"workflow_template.validation.invalid_sort_field",
				"invalid sort field: %s",
			), sortField.Field)
		}
	}

	return nil
}

// validateSearch validates search parameters
func (uc *GetWorkflowTemplateListPageDataUseCase) validateSearch(
	ctx context.Context,
	search *commonpb.SearchRequest,
) error {
	if search.Query == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workflow_template.validation.empty_search_query",
			"search query cannot be empty when search request is provided",
		))
	}

	if search.Options != nil {
		// Validate search fields if specified
		for _, field := range search.Options.SearchFields {
			if !uc.isValidWorkflowTemplateField(field) {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
					ctx,
					uc.services.TranslationService,
					"workflow_template.validation.invalid_search_field",
					"invalid search field: %s",
				), field)
			}
		}

		// Validate max results
		if search.Options.MaxResults < 0 || search.Options.MaxResults > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"workflow_template.validation.invalid_max_results",
				"max results must be between 0 and 1000",
			))
		}
	}

	return nil
}

// isValidWorkflowTemplateField checks if a field name is valid for workflow template filtering/sorting/searching
func (uc *GetWorkflowTemplateListPageDataUseCase) isValidWorkflowTemplateField(field string) bool {
	validFields := map[string]bool{
		"id":                   true,
		"name":                 true,
		"description":          true,
		"status":               true,
		"business_type":        true,
		"workspace_id":         true,
		"created_by":           true,
		"date_created":         true,
		"date_created_string":  true,
		"date_modified":        true,
		"date_modified_string": true,
		"active":               true,
		"version":              true,
		"configuration_json":   true,
	}

	return validFields[field]
}
