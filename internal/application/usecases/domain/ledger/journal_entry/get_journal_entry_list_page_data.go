package journalentry

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	journalentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_entry"
)

// GetJournalEntryListPageDataRepositories groups all repository dependencies
type GetJournalEntryListPageDataRepositories struct {
	JournalEntry journalentrypb.JournalEntryDomainServiceServer // Primary entity repository
}

// GetJournalEntryListPageDataServices groups all business service dependencies
type GetJournalEntryListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetJournalEntryListPageDataUseCase handles the business logic for getting journal entry list page data
// with pagination, filtering, sorting, and search
type GetJournalEntryListPageDataUseCase struct {
	repositories GetJournalEntryListPageDataRepositories
	services     GetJournalEntryListPageDataServices
}

// NewGetJournalEntryListPageDataUseCase creates use case with grouped dependencies
func NewGetJournalEntryListPageDataUseCase(
	repositories GetJournalEntryListPageDataRepositories,
	services GetJournalEntryListPageDataServices,
) *GetJournalEntryListPageDataUseCase {
	return &GetJournalEntryListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get journal entry list page data operation
func (uc *GetJournalEntryListPageDataUseCase) Execute(ctx context.Context, req *journalentrypb.GetJournalEntryListPageDataRequest) (*journalentrypb.GetJournalEntryListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityJournalEntry, entityid.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "journal_entry.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "journal_entry.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	if uc.repositories.JournalEntry == nil {
		return nil, errors.New("journal entry repository is not available")
	}
	resp, err := uc.repositories.JournalEntry.GetJournalEntryListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "journal_entry.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load journal entry list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetJournalEntryListPageDataUseCase) validateInput(ctx context.Context, req *journalentrypb.GetJournalEntryListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "journal_entry.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	// Validate pagination parameters
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 && (req.Pagination.Limit < 1 || req.Pagination.Limit > 100) {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "journal_entry.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
		}
	}

	// Validate filter parameters
	if req.Filters != nil && len(req.Filters.Filters) > 10 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "journal_entry.validation.too_many_filters", "[ERR-DEFAULT] Too many filters"))
	}

	// Validate sort parameters
	if req.Sort != nil && len(req.Sort.Fields) > 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "journal_entry.validation.too_many_sort_fields", "[ERR-DEFAULT] Too many sort fields"))
	}

	// Validate search parameters
	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "journal_entry.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
		}
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting list page data
func (uc *GetJournalEntryListPageDataUseCase) validateBusinessRules(ctx context.Context, req *journalentrypb.GetJournalEntryListPageDataRequest) error {
	// No additional business rules for getting journal entry list page data
	return nil
}
