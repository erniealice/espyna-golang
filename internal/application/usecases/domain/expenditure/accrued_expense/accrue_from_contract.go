package accruedexpense

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
)

// AccrueFromContractRepositories groups repository dependencies.
type AccrueFromContractRepositories struct {
	AccruedExpense accruedexpensepb.AccruedExpenseDomainServiceServer
}

// AccrueFromContractServices groups service dependencies.
type AccrueFromContractServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// AccrueFromContractUseCase emits an AccruedExpense(OUTSTANDING) row for a
// contract cycle when the workspace is on accrual-basis books.
//
// Idempotency: callers should pre-derive idempotency_key. The DB-side
// uq_accrued_expense_contract_cycle partial unique index (migration
// 20260430140300) enforces (supplier_contract_id, cycle_date) uniqueness.
type AccrueFromContractUseCase struct {
	repositories AccrueFromContractRepositories
	services     AccrueFromContractServices
}

// NewAccrueFromContractUseCase creates a use case with grouped dependencies.
func NewAccrueFromContractUseCase(
	repositories AccrueFromContractRepositories,
	services AccrueFromContractServices,
) *AccrueFromContractUseCase {
	return &AccrueFromContractUseCase{repositories: repositories, services: services}
}

// Execute performs the accrue-from-contract operation.
func (uc *AccrueFromContractUseCase) Execute(ctx context.Context, req *accruedexpensepb.AccrueFromContractRequest) (*accruedexpensepb.AccrueFromContractResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityAccruedExpense, entityid.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.GetSupplierContractId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"accrued_expense.validation.contract_id_required", "Supplier contract ID is required [DEFAULT]"))
	}
	if req.GetCycleDate() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"accrued_expense.validation.cycle_date_required", "Cycle date is required [DEFAULT]"))
	}

	now := time.Now()
	id := uc.services.IDGenerator.GenerateID()
	cycleDate := req.GetCycleDate()
	createReq := &accruedexpensepb.CreateAccruedExpenseRequest{
		Data: &accruedexpensepb.AccruedExpense{
			Id:                 id,
			DateCreated:        &[]int64{now.UnixMilli()}[0],
			DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
			DateModified:       &[]int64{now.UnixMilli()}[0],
			DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
			Active:             true,
			Status:             accruedexpensepb.AccruedExpenseStatus_ACCRUED_EXPENSE_STATUS_OUTSTANDING,
			SupplierContractId: req.GetSupplierContractId(),
			CycleDate:          &cycleDate,
			AccruedAmount:      req.GetAccruedAmount(),
			RemainingAmount:    req.GetAccruedAmount(),
		},
	}
	createResp, err := uc.repositories.AccruedExpense.CreateAccruedExpense(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to accrue from contract: %w", err)
	}
	var data *accruedexpensepb.AccruedExpense
	if len(createResp.Data) > 0 {
		data = createResp.Data[0]
	}
	return &accruedexpensepb.AccrueFromContractResponse{Success: true, Data: data}, nil
}
