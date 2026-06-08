package fiscalperiod

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	fiscalperiodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/fiscal_period"
)

// GetFiscalPeriodListPageDataRepositories groups all repository dependencies
type GetFiscalPeriodListPageDataRepositories struct {
	FiscalPeriod fiscalperiodpb.FiscalPeriodDomainServiceServer // Primary entity repository
}

// GetFiscalPeriodListPageDataServices groups all business service dependencies
type GetFiscalPeriodListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetFiscalPeriodListPageDataUseCase handles the business logic for getting fiscal period list page data
// with pagination, filtering, sorting, and search
type GetFiscalPeriodListPageDataUseCase struct {
	repositories GetFiscalPeriodListPageDataRepositories
	services     GetFiscalPeriodListPageDataServices
}

// NewGetFiscalPeriodListPageDataUseCase creates use case with grouped dependencies
func NewGetFiscalPeriodListPageDataUseCase(
	repositories GetFiscalPeriodListPageDataRepositories,
	services GetFiscalPeriodListPageDataServices,
) *GetFiscalPeriodListPageDataUseCase {
	return &GetFiscalPeriodListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get fiscal period list page data operation
func (uc *GetFiscalPeriodListPageDataUseCase) Execute(ctx context.Context, req *fiscalperiodpb.GetFiscalPeriodListPageDataRequest) (*fiscalperiodpb.GetFiscalPeriodListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityFiscalPeriod, entityid.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	if uc.repositories.FiscalPeriod == nil {
		return nil, errors.New("fiscal period repository is not available")
	}
	resp, err := uc.repositories.FiscalPeriod.GetFiscalPeriodListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load fiscal period list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetFiscalPeriodListPageDataUseCase) validateInput(ctx context.Context, req *fiscalperiodpb.GetFiscalPeriodListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	// Validate pagination parameters
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 && (req.Pagination.Limit < 1 || req.Pagination.Limit > 100) {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
		}
	}

	// Validate filter parameters
	if req.Filters != nil && len(req.Filters.Filters) > 10 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.validation.too_many_filters", "[ERR-DEFAULT] Too many filters"))
	}

	// Validate sort parameters
	if req.Sort != nil && len(req.Sort.Fields) > 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.validation.too_many_sort_fields", "[ERR-DEFAULT] Too many sort fields"))
	}

	// Validate search parameters
	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fiscal_period.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
		}
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting list page data
func (uc *GetFiscalPeriodListPageDataUseCase) validateBusinessRules(ctx context.Context, req *fiscalperiodpb.GetFiscalPeriodListPageDataRequest) error {
	// No additional business rules for getting fiscal period list page data
	return nil
}
