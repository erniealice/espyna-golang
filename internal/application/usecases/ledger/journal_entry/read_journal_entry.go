package journalentry

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	journalentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_entry"
)

// ReadJournalEntryRepositories groups all repository dependencies
type ReadJournalEntryRepositories struct {
	JournalEntry journalentrypb.JournalEntryDomainServiceServer // Primary entity repository
}

// ReadJournalEntryServices groups all business service dependencies
type ReadJournalEntryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadJournalEntryUseCase handles the business logic for reading journal entries
type ReadJournalEntryUseCase struct {
	repositories ReadJournalEntryRepositories
	services     ReadJournalEntryServices
}

// NewReadJournalEntryUseCase creates use case with grouped dependencies
func NewReadJournalEntryUseCase(
	repositories ReadJournalEntryRepositories,
	services ReadJournalEntryServices,
) *ReadJournalEntryUseCase {
	return &ReadJournalEntryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read journal entry operation
func (uc *ReadJournalEntryUseCase) Execute(ctx context.Context, req *journalentrypb.ReadJournalEntryRequest) (*journalentrypb.ReadJournalEntryResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityJournalEntry, ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	if uc.repositories.JournalEntry == nil {
		return nil, errors.New("journal entry repository is not available")
	}
	resp, err := uc.repositories.JournalEntry.ReadJournalEntry(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return response as-is (even if empty data for not found case)
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadJournalEntryUseCase) validateInput(ctx context.Context, req *journalentrypb.ReadJournalEntryRequest) error {
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
