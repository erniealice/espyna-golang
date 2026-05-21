package equitytransaction

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	equitytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_transaction"
)

// ListEquityTransactionsRepositories groups all repository dependencies.
type ListEquityTransactionsRepositories struct {
	EquityTransaction equitytransactionpb.EquityTransactionDomainServiceServer
}

// ListEquityTransactionsServices groups all business service dependencies.
type ListEquityTransactionsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListEquityTransactionsUseCase handles the business logic for listing equity transactions.
type ListEquityTransactionsUseCase struct {
	repositories ListEquityTransactionsRepositories
	services     ListEquityTransactionsServices
}

// NewListEquityTransactionsUseCase creates the use case with grouped dependencies.
func NewListEquityTransactionsUseCase(
	repositories ListEquityTransactionsRepositories,
	services ListEquityTransactionsServices,
) *ListEquityTransactionsUseCase {
	return &ListEquityTransactionsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list equity transactions operation.
func (uc *ListEquityTransactionsUseCase) Execute(ctx context.Context, req *equitytransactionpb.ListEquityTransactionsRequest) (*equitytransactionpb.ListEquityTransactionsResponse, error) {
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

	resp, err := uc.repositories.EquityTransaction.ListEquityTransactions(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "equity_transaction.errors.list_failed", "[ERR-DEFAULT] Failed to list equity transactions")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
