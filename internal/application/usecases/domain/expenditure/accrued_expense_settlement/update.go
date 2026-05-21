package accruedexpensesettlement

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
)

// UpdateAccruedExpenseSettlementRepositories groups repository dependencies.
type UpdateAccruedExpenseSettlementRepositories struct {
	AccruedExpenseSettlement accruedexpensepb.AccruedExpenseSettlementDomainServiceServer
}

// UpdateAccruedExpenseSettlementServices groups service dependencies.
type UpdateAccruedExpenseSettlementServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// UpdateAccruedExpenseSettlementUseCase handles updating a settlement.
type UpdateAccruedExpenseSettlementUseCase struct {
	repositories UpdateAccruedExpenseSettlementRepositories
	services     UpdateAccruedExpenseSettlementServices
}

// NewUpdateAccruedExpenseSettlementUseCase creates a use case with grouped dependencies.
func NewUpdateAccruedExpenseSettlementUseCase(
	repositories UpdateAccruedExpenseSettlementRepositories,
	services UpdateAccruedExpenseSettlementServices,
) *UpdateAccruedExpenseSettlementUseCase {
	return &UpdateAccruedExpenseSettlementUseCase{repositories: repositories, services: services}
}

// Execute performs the update operation.
func (uc *UpdateAccruedExpenseSettlementUseCase) Execute(ctx context.Context, req *accruedexpensepb.UpdateAccruedExpenseSettlementRequest) (*accruedexpensepb.UpdateAccruedExpenseSettlementResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAccruedExpenseSettlement, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"accrued_expense_settlement.validation.id_required", "Settlement ID is required [DEFAULT]"))
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.repositories.AccruedExpenseSettlement.UpdateAccruedExpenseSettlement(ctx, req)
}
