package loan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	loanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan"
)

// ListLoansRepositories groups all repository dependencies.
type ListLoansRepositories struct {
	Loan loanpb.LoanDomainServiceServer
}

// ListLoansServices groups all business service dependencies.
type ListLoansServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityLoan, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "loan.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	if uc.repositories.Loan == nil {
		return nil, errors.New("loan repository is not available")
	}

	resp, err := uc.repositories.Loan.ListLoans(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "loan.errors.list_failed", "[ERR-DEFAULT] Failed to list loans")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
