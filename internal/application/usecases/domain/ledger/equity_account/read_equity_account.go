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

// ReadEquityAccountRepositories groups all repository dependencies.
type ReadEquityAccountRepositories struct {
	EquityAccount equityaccountpb.EquityAccountDomainServiceServer
}

// ReadEquityAccountServices groups all business service dependencies.
type ReadEquityAccountServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadEquityAccountUseCase handles the business logic for reading a single equity account.
type ReadEquityAccountUseCase struct {
	repositories ReadEquityAccountRepositories
	services     ReadEquityAccountServices
}

// NewReadEquityAccountUseCase creates the use case with grouped dependencies.
func NewReadEquityAccountUseCase(
	repositories ReadEquityAccountRepositories,
	services ReadEquityAccountServices,
) *ReadEquityAccountUseCase {
	return &ReadEquityAccountUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read equity account operation.
func (uc *ReadEquityAccountUseCase) Execute(ctx context.Context, req *equityaccountpb.ReadEquityAccountRequest) (*equityaccountpb.ReadEquityAccountResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityEquityAccount, entityid.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "equity_account.validation.id_required", "[ERR-DEFAULT] Equity account ID is required"))
	}

	if uc.repositories.EquityAccount == nil {
		return nil, errors.New("equity_account repository is not available")
	}

	resp, err := uc.repositories.EquityAccount.ReadEquityAccount(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "equity_account.errors.read_failed", "[ERR-DEFAULT] Failed to read equity account")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
