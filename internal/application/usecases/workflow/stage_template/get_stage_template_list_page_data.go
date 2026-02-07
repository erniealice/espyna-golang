package stage_template

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	"leapfor.xyz/espyna/internal/application/shared/listdata"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	stageTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage_template"
	workflowTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow_template"
)

type GetStageTemplateListPageDataRepositories struct {
	StageTemplate    stageTemplatepb.StageTemplateDomainServiceServer       // Primary entity repository
	WorkflowTemplate workflowTemplatepb.WorkflowTemplateDomainServiceServer // Foreign key reference
}

type GetStageTemplateListPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetStageTemplateListPageDataUseCase handles the business logic for getting stage template list page data
type GetStageTemplateListPageDataUseCase struct {
	repositories GetStageTemplateListPageDataRepositories
	services     GetStageTemplateListPageDataServices
	processor    *listdata.ListDataProcessor
}

// NewGetStageTemplateListPageDataUseCase creates a new GetStageTemplateListPageDataUseCase
func NewGetStageTemplateListPageDataUseCase(
	repositories GetStageTemplateListPageDataRepositories,
	services GetStageTemplateListPageDataServices,
) *GetStageTemplateListPageDataUseCase {
	return &GetStageTemplateListPageDataUseCase{
		repositories: repositories,
		services:     services,
		processor:    listdata.NewListDataProcessor(),
	}
}

// Execute performs the get stage template list page data operation
func (uc *GetStageTemplateListPageDataUseCase) Execute(
	ctx context.Context,
	req *stageTemplatepb.GetStageTemplateListPageDataRequest,
) (*stageTemplatepb.GetStageTemplateListPageDataResponse, error) {
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

// executeWithTransaction executes stage template list page data retrieval within a transaction
func (uc *GetStageTemplateListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *stageTemplatepb.GetStageTemplateListPageDataRequest,
) (*stageTemplatepb.GetStageTemplateListPageDataResponse, error) {
	var result *stageTemplatepb.GetStageTemplateListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"stage_template.errors.list_page_data_failed",
				"stage template list page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting stage template list page data
func (uc *GetStageTemplateListPageDataUseCase) executeCore(
	ctx context.Context,
	req *stageTemplatepb.GetStageTemplateListPageDataRequest,
) (*stageTemplatepb.GetStageTemplateListPageDataResponse, error) {
	// First, get all stage templates from the repository
	listReq := &stageTemplatepb.ListStageTemplatesRequest{}
	listResp, err := uc.repositories.StageTemplate.ListStageTemplates(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"stage_template.errors.list_failed",
			"failed to retrieve stage templates: %w",
		), err)
	}

	if listResp == nil || len(listResp.Data) == 0 {
		// Return empty response with proper pagination metadata
		emptyPagination := uc.processor.GetPaginationUtils().CreatePaginationResponse(req.Pagination, 0, false)
		return &stageTemplatepb.GetStageTemplateListPageDataResponse{
			StageTemplateList: []*stageTemplatepb.StageTemplate{},
			Pagination:        emptyPagination,
			SearchResults:     []*commonpb.SearchResult{},
			Success:           true,
		}, nil
	}

	// Process the data with filtering, sorting, searching, and pagination
	result, err := uc.processor.ProcessListRequest(
		listResp.Data,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"stage_template.errors.processing_failed",
			"failed to process stage template list data: %w",
		), err)
	}

	// Convert processed items back to stage template protobuf format
	stageTemplates := make([]*stageTemplatepb.StageTemplate, len(result.Items))
	for i, item := range result.Items {
		if stageTemplate, ok := item.(*stageTemplatepb.StageTemplate); ok {
			stageTemplates[i] = stageTemplate
		} else {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"stage_template.errors.type_conversion_failed",
				"failed to convert item to stage template type",
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

	return &stageTemplatepb.GetStageTemplateListPageDataResponse{
		StageTemplateList: stageTemplates,
		Pagination:        result.PaginationResponse,
		SearchResults:     searchResults,
		Success:           true,
	}, nil
}

// validateInput validates the input request
func (uc *GetStageTemplateListPageDataUseCase) validateInput(
	ctx context.Context,
	req *stageTemplatepb.GetStageTemplateListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"stage_template.validation.request_required",
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
func (uc *GetStageTemplateListPageDataUseCase) validatePagination(
	ctx context.Context,
	pagination *commonpb.PaginationRequest,
) error {
	if pagination.Limit < 0 || pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"stage_template.validation.invalid_limit",
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
				"stage_template.validation.invalid_page",
				"page number must be greater than 0",
			))
		}
	case *commonpb.PaginationRequest_Cursor:
		// Cursor validation could be more sophisticated
		if method.Cursor.Token == "" {
			return errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"stage_template.validation.invalid_cursor",
				"cursor token cannot be empty",
			))
		}
	}

	return nil
}

