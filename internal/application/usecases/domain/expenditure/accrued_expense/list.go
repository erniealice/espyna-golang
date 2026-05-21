package accruedexpense

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
)

// ListAccruedExpensesRepositories groups repository dependencies.
type ListAccruedExpensesRepositories struct {
	AccruedExpense accruedexpensepb.AccruedExpenseDomainServiceServer
}

// ListAccruedExpensesServices groups service dependencies.
type ListAccruedExpensesServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ListAccruedExpensesUseCase handles listing accrued expenses.
type ListAccruedExpensesUseCase struct {
	repositories ListAccruedExpensesRepositories
	services     ListAccruedExpensesServices
}

// NewListAccruedExpensesUseCase creates a use case with grouped dependencies.
func NewListAccruedExpensesUseCase(
	repositories ListAccruedExpensesRepositories,
	services ListAccruedExpensesServices,
) *ListAccruedExpensesUseCase {
	return &ListAccruedExpensesUseCase{repositories: repositories, services: services}
}

// Execute performs the list operation.
func (uc *ListAccruedExpensesUseCase) Execute(ctx context.Context, req *accruedexpensepb.ListAccruedExpensesRequest) (*accruedexpensepb.ListAccruedExpensesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAccruedExpense, ports.ActionList); err != nil {
		return nil, err
	}
	return uc.repositories.AccruedExpense.ListAccruedExpenses(ctx, req)
}
