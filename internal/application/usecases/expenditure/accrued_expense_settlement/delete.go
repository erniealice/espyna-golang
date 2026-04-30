package accruedexpensesettlement

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
)

// DeleteAccruedExpenseSettlementRepositories groups repository dependencies.
type DeleteAccruedExpenseSettlementRepositories struct {
	AccruedExpenseSettlement accruedexpensepb.AccruedExpenseSettlementDomainServiceServer
}

// DeleteAccruedExpenseSettlementServices groups service dependencies.
type DeleteAccruedExpenseSettlementServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAccruedExpenseSettlement, ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"accrued_expense_settlement.validation.id_required", "Settlement ID is required [DEFAULT]"))
	}
	return uc.repositories.AccruedExpenseSettlement.DeleteAccruedExpenseSettlement(ctx, req)
}