// validateFilters validates filter parameters
func (uc *GetStageTemplateListPageDataUseCase) validateFilters(
	ctx context.Context,
	filters *commonpb.FilterRequest,
) error {
	if len(filters.Filters) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"stage_template.validation.empty_filters",
			"filters cannot be empty when filter request is provided",
		))
	}

	// Validate individual filters
	for i, filter := range filters.Filters {
		if filter.Field == "" {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"stage_template.validation.filter_field_required",
				"filter field is required for filter %d",
			), i)
		}

		// Validate that the field exists on the stage template entity
		if !uc.isValidStageTemplateField(filter.Field) {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"stage_template.validation.invalid_filter_field",
				"invalid filter field: %s",
			), filter.Field)
		}
	}

	return nil
}

// validateSort validates sort parameters
func (uc *GetStageTemplateListPageDataUseCase) validateSort(
	ctx context.Context,
	sort *commonpb.SortRequest,
) error {
	if len(sort.Fields) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"stage_template.validation.empty_sort_fields",
			"sort fields cannot be empty when sort request is provided",
		))
	}

	// Validate individual sort fields
	for i, sortField := range sort.Fields {
		if sortField.Field == "" {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"stage_template.validation.sort_field_required",
				"sort field is required for sort field %d",
			), i)
		}

		// Validate that the field exists on the stage template entity
		if !uc.isValidStageTemplateField(sortField.Field) {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"stage_template.validation.invalid_sort_field",
				"invalid sort field: %s",
			), sortField.Field)
		}
	}

	return nil
}

// validateSearch validates search parameters
func (uc *GetStageTemplateListPageDataUseCase) validateSearch(
	ctx context.Context,
	search *commonpb.SearchRequest,
) error {
	if search.Query == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"stage_template.validation.empty_search_query",
			"search query cannot be empty when search request is provided",
		))
	}

	if search.Options != nil {
		// Validate search fields if specified
		for _, field := range search.Options.SearchFields {
			if !uc.isValidStageTemplateField(field) {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
					ctx,
					uc.services.TranslationService,
					"stage_template.validation.invalid_search_field",
					"invalid search field: %s",
				), field)
			}
		}

		// Validate max results
		if search.Options.MaxResults < 0 || search.Options.MaxResults > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"stage_template.validation.invalid_max_results",
				"max results must be between 0 and 1000",
			))
		}
	}

	return nil
}

// isValidStageTemplateField checks if a field name is valid for stage template filtering/sorting/searching
func (uc *GetStageTemplateListPageDataUseCase) isValidStageTemplateField(field string) bool {
	validFields := map[string]bool{
		"id":                   true,
		"name":                 true,
		"description":          true,
		"workflow_id":          true,
		"status":               true,
		"stage_type":           true,
		"order_index":          true,
		"is_required":          true,
		"condition_expression": true,
		"created_by":           true,
		"date_created":         true,
		"date_created_string":  true,
		"date_modified":        true,
		"date_modified_string": true,
		"active":               true,
	}

	return validFields[field]
}
