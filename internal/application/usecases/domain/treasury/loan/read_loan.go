package loan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	loanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan"
)

// ReadLoanRepositories groups all repository dependencies.
type ReadLoanRepositories struct {
	Loan loanpb.LoanDomainServiceServer
}

// ReadLoanServices groups all business service dependencies.
type ReadLoanServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityLoan, entityid.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan.validation.id_required", "[ERR-DEFAULT] Loan ID is required"))
	}

	if uc.repositories.Loan == nil {
		return nil, errors.New("loan repository is not available")
	}

	resp, err := uc.repositories.Loan.ReadLoan(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan.errors.read_failed", "[ERR-DEFAULT] Failed to read loan")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
