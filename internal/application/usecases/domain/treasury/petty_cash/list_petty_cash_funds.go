package pettycash

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pettycashfundpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_fund"
)

// ListPettyCashFundsRepositories groups all repository dependencies
type ListPettyCashFundsRepositories struct {
	PettyCashFund pettycashfundpb.PettyCashFundDomainServiceServer
}

// ListPettyCashFundsServices groups all business service dependencies
type ListPettyCashFundsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListPettyCashFundsUseCase handles the business logic for listing petty cash funds
type ListPettyCashFundsUseCase struct {
	repositories ListPettyCashFundsRepositories
	services     ListPettyCashFundsServices
}

// NewListPettyCashFundsUseCase creates a new ListPettyCashFundsUseCase
func NewListPettyCashFundsUseCase(
	repositories ListPettyCashFundsRepositories,
	services ListPettyCashFundsServices,
) *ListPettyCashFundsUseCase {
	return &ListPettyCashFundsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list petty cash funds operation
func (uc *ListPettyCashFundsUseCase) Execute(ctx context.Context, req *pettycashfundpb.ListPettyCashFundsRequest) (*pettycashfundpb.ListPettyCashFundsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPettyCashFund, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "petty_cash_fund.validation.request_required", "Request is required [DEFAULT]"))
	}

	if uc.repositories.PettyCashFund == nil {
		return nil, errors.New("petty cash fund repository is not available")
	}
	return uc.repositories.PettyCashFund.ListPettyCashFunds(ctx, req)
}
