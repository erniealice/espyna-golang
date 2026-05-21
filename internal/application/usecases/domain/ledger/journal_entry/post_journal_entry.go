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

// PostJournalEntryRepositories groups all repository dependencies
type PostJournalEntryRepositories struct {
	JournalEntry journalentrypb.JournalEntryDomainServiceServer // Primary entity repository
}

// PostJournalEntryServices groups all business service dependencies
type PostJournalEntryServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// PostJournalEntryUseCase handles the business logic for posting journal entries.
// Posting validates that the entry is in DRAFT status, then delegates balance validation
// and status transition to the repository. Manual journal entries (source_type=MANUAL)
// require the journal:post_manual permission; all others require journal:post_guided.
type PostJournalEntryUseCase struct {
	repositories PostJournalEntryRepositories
	services     PostJournalEntryServices
}

// NewPostJournalEntryUseCase creates use case with grouped dependencies
func NewPostJournalEntryUseCase(
	repositories PostJournalEntryRepositories,
	services PostJournalEntryServices,
) *PostJournalEntryUseCase {
	return &PostJournalEntryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the post journal entry operation
func (uc *PostJournalEntryUseCase) Execute(ctx context.Context, req *journalentrypb.PostJournalEntryRequest) (*journalentrypb.PostJournalEntryResponse, error) {
	// Authorization check — posting is a lifecycle action beyond standard CRUD
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityJournalEntry, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "journal_entry.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Check if transaction service is available and supports transactions
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes posting within a transaction
func (uc *PostJournalEntryUseCase) executeWithTransaction(ctx context.Context, req *journalentrypb.PostJournalEntryRequest) (*journalentrypb.PostJournalEntryResponse, error) {
	var result *journalentrypb.PostJournalEntryResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "journal_entry.errors.post_failed", "Journal entry posting failed [DEFAULT]")
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

// executeCore contains the core business logic for posting.
// The repository is responsible for:
//   - Reading the entry and verifying status == DRAFT
//   - Validating that total_debit == total_credit (balanced entry)
//   - Setting status=POSTED and recording posted_by / posted_at audit fields
func (uc *PostJournalEntryUseCase) executeCore(ctx context.Context, req *journalentrypb.PostJournalEntryRequest) (*journalentrypb.PostJournalEntryResponse, error) {
	// Call repository
	if uc.repositories.JournalEntry == nil {
		return nil, errors.New("journal entry repository is not available")
	}
	resp, err := uc.repositories.JournalEntry.PostJournalEntry(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "journal_entry.errors.post_failed", "[ERR-DEFAULT] Journal entry posting failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *PostJournalEntryUseCase) validateInput(ctx context.Context, req *journalentrypb.PostJournalEntryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "journal_entry.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.JournalEntryId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "journal_entry.validation.id_required", "[ERR-DEFAULT] Journal entry ID is required"))
	}
	if req.PostedBy == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "journal_entry.validation.posted_by_required", "[ERR-DEFAULT] Posted by (user ID) is required"))
	}
	return nil
}
