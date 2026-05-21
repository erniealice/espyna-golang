package accruedexpense

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
)

const entityAccruedExpense = "accrued_expense"

// CreateAccruedExpenseRepositories groups repository dependencies.
type CreateAccruedExpenseRepositories struct {
	AccruedExpense accruedexpensepb.AccruedExpenseDomainServiceServer
}

// CreateAccruedExpenseServices groups service dependencies.
type CreateAccruedExpenseServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateAccruedExpenseUseCase handles creating a new accrued-expense row.
type CreateAccruedExpenseUseCase struct {
	repositories CreateAccruedExpenseRepositories
	services     CreateAccruedExpenseServices
}

// NewCreateAccruedExpenseUseCase creates a use case with grouped dependencies.
func NewCreateAccruedExpenseUseCase(
	repositories CreateAccruedExpenseRepositories,
	services CreateAccruedExpenseServices,
) *CreateAccruedExpenseUseCase {
	return &CreateAccruedExpenseUseCase{repositories: repositories, services: services}
}

// Execute performs the create operation.
func (uc *CreateAccruedExpenseUseCase) Execute(ctx context.Context, req *accruedexpensepb.CreateAccruedExpenseRequest) (*accruedexpensepb.CreateAccruedExpenseResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAccruedExpense, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *accruedexpensepb.CreateAccruedExpenseResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("accrued expense creation failed: %w", err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.executeCore(ctx, req)
}

func (uc *CreateAccruedExpenseUseCase) executeCore(ctx context.Context, req *accruedexpensepb.CreateAccruedExpenseRequest) (*accruedexpensepb.CreateAccruedExpenseResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"accrued_expense.validation.data_required", "Accrued expense data is required [DEFAULT]"))
	}

	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDService.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true

	if req.Data.Status == accruedexpensepb.AccruedExpenseStatus_ACCRUED_EXPENSE_STATUS_UNSPECIFIED {
		req.Data.Status = accruedexpensepb.AccruedExpenseStatus_ACCRUED_EXPENSE_STATUS_OUTSTANDING
	}
	// Initialise settled / remaining if not set.
	if req.Data.SettledAmount == 0 && req.Data.RemainingAmount == 0 {
		req.Data.RemainingAmount = req.Data.AccruedAmount
	}

	return uc.repositories.AccruedExpense.CreateAccruedExpense(ctx, req)
}
