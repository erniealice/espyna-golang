package accruedexpensesettlement

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
)

// ReadAccruedExpenseSettlementRepositories groups repository dependencies.
type ReadAccruedExpenseSettlementRepositories struct {
	AccruedExpenseSettlement accruedexpensepb.AccruedExpenseSettlementDomainServiceServer
}

// ReadAccruedExpenseSettlementServices groups service dependencies.
type ReadAccruedExpenseSettlementServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ReadAccruedExpenseSettlementUseCase handles reading a settlement.
type ReadAccruedExpenseSettlementUseCase struct {
	repositories ReadAccruedExpenseSettlementRepositories
	services     ReadAccruedExpenseSettlementServices
}

// NewReadAccruedExpenseSettlementUseCase creates a use case with grouped dependencies.
func NewReadAccruedExpenseSettlementUseCase(
	repositories ReadAccruedExpenseSettlementRepositories,
	services ReadAccruedExpenseSettlementServices,
) *ReadAccruedExpenseSettlementUseCase {
	return &ReadAccruedExpenseSettlementUseCase{repositories: repositories, services: services}
}

// Execute performs the read operation.
func (uc *ReadAccruedExpenseSettlementUseCase) Execute(ctx context.Context, req *accruedexpensepb.ReadAccruedExpenseSettlementRequest) (*accruedexpensepb.ReadAccruedExpenseSettlementResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAccruedExpenseSettlement, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"accrued_expense_settlement.validation.id_required", "Settlement ID is required [DEFAULT]"))
	}
	return uc.repositories.AccruedExpenseSettlement.ReadAccruedExpenseSettlement(ctx, req)
}
