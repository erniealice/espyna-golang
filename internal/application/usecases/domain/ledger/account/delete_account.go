package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	accountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account"
)

// DeleteAccountRepositories groups all repository dependencies
type DeleteAccountRepositories struct {
	Account accountpb.AccountDomainServiceServer // Primary entity repository
}

// DeleteAccountServices groups all business service dependencies
type DeleteAccountServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteAccountUseCase handles the business logic for deleting accounts
type DeleteAccountUseCase struct {
	repositories DeleteAccountRepositories
	services     DeleteAccountServices
}

// NewDeleteAccountUseCase creates use case with grouped dependencies
func NewDeleteAccountUseCase(
	repositories DeleteAccountRepositories,
	services DeleteAccountServices,
) *DeleteAccountUseCase {
	return &DeleteAccountUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete account operation
func (uc *DeleteAccountUseCase) Execute(ctx context.Context, req *accountpb.DeleteAccountRequest) (*accountpb.DeleteAccountResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAccount, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	if uc.repositories.Account == nil {
		return nil, errors.New("account repository is not available")
	}
	resp, err := uc.repositories.Account.DeleteAccount(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.errors.deletion_failed", "[ERR-DEFAULT] Account deletion failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteAccountUseCase) validateInput(ctx context.Context, req *accountpb.DeleteAccountRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteAccountUseCase) validateBusinessRules(ctx context.Context, req *accountpb.DeleteAccountRequest) error {
	// TODO: Add business rules for account deletion
	// Example: Check if account has journal entries, prevent deletion of system accounts
	// For now, allow all deletions
	return nil
}
