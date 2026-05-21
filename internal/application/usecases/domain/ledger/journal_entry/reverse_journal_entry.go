package journalentry

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	journalentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_entry"
)

// ReverseJournalEntryRepositories groups all repository dependencies
type ReverseJournalEntryRepositories struct {
	JournalEntry journalentrypb.JournalEntryDomainServiceServer // Primary entity repository
}

// ReverseJournalEntryServices groups all business service dependencies
type ReverseJournalEntryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// ReverseJournalEntryUseCase handles the business logic for reversing posted journal entries.
// The repository is responsible for:
//   - Reading the original POSTED entry and verifying it has not already been reversed
//   - Creating the offsetting reversal entry (new entry with swapped debit/credit lines)
//   - Setting reversal_entry_id on the original entry
//   - Setting status=REVERSED on the original entry
//   - Returning the new reversal entry in the response
type ReverseJournalEntryUseCase struct {
	repositories ReverseJournalEntryRepositories
	services     ReverseJournalEntryServices
}

// NewReverseJournalEntryUseCase creates use case with grouped dependencies
func NewReverseJournalEntryUseCase(
	repositories ReverseJournalEntryRepositories,
	services ReverseJournalEntryServices,
) *ReverseJournalEntryUseCase {
	return &ReverseJournalEntryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the reverse journal entry operation
func (uc *ReverseJournalEntryUseCase) Execute(ctx context.Context, req *journalentrypb.ReverseJournalEntryRequest) (*journalentrypb.ReverseJournalEntryResponse, error) {
	// Authorization check — reversing requires update-level access
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityJournalEntry, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes reversal within a transaction
func (uc *ReverseJournalEntryUseCase) executeWithTransaction(ctx context.Context, req *journalentrypb.ReverseJournalEntryRequest) (*journalentrypb.ReverseJournalEntryResponse, error) {
	var result *journalentrypb.ReverseJournalEntryResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "journal_entry.errors.reversal_failed", "Journal entry reversal failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for reversal
func (uc *ReverseJournalEntryUseCase) executeCore(ctx context.Context, req *journalentrypb.ReverseJournalEntryRequest) (*journalentrypb.ReverseJournalEntryResponse, error) {
	// Call repository
	if uc.repositories.JournalEntry == nil {
		return nil, errors.New("journal entry repository is not available")
	}
	resp, err := uc.repositories.JournalEntry.ReverseJournalEntry(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.errors.reversal_failed", "[ERR-DEFAULT] Journal entry reversal failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReverseJournalEntryUseCase) validateInput(ctx context.Context, req *journalentrypb.ReverseJournalEntryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.JournalEntryId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.validation.id_required", "[ERR-DEFAULT] Journal entry ID is required"))
	}
	if req.ReversedBy == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.validation.reversed_by_required", "[ERR-DEFAULT] Reversed by (user ID) is required"))
	}
	return nil
}
