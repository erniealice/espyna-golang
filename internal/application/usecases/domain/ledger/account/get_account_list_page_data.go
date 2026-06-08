package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	accountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account"
)

// GetAccountListPageDataRepositories groups all repository dependencies
type GetAccountListPageDataRepositories struct {
	Account accountpb.AccountDomainServiceServer // Primary entity repository
}

// GetAccountListPageDataServices groups all business service dependencies
type GetAccountListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetAccountListPageDataUseCase handles the business logic for getting account list page data
// with pagination, filtering, sorting, and search
type GetAccountListPageDataUseCase struct {
	repositories GetAccountListPageDataRepositories
	services     GetAccountListPageDataServices
}

// NewGetAccountListPageDataUseCase creates use case with grouped dependencies
func NewGetAccountListPageDataUseCase(
	repositories GetAccountListPageDataRepositories,
	services GetAccountListPageDataServices,
) *GetAccountListPageDataUseCase {
	return &GetAccountListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get account list page data operation
func (uc *GetAccountListPageDataUseCase) Execute(ctx context.Context, req *accountpb.GetAccountListPageDataRequest) (*accountpb.GetAccountListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityAccount, entityid.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "account.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "account.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	if uc.repositories.Account == nil {
		return nil, errors.New("account repository is not available")
	}
	resp, err := uc.repositories.Account.GetAccountListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "account.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load account list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetAccountListPageDataUseCase) validateInput(ctx context.Context, req *accountpb.GetAccountListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "account.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	// Validate pagination parameters
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 && (req.Pagination.Limit < 1 || req.Pagination.Limit > 100) {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "account.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
		}
	}

	// Validate filter parameters
	if req.Filters != nil && len(req.Filters.Filters) > 10 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "account.validation.too_many_filters", "[ERR-DEFAULT] Too many filters"))
	}

	// Validate sort parameters
	if req.Sort != nil && len(req.Sort.Fields) > 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "account.validation.too_many_sort_fields", "[ERR-DEFAULT] Too many sort fields"))
	}

	// Validate search parameters
	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "account.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
		}
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting list page data
func (uc *GetAccountListPageDataUseCase) validateBusinessRules(ctx context.Context, req *accountpb.GetAccountListPageDataRequest) error {
	// No additional business rules for getting account list page data
	return nil
}
