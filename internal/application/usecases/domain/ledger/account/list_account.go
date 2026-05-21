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

// ListAccountsRepositories groups all repository dependencies
type ListAccountsRepositories struct {
	Account accountpb.AccountDomainServiceServer // Primary entity repository
}

// ListAccountsServices groups all business service dependencies
type ListAccountsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListAccountsUseCase handles the business logic for listing accounts
type ListAccountsUseCase struct {
	repositories ListAccountsRepositories
	services     ListAccountsServices
}

// NewListAccountsUseCase creates use case with grouped dependencies
func NewListAccountsUseCase(
	repositories ListAccountsRepositories,
	services ListAccountsServices,
) *ListAccountsUseCase {
	return &ListAccountsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list accounts operation
func (uc *ListAccountsUseCase) Execute(ctx context.Context, req *accountpb.ListAccountsRequest) (*accountpb.ListAccountsResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityAccount, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "account.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "account.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	if uc.repositories.Account == nil {
		return nil, errors.New("account repository is not available")
	}
	resp, err := uc.repositories.Account.ListAccounts(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "account.errors.list_failed", "[ERR-DEFAULT] Failed to list accounts")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListAccountsUseCase) validateInput(ctx context.Context, req *accountpb.ListAccountsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "account.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for listing
func (uc *ListAccountsUseCase) validateBusinessRules(ctx context.Context, req *accountpb.ListAccountsRequest) error {
	// No additional business rules for listing accounts
	return nil
}
