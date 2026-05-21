package accruedexpense

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
)

// ReadAccruedExpenseRepositories groups repository dependencies.
type ReadAccruedExpenseRepositories struct {
	AccruedExpense accruedexpensepb.AccruedExpenseDomainServiceServer
}

// ReadAccruedExpenseServices groups service dependencies.
type ReadAccruedExpenseServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// ReadAccruedExpenseUseCase handles reading an accrued-expense.
type ReadAccruedExpenseUseCase struct {
	repositories ReadAccruedExpenseRepositories
	services     ReadAccruedExpenseServices
}

// NewReadAccruedExpenseUseCase creates a use case with grouped dependencies.
func NewReadAccruedExpenseUseCase(
	repositories ReadAccruedExpenseRepositories,
	services ReadAccruedExpenseServices,
) *ReadAccruedExpenseUseCase {
	return &ReadAccruedExpenseUseCase{repositories: repositories, services: services}
}

// Execute performs the read operation.
func (uc *ReadAccruedExpenseUseCase) Execute(ctx context.Context, req *accruedexpensepb.ReadAccruedExpenseRequest) (*accruedexpensepb.ReadAccruedExpenseResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityAccruedExpense, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"accrued_expense.validation.id_required", "Accrued expense ID is required [DEFAULT]"))
	}
	return uc.repositories.AccruedExpense.ReadAccruedExpense(ctx, req)
}
