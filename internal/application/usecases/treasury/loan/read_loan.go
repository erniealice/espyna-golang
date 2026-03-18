package loan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	loanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan"
)

// ReadLoanRepositories groups all repository dependencies.
type ReadLoanRepositories struct {
	Loan loanpb.LoanDomainServiceServer
}

// ReadLoanServices groups all business service dependencies.
type ReadLoanServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadLoanUseCase handles the business logic for reading a single loan.
type ReadLoanUseCase struct {
	repositories ReadLoanRepositories
	services     ReadLoanServices
}

// NewReadLoanUseCase creates the use case with grouped dependencies.
func NewReadLoanUseCase(
	repositories ReadLoanRepositories,
	services ReadLoanServices,
) *ReadLoanUseCase {
	return &ReadLoanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read loan operation.
func (uc *ReadLoanUseCase) Execute(ctx context.Context, req *loanpb.ReadLoanRequest) (*loanpb.ReadLoanResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityLoan, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "loan.validation.id_required", "[ERR-DEFAULT] Loan ID is required"))
	}

	if uc.repositories.Loan == nil {
		return nil, errors.New("loan repository is not available")
	}

	resp, err := uc.repositories.Loan.ReadLoan(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "loan.errors.read_failed", "[ERR-DEFAULT] Failed to read loan")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
