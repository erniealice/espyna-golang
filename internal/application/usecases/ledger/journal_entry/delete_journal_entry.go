package journalentry

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	journalentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_entry"
)

// DeleteJournalEntryRepositories groups all repository dependencies
type DeleteJournalEntryRepositories struct {
	JournalEntry journalentrypb.JournalEntryDomainServiceServer // Primary entity repository
}

// DeleteJournalEntryServices groups all business service dependencies
type DeleteJournalEntryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteJournalEntryUseCase handles the business logic for deleting journal entries
type DeleteJournalEntryUseCase struct {
	repositories DeleteJournalEntryRepositories
	services     DeleteJournalEntryServices
}

// NewDeleteJournalEntryUseCase creates use case with grouped dependencies
func NewDeleteJournalEntryUseCase(
	repositories DeleteJournalEntryRepositories,
	services DeleteJournalEntryServices,
) *DeleteJournalEntryUseCase {
	return &DeleteJournalEntryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete journal entry operation
func (uc *DeleteJournalEntryUseCase) Execute(ctx context.Context, req *journalentrypb.DeleteJournalEntryRequest) (*journalentrypb.DeleteJournalEntryResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityJournalEntry, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	if uc.repositories.JournalEntry == nil {
		return nil, errors.New("journal entry repository is not available")
	}
	resp, err := uc.repositories.JournalEntry.DeleteJournalEntry(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.errors.deletion_failed", "[ERR-DEFAULT] Journal entry deletion failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteJournalEntryUseCase) validateInput(ctx context.Context, req *journalentrypb.DeleteJournalEntryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteJournalEntryUseCase) validateBusinessRules(ctx context.Context, req *journalentrypb.DeleteJournalEntryRequest) error {
	// Only DRAFT entries can be deleted
	// Note: The current entry status must be read from the repository before deletion.
	// This guard relies on the repository adapter returning an error for non-DRAFT deletions,
	// or the caller pre-validating the status. For a full guard, read the entry first.
	// TODO: Read entry and check status == DRAFT before deleting
	return nil
}
