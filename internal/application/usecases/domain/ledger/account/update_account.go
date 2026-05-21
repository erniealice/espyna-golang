package account

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	accountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account"
)

// UpdateAccountRepositories groups all repository dependencies
type UpdateAccountRepositories struct {
	Account accountpb.AccountDomainServiceServer // Primary entity repository
}

// UpdateAccountServices groups all business service dependencies
type UpdateAccountServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateAccountUseCase handles the business logic for updating accounts
type UpdateAccountUseCase struct {
	repositories UpdateAccountRepositories
	services     UpdateAccountServices
}

// NewUpdateAccountUseCase creates use case with grouped dependencies
func NewUpdateAccountUseCase(
	repositories UpdateAccountRepositories,
	services UpdateAccountServices,
) *UpdateAccountUseCase {
	return &UpdateAccountUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update account operation
func (uc *UpdateAccountUseCase) Execute(ctx context.Context, req *accountpb.UpdateAccountRequest) (*accountpb.UpdateAccountResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAccount, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichAccountData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.errors.enrichment_failed", "[ERR-DEFAULT] Data enrichment failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	if uc.repositories.Account == nil {
		return nil, errors.New("account repository is not available")
	}
	resp, err := uc.repositories.Account.UpdateAccount(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.errors.update_failed", "[ERR-DEFAULT] Account update failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateAccountUseCase) validateInput(ctx context.Context, req *accountpb.UpdateAccountRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.validation.data_required", "[ERR-DEFAULT] Account data is required"))
	}

	// Trim leading and trailing spaces
	req.Data.Name = strings.TrimSpace(req.Data.Name)
	req.Data.Code = strings.TrimSpace(req.Data.Code)

	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.validation.id_required", "[ERR-DEFAULT] Account ID is required"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.validation.name_required", "[ERR-DEFAULT] Name is required"))
	}
	if req.Data.Code == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.validation.code_required", "[ERR-DEFAULT] Account code is required"))
	}
	return nil
}

// enrichAccountData adds audit information for updates
func (uc *UpdateAccountUseCase) enrichAccountData(account *accountpb.Account) error {
	now := time.Now()

	// Set audit fields for modification
	account.DateModified = &[]int64{now.UnixMilli()}[0]
	account.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateAccountUseCase) validateBusinessRules(ctx context.Context, account *accountpb.Account) error {
	// Validate name length
	if len(account.Name) > 200 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.validation.name_too_long", "[ERR-DEFAULT] Name must not exceed 200 characters"))
	}

	// Validate code length
	if len(account.Code) > 50 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.validation.code_too_long", "[ERR-DEFAULT] Account code must not exceed 50 characters"))
	}

	return nil
}
