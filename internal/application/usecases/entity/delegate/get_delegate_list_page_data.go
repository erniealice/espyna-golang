package delegate

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
)

// GetDelegateListPageDataRepositories groups all repository dependencies
type GetDelegateListPageDataRepositories struct {
	Delegate delegatepb.DelegateDomainServiceServer // Primary entity repository
}

// GetDelegateListPageDataServices groups all business service dependencies
type GetDelegateListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetDelegateListPageDataUseCase handles the business logic for getting delegate list page data with pagination, filtering, sorting, and search
type GetDelegateListPageDataUseCase struct {
	repositories GetDelegateListPageDataRepositories
	services     GetDelegateListPageDataServices
}

// NewGetDelegateListPageDataUseCase creates use case with grouped dependencies
func NewGetDelegateListPageDataUseCase(
	repositories GetDelegateListPageDataRepositories,
	services GetDelegateListPageDataServices,
) *GetDelegateListPageDataUseCase {
	return &GetDelegateListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get delegate list page data operation
func (uc *GetDelegateListPageDataUseCase) Execute(ctx context.Context, req *delegatepb.GetDelegateListPageDataRequest) (*delegatepb.GetDelegateListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityDelegate, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Delegate.GetDelegateListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load delegate list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetDelegateListPageDataUseCase) validateInput(ctx context.Context, req *delegatepb.GetDelegateListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	// Validate pagination parameters
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 && (req.Pagination.Limit < 1 || req.Pagination.Limit > 100) {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
		}
	}

	// Validate filter parameters
	if req.Filters != nil && len(req.Filters.Filters) > 10 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.too_many_filters", "[ERR-DEFAULT] Too many filters"))
	}

	// Validate sort parameters
	if req.Sort != nil && len(req.Sort.Fields) > 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.too_many_sort_fields", "[ERR-DEFAULT] Too many sort fields"))
	}

	// Validate search parameters
	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
		}
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting list page data
func (uc *GetDelegateListPageDataUseCase) validateBusinessRules(ctx context.Context, req *delegatepb.GetDelegateListPageDataRequest) error {
	// Check authorization for viewing delegates
	// This would typically involve checking user permissions
	// For now, we'll allow all authenticated users to view delegate lists

	return nil
}
