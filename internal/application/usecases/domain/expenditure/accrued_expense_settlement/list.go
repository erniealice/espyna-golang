package accruedexpensesettlement

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
)

// ListAccruedExpenseSettlementsRepositories groups repository dependencies.
type ListAccruedExpenseSettlementsRepositories struct {
	AccruedExpenseSettlement accruedexpensepb.AccruedExpenseSettlementDomainServiceServer
}

// ListAccruedExpenseSettlementsServices groups service dependencies.
type ListAccruedExpenseSettlementsServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// ListAccruedExpenseSettlementsUseCase handles listing settlements.
type ListAccruedExpenseSettlementsUseCase struct {
	repositories ListAccruedExpenseSettlementsRepositories
	services     ListAccruedExpenseSettlementsServices
}

// NewListAccruedExpenseSettlementsUseCase creates a use case with grouped dependencies.
func NewListAccruedExpenseSettlementsUseCase(
	repositories ListAccruedExpenseSettlementsRepositories,
	services ListAccruedExpenseSettlementsServices,
) *ListAccruedExpenseSettlementsUseCase {
	return &ListAccruedExpenseSettlementsUseCase{repositories: repositories, services: services}
}

// Execute performs the list operation.
func (uc *ListAccruedExpenseSettlementsUseCase) Execute(ctx context.Context, req *accruedexpensepb.ListAccruedExpenseSettlementsRequest) (*accruedexpensepb.ListAccruedExpenseSettlementsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityAccruedExpenseSettlement, ports.ActionList); err != nil {
		return nil, err
	}
	return uc.repositories.AccruedExpenseSettlement.ListAccruedExpenseSettlements(ctx, req)
}
