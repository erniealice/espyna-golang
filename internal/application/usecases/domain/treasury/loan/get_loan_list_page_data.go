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

// GetLoanListPageDataRepositories groups all repository dependencies.
type GetLoanListPageDataRepositories struct {
	Loan loanpb.LoanDomainServiceServer
}

// GetLoanListPageDataServices groups all business service dependencies.
type GetLoanListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetLoanListPageDataUseCase handles the business logic for getting loan list page data.
type GetLoanListPageDataUseCase struct {
	repositories GetLoanListPageDataRepositories
	services     GetLoanListPageDataServices
}

// NewGetLoanListPageDataUseCase creates the use case with grouped dependencies.
func NewGetLoanListPageDataUseCase(
	repositories GetLoanListPageDataRepositories,
	services GetLoanListPageDataServices,
) *GetLoanListPageDataUseCase {
	return &GetLoanListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get loan list page data operation.
func (uc *GetLoanListPageDataUseCase) Execute(ctx context.Context, req *loanpb.GetLoanListPageDataRequest) (*loanpb.GetLoanListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityLoan, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	// Validate pagination parameters
	if req.Pagination != nil && req.Pagination.Limit > 0 && req.Pagination.Limit > 100 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
	}

	if uc.repositories.Loan == nil {
		return nil, errors.New("loan repository is not available")
	}

	resp, err := uc.repositories.Loan.GetLoanListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load loan list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
