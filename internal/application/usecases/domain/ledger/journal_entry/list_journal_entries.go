package journalentry

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	journalentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_entry"
)

// ListJournalEntriesRepositories groups all repository dependencies
type ListJournalEntriesRepositories struct {
	JournalEntry journalentrypb.JournalEntryDomainServiceServer // Primary entity repository
}

// ListJournalEntriesServices groups all business service dependencies
type ListJournalEntriesServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListJournalEntriesUseCase handles the business logic for listing journal entries
type ListJournalEntriesUseCase struct {
	repositories ListJournalEntriesRepositories
	services     ListJournalEntriesServices
}

// NewListJournalEntriesUseCase creates use case with grouped dependencies
func NewListJournalEntriesUseCase(
	repositories ListJournalEntriesRepositories,
	services ListJournalEntriesServices,
) *ListJournalEntriesUseCase {
	return &ListJournalEntriesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list journal entries operation
func (uc *ListJournalEntriesUseCase) Execute(ctx context.Context, req *journalentrypb.ListJournalEntriesRequest) (*journalentrypb.ListJournalEntriesResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityJournalEntry,
		Action: entityid.ActionList,
	}); err != nil {
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
	resp, err := uc.repositories.JournalEntry.ListJournalEntries(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "journal_entry.errors.list_failed", "[ERR-DEFAULT] Failed to list journal entries")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListJournalEntriesUseCase) validateInput(ctx context.Context, req *journalentrypb.ListJournalEntriesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "journal_entry.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for listing
func (uc *ListJournalEntriesUseCase) validateBusinessRules(ctx context.Context, req *journalentrypb.ListJournalEntriesRequest) error {
	// No additional business rules for listing journal entries
	return nil
}
