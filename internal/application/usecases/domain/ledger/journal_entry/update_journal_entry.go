package journalentry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	journalentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_entry"
)

// UpdateJournalEntryRepositories groups all repository dependencies
type UpdateJournalEntryRepositories struct {
	JournalEntry journalentrypb.JournalEntryDomainServiceServer // Primary entity repository
}

// UpdateJournalEntryServices groups all business service dependencies
type UpdateJournalEntryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateJournalEntryUseCase handles the business logic for updating journal entries
type UpdateJournalEntryUseCase struct {
	repositories UpdateJournalEntryRepositories
	services     UpdateJournalEntryServices
}

// NewUpdateJournalEntryUseCase creates use case with grouped dependencies
func NewUpdateJournalEntryUseCase(
	repositories UpdateJournalEntryRepositories,
	services UpdateJournalEntryServices,
) *UpdateJournalEntryUseCase {
	return &UpdateJournalEntryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update journal entry operation
func (uc *UpdateJournalEntryUseCase) Execute(ctx context.Context, req *journalentrypb.UpdateJournalEntryRequest) (*journalentrypb.UpdateJournalEntryResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityJournalEntry, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichJournalEntryData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.errors.enrichment_failed", "[ERR-DEFAULT] Data enrichment failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	if uc.repositories.JournalEntry == nil {
		return nil, errors.New("journal entry repository is not available")
	}
	resp, err := uc.repositories.JournalEntry.UpdateJournalEntry(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.errors.update_failed", "[ERR-DEFAULT] Journal entry update failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateJournalEntryUseCase) validateInput(ctx context.Context, req *journalentrypb.UpdateJournalEntryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.validation.data_required", "[ERR-DEFAULT] Journal entry data is required"))
	}

	// Trim leading and trailing spaces
	req.Data.Description = strings.TrimSpace(req.Data.Description)

	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.validation.id_required", "[ERR-DEFAULT] Journal entry ID is required"))
	}
	if req.Data.Description == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.validation.description_required", "[ERR-DEFAULT] Description is required"))
	}

	return nil
}

// enrichJournalEntryData adds audit information for updates
func (uc *UpdateJournalEntryUseCase) enrichJournalEntryData(entry *journalentrypb.JournalEntry) error {
	now := time.Now()

	// Set audit fields for modification
	entry.DateModified = &[]int64{now.UnixMilli()}[0]
	entry.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateJournalEntryUseCase) validateBusinessRules(ctx context.Context, entry *journalentrypb.JournalEntry) error {
	// Only DRAFT entries can be edited
	if entry.Status != journalentrypb.JournalEntryStatus_JOURNAL_ENTRY_STATUS_DRAFT &&
		entry.Status != journalentrypb.JournalEntryStatus_JOURNAL_ENTRY_STATUS_UNSPECIFIED {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.validation.only_draft_editable", "[ERR-DEFAULT] Only DRAFT journal entries can be edited"))
	}

	// Validate description length
	if len(entry.Description) > 500 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "journal_entry.validation.description_too_long", "[ERR-DEFAULT] Description must not exceed 500 characters"))
	}

	return nil
}
