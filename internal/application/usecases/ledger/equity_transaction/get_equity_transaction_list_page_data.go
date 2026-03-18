package equitytransaction

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	equitytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_transaction"
)

// GetEquityTransactionListPageDataRepositories groups all repository dependencies.
type GetEquityTransactionListPageDataRepositories struct {
	EquityTransaction equitytransactionpb.EquityTransactionDomainServiceServer
}

// GetEquityTransactionListPageDataServices groups all business service dependencies.
type GetEquityTransactionListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetEquityTransactionListPageDataUseCase handles the business logic for getting equity transaction list page data.
type GetEquityTransactionListPageDataUseCase struct {
	repositories GetEquityTransactionListPageDataRepositories
	services     GetEquityTransactionListPageDataServices
}

// NewGetEquityTransactionListPageDataUseCase creates the use case with grouped dependencies.
func NewGetEquityTransactionListPageDataUseCase(
	repositories GetEquityTransactionListPageDataRepositories,
	services GetEquityTransactionListPageDataServices,
) *GetEquityTransactionListPageDataUseCase {
	return &GetEquityTransactionListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get equity transaction list page data operation.
func (uc *GetEquityTransactionListPageDataUseCase) Execute(ctx context.Context, req *equitytransactionpb.GetEquityTransactionListPageDataRequest) (*equitytransactionpb.GetEquityTransactionListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityEquityTransaction, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "equity_transaction.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	if uc.repositories.EquityTransaction == nil {
		return nil, errors.New("equity_transaction repository is not available")
	}

	resp, err := uc.repositories.EquityTransaction.GetEquityTransactionListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "equity_transaction.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load equity transaction list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
