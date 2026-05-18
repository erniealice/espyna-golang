package accruedexpensesettlement

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
)

const entityAccruedExpenseSettlement = "accrued_expense_settlement"

// CreateAccruedExpenseSettlementRepositories groups repository dependencies.
type CreateAccruedExpenseSettlementRepositories struct {
	AccruedExpenseSettlement accruedexpensepb.AccruedExpenseSettlementDomainServiceServer
}

// CreateAccruedExpenseSettlementServices groups service dependencies.
type CreateAccruedExpenseSettlementServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateAccruedExpenseSettlementUseCase handles creating a settlement row.
//
// HIGH-3 single-write boundary: settlement rows are the canonical owner of
// `AccruedExpense.settled_amount` and `AccruedExpense.remaining_amount`. The
// Opus settle_accrual.go orchestrates the cross-row total update; this use case
// just persists the settlement row.
type CreateAccruedExpenseSettlementUseCase struct {
	repositories CreateAccruedExpenseSettlementRepositories
	services     CreateAccruedExpenseSettlementServices
}

// NewCreateAccruedExpenseSettlementUseCase creates a use case with grouped dependencies.
func NewCreateAccruedExpenseSettlementUseCase(
	repositories CreateAccruedExpenseSettlementRepositories,
	services CreateAccruedExpenseSettlementServices,
) *CreateAccruedExpenseSettlementUseCase {
	return &CreateAccruedExpenseSettlementUseCase{repositories: repositories, services: services}
}

// Execute performs the create operation.
func (uc *CreateAccruedExpenseSettlementUseCase) Execute(ctx context.Context, req *accruedexpensepb.CreateAccruedExpenseSettlementRequest) (*accruedexpensepb.CreateAccruedExpenseSettlementResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAccruedExpenseSettlement, ports.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"accrued_expense_settlement.validation.data_required", "Settlement data is required [DEFAULT]"))
	}
	if req.Data.AccruedExpenseId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"accrued_expense_settlement.validation.accrued_expense_id_required", "Accrued expense ID is required [DEFAULT]"))
	}
	if req.Data.ExpenditureId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"accrued_expense_settlement.validation.expenditure_id_required", "Expenditure ID is required [DEFAULT]"))
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
	return uc.repositories.AccruedExpenseSettlement.CreateAccruedExpenseSettlement(ctx, req)
}
