package accruedexpense

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
)

// DeleteAccruedExpenseRepositories groups repository dependencies.
type DeleteAccruedExpenseRepositories struct {
	AccruedExpense accruedexpensepb.AccruedExpenseDomainServiceServer
}

// DeleteAccruedExpenseServices groups service dependencies.
type DeleteAccruedExpenseServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// DeleteAccruedExpenseUseCase handles deleting an accrued-expense.
type DeleteAccruedExpenseUseCase struct {
	repositories DeleteAccruedExpenseRepositories
	services     DeleteAccruedExpenseServices
}

// NewDeleteAccruedExpenseUseCase creates a use case with grouped dependencies.
func NewDeleteAccruedExpenseUseCase(
	repositories DeleteAccruedExpenseRepositories,
	services DeleteAccruedExpenseServices,
) *DeleteAccruedExpenseUseCase {
	return &DeleteAccruedExpenseUseCase{repositories: repositories, services: services}
}

// Execute performs the delete operation.
func (uc *DeleteAccruedExpenseUseCase) Execute(ctx context.Context, req *accruedexpensepb.DeleteAccruedExpenseRequest) (*accruedexpensepb.DeleteAccruedExpenseResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAccruedExpense, ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"accrued_expense.validation.id_required", "Accrued expense ID is required [DEFAULT]"))
	}
	return uc.repositories.AccruedExpense.DeleteAccruedExpense(ctx, req)
}
