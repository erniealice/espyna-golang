package loan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	loanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan"
)

// ListLoansRepositories groups all repository dependencies.
type ListLoansRepositories struct {
	Loan loanpb.LoanDomainServiceServer
}

// ListLoansServices groups all business service dependencies.
type ListLoansServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListLoansUseCase handles the business logic for listing loans.
type ListLoansUseCase struct {
	repositories ListLoansRepositories
	services     ListLoansServices
}

// NewListLoansUseCase creates the use case with grouped dependencies.
func NewListLoansUseCase(
	repositories ListLoansRepositories,
	services ListLoansServices,
) *ListLoansUseCase {
	return &ListLoansUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list loans operation.
func (uc *ListLoansUseCase) Execute(ctx context.Context, req *loanpb.ListLoansRequest) (*loanpb.ListLoansResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityLoan,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	if uc.repositories.Loan == nil {
		return nil, errors.New("loan repository is not available")
	}

	resp, err := uc.repositories.Loan.ListLoans(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan.errors.list_failed", "[ERR-DEFAULT] Failed to list loans")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
