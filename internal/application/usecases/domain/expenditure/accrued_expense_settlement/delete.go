package accruedexpensesettlement

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
)

// DeleteAccruedExpenseSettlementRepositories groups repository dependencies.
type DeleteAccruedExpenseSettlementRepositories struct {
	AccruedExpenseSettlement accruedexpensepb.AccruedExpenseSettlementDomainServiceServer
}

// DeleteAccruedExpenseSettlementServices groups service dependencies.
type DeleteAccruedExpenseSettlementServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteAccruedExpenseSettlementUseCase handles deleting a settlement.
type DeleteAccruedExpenseSettlementUseCase struct {
	repositories DeleteAccruedExpenseSettlementRepositories
	services     DeleteAccruedExpenseSettlementServices
}

// NewDeleteAccruedExpenseSettlementUseCase creates a use case with grouped dependencies.
func NewDeleteAccruedExpenseSettlementUseCase(
	repositories DeleteAccruedExpenseSettlementRepositories,
	services DeleteAccruedExpenseSettlementServices,
) *DeleteAccruedExpenseSettlementUseCase {
	return &DeleteAccruedExpenseSettlementUseCase{repositories: repositories, services: services}
}

// Execute performs the delete operation.
func (uc *DeleteAccruedExpenseSettlementUseCase) Execute(ctx context.Context, req *accruedexpensepb.DeleteAccruedExpenseSettlementRequest) (*accruedexpensepb.DeleteAccruedExpenseSettlementResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityAccruedExpenseSettlement,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"accrued_expense_settlement.validation.id_required", "Settlement ID is required [DEFAULT]"))
	}
	return uc.repositories.AccruedExpenseSettlement.DeleteAccruedExpenseSettlement(ctx, req)
}
