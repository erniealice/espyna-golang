package accruedexpense

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
)

// UpdateAccruedExpenseRepositories groups repository dependencies.
type UpdateAccruedExpenseRepositories struct {
	AccruedExpense accruedexpensepb.AccruedExpenseDomainServiceServer
}

// UpdateAccruedExpenseServices groups service dependencies.
type UpdateAccruedExpenseServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// UpdateAccruedExpenseUseCase handles updating an accrued-expense.
//
// Single-write boundary discipline (plan §10 R2/R3): callers must NOT update
// settled_amount or remaining_amount through this use case. Those fields are
// owned exclusively by the AccruedExpenseSettlement service / settle_accrual.go
// (Opus scope).
type UpdateAccruedExpenseUseCase struct {
	repositories UpdateAccruedExpenseRepositories
	services     UpdateAccruedExpenseServices
}

// NewUpdateAccruedExpenseUseCase creates a use case with grouped dependencies.
func NewUpdateAccruedExpenseUseCase(
	repositories UpdateAccruedExpenseRepositories,
	services UpdateAccruedExpenseServices,
) *UpdateAccruedExpenseUseCase {
	return &UpdateAccruedExpenseUseCase{repositories: repositories, services: services}
}

// Execute performs the update operation.
func (uc *UpdateAccruedExpenseUseCase) Execute(ctx context.Context, req *accruedexpensepb.UpdateAccruedExpenseRequest) (*accruedexpensepb.UpdateAccruedExpenseResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityAccruedExpense, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"accrued_expense.validation.id_required", "Accrued expense ID is required [DEFAULT]"))
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.AccruedExpense.UpdateAccruedExpense(ctx, req)
}
