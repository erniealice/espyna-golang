package account

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	accountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account"
)

const entityAccount = "account"

// CreateAccountRepositories groups all repository dependencies
type CreateAccountRepositories struct {
	Account accountpb.AccountDomainServiceServer // Primary entity repository
}

// CreateAccountServices groups all business service dependencies
type CreateAccountServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateAccountUseCase handles the business logic for creating accounts
type CreateAccountUseCase struct {
	repositories CreateAccountRepositories
	services     CreateAccountServices
}

// NewCreateAccountUseCase creates use case with grouped dependencies
func NewCreateAccountUseCase(
	repositories CreateAccountRepositories,
	services CreateAccountServices,
) *CreateAccountUseCase {
	return &CreateAccountUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create account operation
func (uc *CreateAccountUseCase) Execute(ctx context.Context, req *accountpb.CreateAccountRequest) (*accountpb.CreateAccountResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAccount, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes account creation within a transaction
func (uc *CreateAccountUseCase) executeWithTransaction(ctx context.Context, req *accountpb.CreateAccountRequest) (*accountpb.CreateAccountResponse, error) {
	var result *accountpb.CreateAccountResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "account.errors.creation_failed", "Account creation failed [DEFAULT]")
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
func (uc *CreateAccountUseCase) executeCore(ctx context.Context, req *accountpb.CreateAccountRequest) (*accountpb.CreateAccountResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichAccountData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	if uc.repositories.Account == nil {
		return nil, errors.New("account repository is not available")
	}
	return uc.repositories.Account.CreateAccount(ctx, req)
}

// validateInput validates the input request
func (uc *CreateAccountUseCase) validateInput(ctx context.Context, req *accountpb.CreateAccountRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.validation.data_required", "[ERR-DEFAULT] Account data is required"))
	}

	// Trim leading and trailing spaces
	req.Data.Name = strings.TrimSpace(req.Data.Name)
	req.Data.Code = strings.TrimSpace(req.Data.Code)

	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.validation.name_required", "[ERR-DEFAULT] Name is required"))
	}
	if req.Data.Code == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "account.validation.code_required", "[ERR-DEFAULT] Account code is required"))
	}
	return nil
}

// enrichAccountData adds generated fields and audit information
func (uc *CreateAccountUseCase) enrichAccountData(account *accountpb.Account) error {
	now := time.Now()

	// Generate Account ID if not provided
	if account.Id == "" {
		account.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	account.DateCreated = &[]int64{now.UnixMilli()}[0]
	account.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	account.DateModified = &[]int64{now.UnixMilli()}[0]
	account.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	account.Active = true

	// Default status to ACTIVE if not set (ACCOUNT_STATUS_UNSPECIFIED = 0 means not set)
	if account.Status == accountpb.AccountStatus_ACCOUNT_STATUS_UNSPECIFIED {
		account.Status = accountpb.AccountStatus_ACCOUNT_STATUS_ACTIVE
	}

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateAccountUseCase) validateBusinessRules(ctx context.Context, account *accountpb.Account) error {
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
