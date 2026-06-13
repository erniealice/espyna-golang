package work_request_type

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	work_request_typepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request_type"
)

// GetWorkRequestTypeListPageDataRepositories groups all repository dependencies
type GetWorkRequestTypeListPageDataRepositories struct {
	WorkRequestType work_request_typepb.WorkRequestTypeDomainServiceServer // Primary entity repository
}

// GetWorkRequestTypeListPageDataServices groups all business service dependencies
type GetWorkRequestTypeListPageDataServices struct {
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetWorkRequestTypeListPageDataUseCase handles the business logic for getting work request type list page data
type GetWorkRequestTypeListPageDataUseCase struct {
	repositories GetWorkRequestTypeListPageDataRepositories
	services     GetWorkRequestTypeListPageDataServices
}

// NewGetWorkRequestTypeListPageDataUseCase creates use case with grouped dependencies
func NewGetWorkRequestTypeListPageDataUseCase(
	repositories GetWorkRequestTypeListPageDataRepositories,
	services GetWorkRequestTypeListPageDataServices,
) *GetWorkRequestTypeListPageDataUseCase {
	return &GetWorkRequestTypeListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get work request type list page data operation
func (uc *GetWorkRequestTypeListPageDataUseCase) Execute(ctx context.Context, req *work_request_typepb.GetWorkRequestTypeListPageDataRequest) (*work_request_typepb.GetWorkRequestTypeListPageDataResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkRequestType,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.WorkRequestType.GetWorkRequestTypeListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load work request type list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetWorkRequestTypeListPageDataUseCase) validateInput(ctx context.Context, req *work_request_typepb.GetWorkRequestTypeListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	// Validate pagination parameters
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 && (req.Pagination.Limit < 1 || req.Pagination.Limit > 100) {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
		}
	}

	// Validate filter count
	if req.Filters != nil && len(req.Filters.Filters) > 10 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.too_many_filters", "[ERR-DEFAULT] Too many filters"))
	}

	// Validate sort field count
	if req.Sort != nil && len(req.Sort.Fields) > 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.too_many_sort_fields", "[ERR-DEFAULT] Too many sort fields"))
	}

	// Validate search query length
	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
		}
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting list page data
func (uc *GetWorkRequestTypeListPageDataUseCase) validateBusinessRules(ctx context.Context, req *work_request_typepb.GetWorkRequestTypeListPageDataRequest) error {
	// No additional business rules for getting list page data
	return nil
}
