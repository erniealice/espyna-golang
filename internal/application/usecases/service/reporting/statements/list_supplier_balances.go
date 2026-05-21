package statements

import (
	"context"
	"errors"

	stmtspb "github.com/erniealice/esqyma/pkg/schema/v1/service/reporting/statements"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// ListSupplierBalancesUseCase is the proto-shaped wrapper for the
// per-supplier outstanding-balance lookup. The legacy adapter returns
// `map[string]int64` (supplier_id → centavo balance); per Q-SDM-MAP-SHAPES
// the new contract emits `repeated BalanceRow` (typed proto rows).
//
// **Sort order:** rows are sorted by `counterparty_id` ascending. Shares
// the helper `balanceMapToRows` defined in `list_client_balances.go`.
type ListSupplierBalancesUseCase struct {
	reporter             reporter
	authorizationService ports.Authorizer
	translationService   ports.Translator
}

// NewListSupplierBalancesUseCase wires the use case with nil-safe deps.
func NewListSupplierBalancesUseCase(
	r reporter,
	authSvc ports.Authorizer,
	i18nSvc ports.Translator,
) *ListSupplierBalancesUseCase {
	if i18nSvc == nil {
		i18nSvc = ports.NewNoOpTranslator()
	}
	return &ListSupplierBalancesUseCase{
		reporter:             r,
		authorizationService: authSvc,
		translationService:   i18nSvc,
	}
}

// Execute runs the per-supplier balance lookup under the "reports" +
// ActionList authcheck.
func (uc *ListSupplierBalancesUseCase) Execute(
	ctx context.Context,
	req *stmtspb.ListSupplierBalancesRequest,
) (*stmtspb.ListSupplierBalancesResponse, error) {
	if err := authcheck.Check(
		ctx,
		uc.authorizationService,
		uc.translationService,
		"reports",
		ports.ActionList,
	); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.translationService,
			"reports.validation.request_required", "List supplier balances request is required [DEFAULT]"))
	}
	if uc.reporter == nil {
		return &stmtspb.ListSupplierBalancesResponse{Success: true}, nil
	}

	balances, err := uc.reporter.GetSupplierBalances(ctx)
	if err != nil {
		return nil, err
	}
	return &stmtspb.ListSupplierBalancesResponse{
		Balances: balanceMapToRows(balances),
		Success:  true,
	}, nil
}
