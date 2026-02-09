package location

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
)

// GetLocationListPageDataRepositories groups all repository dependencies
type GetLocationListPageDataRepositories struct {
	Location locationpb.LocationDomainServiceServer // Primary entity repository
}

// GetLocationListPageDataServices groups all business service dependencies
type GetLocationListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetLocationListPageDataUseCase handles the business logic for getting location list page data with pagination, filtering, sorting, and search
type GetLocationListPageDataUseCase struct {
	repositories GetLocationListPageDataRepositories
	services     GetLocationListPageDataServices
}

// NewGetLocationListPageDataUseCase creates use case with grouped dependencies
func NewGetLocationListPageDataUseCase(
	repositories GetLocationListPageDataRepositories,
	services GetLocationListPageDataServices,
) *GetLocationListPageDataUseCase {
	return &GetLocationListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get location list page data operation
func (uc *GetLocationListPageDataUseCase) Execute(ctx context.Context, req *locationpb.GetLocationListPageDataRequest) (*locationpb.GetLocationListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityLocation, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.errors.input_validation_failed", "")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.errors.business_rule_validation_failed", "")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Location.GetLocationListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.errors.get_list_page_data_failed", "")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetLocationListPageDataUseCase) validateInput(ctx context.Context, req *locationpb.GetLocationListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.request_required", ""))
	}

	// Validate pagination parameters
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 && (req.Pagination.Limit < 1 || req.Pagination.Limit > 100) {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.invalid_pagination_limit", ""))
		}
	}

	// Validate filter parameters
	if req.Filters != nil && len(req.Filters.Filters) > 10 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.too_many_filters", ""))
	}

	// Validate sort parameters
	if req.Sort != nil && len(req.Sort.Fields) > 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.too_many_sort_fields", ""))
	}

	// Validate search parameters
	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.search_query_too_long", ""))
		}
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting list page data
func (uc *GetLocationListPageDataUseCase) validateBusinessRules(ctx context.Context, req *locationpb.GetLocationListPageDataRequest) error {
	// Check authorization for viewing locations
	// This would typically involve checking user permissions
	// For now, we'll allow all authenticated users to view location lists

	return nil
}
