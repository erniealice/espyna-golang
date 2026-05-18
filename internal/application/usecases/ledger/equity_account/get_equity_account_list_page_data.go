package equityaccount

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	equityaccountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_account"
)

// GetEquityAccountListPageDataRepositories groups all repository dependencies.
type GetEquityAccountListPageDataRepositories struct {
	EquityAccount equityaccountpb.EquityAccountDomainServiceServer
}

// GetEquityAccountListPageDataServices groups all business service dependencies.
type GetEquityAccountListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetEquityAccountListPageDataUseCase handles the business logic for getting equity account list page data.
type GetEquityAccountListPageDataUseCase struct {
	repositories GetEquityAccountListPageDataRepositories
	services     GetEquityAccountListPageDataServices
}

// NewGetEquityAccountListPageDataUseCase creates the use case with grouped dependencies.
func NewGetEquityAccountListPageDataUseCase(
	repositories GetEquityAccountListPageDataRepositories,
	services GetEquityAccountListPageDataServices,
) *GetEquityAccountListPageDataUseCase {
	return &GetEquityAccountListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get equity account list page data operation.
func (uc *GetEquityAccountListPageDataUseCase) Execute(ctx context.Context, req *equityaccountpb.GetEquityAccountListPageDataRequest) (*equityaccountpb.GetEquityAccountListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityEquityAccount, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "equity_account.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	if uc.repositories.EquityAccount == nil {
		return nil, errors.New("equity_account repository is not available")
	}

	resp, err := uc.repositories.EquityAccount.GetEquityAccountListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "equity_account.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load equity account list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
