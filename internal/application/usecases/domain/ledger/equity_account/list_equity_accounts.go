package equityaccount

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	equityaccountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_account"
)

// ListEquityAccountsRepositories groups all repository dependencies.
type ListEquityAccountsRepositories struct {
	EquityAccount equityaccountpb.EquityAccountDomainServiceServer
}

// ListEquityAccountsServices groups all business service dependencies.
type ListEquityAccountsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListEquityAccountsUseCase handles the business logic for listing equity accounts.
type ListEquityAccountsUseCase struct {
	repositories ListEquityAccountsRepositories
	services     ListEquityAccountsServices
}

// NewListEquityAccountsUseCase creates the use case with grouped dependencies.
func NewListEquityAccountsUseCase(
	repositories ListEquityAccountsRepositories,
	services ListEquityAccountsServices,
) *ListEquityAccountsUseCase {
	return &ListEquityAccountsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list equity accounts operation.
func (uc *ListEquityAccountsUseCase) Execute(ctx context.Context, req *equityaccountpb.ListEquityAccountsRequest) (*equityaccountpb.ListEquityAccountsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityEquityAccount, entityid.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "equity_account.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	if uc.repositories.EquityAccount == nil {
		return nil, errors.New("equity_account repository is not available")
	}

	resp, err := uc.repositories.EquityAccount.ListEquityAccounts(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "equity_account.errors.list_failed", "[ERR-DEFAULT] Failed to list equity accounts")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
