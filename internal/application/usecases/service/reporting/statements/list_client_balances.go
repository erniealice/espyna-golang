package statements

import (
	"context"
	"errors"
	"sort"

	stmtspb "github.com/erniealice/esqyma/pkg/schema/v1/service/reporting/statements"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

// ListClientBalancesUseCase is the proto-shaped wrapper for the per-client
// outstanding-balance lookup. The legacy adapter returns
// `map[string]int64` (client_id → centavo balance); per Q-SDM-MAP-SHAPES
// the new contract emits `repeated BalanceRow` (typed proto rows).
//
// **Sort order:** rows are sorted by `counterparty_id` ascending. This
// gives downstream consumers a deterministic ordering they can rely on
// when paginating/diffing successive responses; the map-shaped legacy
// API gave no ordering guarantee.
type ListClientBalancesUseCase struct {
	reporter             reporter
	authorizationService ports.Authorizer
	translationService   ports.Translator
	actionGatekeeper  *actiongate.ActionGatekeeper
}

// NewListClientBalancesUseCase wires the use case with nil-safe deps.
func NewListClientBalancesUseCase(
	r reporter,
	authSvc ports.Authorizer,
	i18nSvc ports.Translator,
) *ListClientBalancesUseCase {
	if i18nSvc == nil {
		i18nSvc = ports.NewNoOpTranslator()
	}
	return &ListClientBalancesUseCase{
		reporter:             r,
		authorizationService: authSvc,
		translationService:   i18nSvc,
	}
}

// Execute runs the per-client balance lookup under the "reports" +
// ActionList authcheck.
func (uc *ListClientBalancesUseCase) Execute(
	ctx context.Context,
	req *stmtspb.ListClientBalancesRequest,
) (*stmtspb.ListClientBalancesResponse, error) {
	if err := uc.actionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "reports",
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.translationService,
			"reports.validation.request_required", "List client balances request is required [DEFAULT]"))
	}
	if uc.reporter == nil {
		return &stmtspb.ListClientBalancesResponse{Success: true}, nil
	}

	balances, err := uc.reporter.GetClientBalances(ctx)
	if err != nil {
		return nil, err
	}
	return &stmtspb.ListClientBalancesResponse{
		Balances: balanceMapToRows(balances),
		Success:  true,
	}, nil
}

// balanceMapToRows converts the legacy `map[string]int64` shape into the
// typed `repeated BalanceRow` proto response. Sorted by counterparty_id
// for deterministic ordering.
func balanceMapToRows(balances map[string]int64) []*stmtspb.BalanceRow {
	if len(balances) == 0 {
		return nil
	}
	ids := make([]string, 0, len(balances))
	for id := range balances {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	rows := make([]*stmtspb.BalanceRow, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, &stmtspb.BalanceRow{
			CounterpartyId: id,
			AmountCentavos: balances[id],
		})
	}
	return rows
}
