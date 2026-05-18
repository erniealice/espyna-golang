package journalentry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	journalentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_entry"
)

const entityJournalEntry = "journal_entry"

// CreateJournalEntryRepositories groups all repository dependencies
type CreateJournalEntryRepositories struct {
	JournalEntry journalentrypb.JournalEntryDomainServiceServer // Primary entity repository
}

// CreateJournalEntryServices groups all business service dependencies
type CreateJournalEntryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateJournalEntryUseCase handles the business logic for creating journal entries
type CreateJournalEntryUseCase struct {
	repositories CreateJournalEntryRepositories
	services     CreateJournalEntryServices
}

// NewCreateJournalEntryUseCase creates use case with grouped dependencies
func NewCreateJournalEntryUseCase(
	repositories CreateJournalEntryRepositories,
	services CreateJournalEntryServices,
) *CreateJournalEntryUseCase {
	return &CreateJournalEntryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create journal entry operation
func (uc *CreateJournalEntryUseCase) Execute(ctx context.Context, req *journalentrypb.CreateJournalEntryRequest) (*journalentrypb.CreateJournalEntryResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityJournalEntry, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes journal entry creation within a transaction
func (uc *CreateJournalEntryUseCase) executeWithTransaction(ctx context.Context, req *journalentrypb.CreateJournalEntryRequest) (*journalentrypb.CreateJournalEntryResponse, error) {
	var result *journalentrypb.CreateJournalEntryResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "journal_entry.errors.creation_failed", "Journal entry creation failed [DEFAULT]")
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

// executeCore contains the core business logic
func (uc *CreateJournalEntryUseCase) executeCore(ctx context.Context, req *journalentrypb.CreateJournalEntryRequest) (*journalentrypb.CreateJournalEntryResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichJournalEntryData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	if uc.repositories.JournalEntry == nil {
		return nil, errors.New("journal entry repository is not available")
	}
	return uc.repositories.JournalEntry.CreateJournalEntry(ctx, req)
}

// validateInput validates the input request
func (uc *CreateJournalEntryUseCase) validateInput(ctx context.Context, req *journalentrypb.CreateJournalEntryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.validation.data_required", "[ERR-DEFAULT] Journal entry data is required"))
	}

	// Trim leading and trailing spaces
	req.Data.Description = strings.TrimSpace(req.Data.Description)

	if req.Data.Description == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.validation.description_required", "[ERR-DEFAULT] Description is required"))
	}

	return nil
}

// enrichJournalEntryData adds generated fields and audit information
func (uc *CreateJournalEntryUseCase) enrichJournalEntryData(entry *journalentrypb.JournalEntry) error {
	now := time.Now()

	// Generate Journal Entry ID if not provided
	if entry.Id == "" {
		entry.Id = uc.services.IDService.GenerateID()
	}

	// Generate entry_number if not provided.
	// Format: JE-YYYYMMDD-{6-char suffix from ID}.
	// Uses a timestamp+ID-suffix approach to guarantee uniqueness without a DB sequence.
	if entry.EntryNumber == "" {
		datePrefix := now.UTC().Format("20060102")
		idSuffix := entry.Id
		if len(idSuffix) > 6 {
			idSuffix = idSuffix[len(idSuffix)-6:]
		}
		entry.EntryNumber = fmt.Sprintf("JE-%s-%s", datePrefix, strings.ToUpper(idSuffix))
	}

	// Set source_type to MANUAL if not set (required NOT NULL column)
	if entry.SourceType == journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_UNSPECIFIED {
		entry.SourceType = journalentrypb.JournalSourceType_JOURNAL_SOURCE_TYPE_MANUAL
	}

	// Set status to DRAFT if not set
	if entry.Status == journalentrypb.JournalEntryStatus_JOURNAL_ENTRY_STATUS_UNSPECIFIED {
		entry.Status = journalentrypb.JournalEntryStatus_JOURNAL_ENTRY_STATUS_DRAFT
	}

	// Set entry date — prefer EntryDateString (UI date picker "YYYY-MM-DD"), fallback to now.
	if entry.EntryDate == 0 {
		if entry.EntryDateString != nil && *entry.EntryDateString != "" {
			// Parse UI date string (YYYY-MM-DD or RFC3339); use UTC midnight for the date.
			parsed := false
			for _, layout := range []string{"2006-01-02", time.RFC3339} {
				if t, err := time.Parse(layout, *entry.EntryDateString); err == nil {
					entry.EntryDate = t.UnixMilli()
					parsed = true
					break
				}
			}
			if !parsed {
				// Fallback: use current time if string is unparseable
				entry.EntryDate = now.UnixMilli()
				s := now.Format("2006-01-02")
				entry.EntryDateString = &s
			}
		} else {
			entry.EntryDate = now.UnixMilli()
			s := now.Format("2006-01-02")
			entry.EntryDateString = &s
		}
	}

	// Set audit fields
	entry.DateCreated = &[]int64{now.UnixMilli()}[0]
	entry.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	entry.DateModified = &[]int64{now.UnixMilli()}[0]
	entry.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	entry.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateJournalEntryUseCase) validateBusinessRules(ctx context.Context, entry *journalentrypb.JournalEntry) error {
	// Validate description length
	if len(entry.Description) > 500 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.validation.description_too_long", "[ERR-DEFAULT] Description must not exceed 500 characters"))
	}

	return nil
}
