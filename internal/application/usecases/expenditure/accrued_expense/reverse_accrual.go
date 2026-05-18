package accruedexpense

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
)

// ReverseAccrualRepositories groups repository dependencies.
type ReverseAccrualRepositories struct {
	AccruedExpense accruedexpensepb.AccruedExpenseDomainServiceServer
}

// ReverseAccrualServices groups service dependencies.
type ReverseAccrualServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReverseAccrualUseCase flips an accrual to status=REVERSED. Guards against
// reversing an accrual that has un-reversed settlements (the AP team must
// reverse those settlements first via the AccruedExpenseSettlement service).
type ReverseAccrualUseCase struct {
	repositories ReverseAccrualRepositories
	services     ReverseAccrualServices
}

// NewReverseAccrualUseCase creates a use case with grouped dependencies.
func NewReverseAccrualUseCase(
	repositories ReverseAccrualRepositories,
	services ReverseAccrualServices,
) *ReverseAccrualUseCase {
	return &ReverseAccrualUseCase{repositories: repositories, services: services}
}

// Execute performs the reverse-accrual operation.
func (uc *ReverseAccrualUseCase) Execute(ctx context.Context, req *accruedexpensepb.ReverseAccrualRequest) (*accruedexpensepb.ReverseAccrualResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAccruedExpense, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.GetAccruedExpenseId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"accrued_expense.validation.id_required", "Accrued expense ID is required [DEFAULT]"))
	}

	// Guard: cannot reverse an accrual that has settled into a non-zero amount,
	// unless those settlements have been reversed first.
	readResp, err := uc.repositories.AccruedExpense.ReadAccruedExpense(ctx, &accruedexpensepb.ReadAccruedExpenseRequest{
		Data: &accruedexpensepb.AccruedExpense{Id: req.GetAccruedExpenseId()},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read accrual: %w", err)
	}
	if len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"accrued_expense.errors.not_found", "[ERR-DEFAULT] Accrued expense not found"))
	}
	original := readResp.Data[0]
	if original.GetSettledAmount() != 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"accrued_expense.errors.cannot_reverse_settled", "[ERR-DEFAULT] Cannot reverse an accrual with un-reversed settlements"))
	}

	// Flip status to REVERSED via update.
	now := time.Now()
	original.Status = accruedexpensepb.AccruedExpenseStatus_ACCRUED_EXPENSE_STATUS_REVERSED
	original.DateModified = &[]int64{now.UnixMilli()}[0]
	original.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	if _, err := uc.repositories.AccruedExpense.UpdateAccruedExpense(ctx, &accruedexpensepb.UpdateAccruedExpenseRequest{Data: original}); err != nil {
		return nil, fmt.Errorf("failed to mark accrual reversed: %w", err)
	}
	return &accruedexpensepb.ReverseAccrualResponse{Success: true, Data: original}, nil
}
