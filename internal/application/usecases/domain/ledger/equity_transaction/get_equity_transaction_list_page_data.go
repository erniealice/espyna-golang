package equitytransaction

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	equitytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_transaction"
)

// GetEquityTransactionListPageDataRepositories groups all repository dependencies.
type GetEquityTransactionListPageDataRepositories struct {
	EquityTransaction equitytransactionpb.EquityTransactionDomainServiceServer
}

// GetEquityTransactionListPageDataServices groups all business service dependencies.
type GetEquityTransactionListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityEquityTransaction,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "equity_transaction.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	if uc.repositories.EquityTransaction == nil {
		return nil, errors.New("equity_transaction repository is not available")
	}

	resp, err := uc.repositories.EquityTransaction.GetEquityTransactionListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "equity_transaction.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load equity transaction list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
