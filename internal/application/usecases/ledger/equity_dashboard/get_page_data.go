// Package equity_dashboard implements the read-only Equity Dashboard use case
// (Phase 2 — Pyeza dashboard block + per-app live dashboards plan).
//
// Wiring deferred: the orchestrator must construct
// *GetEquityDashboardPageDataUseCase from the postgres ledger equity_account
// + equity_transaction adapters and add it to LedgerUseCases.
//
// Phase 0i: Execute takes/returns proto types (GetEquityDashboardRequest /
// GetEquityDashboardResponse). The old Go-struct Request/Response/EquityStats/
// EquityAccountSlice are deleted — proto-generated types replace them.
package equity_dashboard

import (
	"context"
	"time"

	dashboardpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_dashboard"
	equitytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_transaction"
)

// EquityAccountSlice mirrors the equity_account row shape for the dashboard
// queries. Kept as a Go-only type because it is the output of
// EquityAccountDashboardQueries.TopContributors.
type EquityAccountSlice struct {
	ID          string
	Name        string
	OwnerName   string
	AccountType string
	Balance     int64
}

// EquityAccountDashboardQueries is the slice of the postgres equity_account
// adapter the dashboard use case needs.
type EquityAccountDashboardQueries interface {
	SumContributedTotal(ctx context.Context, workspaceID string) (int64, error)
	CountActive(ctx context.Context, workspaceID string) (int64, error)
	TopContributors(ctx context.Context, workspaceID string, limit int32) ([]EquityAccountSlice, error)
}

// EquityTransactionDashboardQueries is the slice of the postgres
// equity_transaction adapter the dashboard use case needs.
type EquityTransactionDashboardQueries interface {
	SumByTypeYTD(ctx context.Context, workspaceID string, year int) (map[string]int64, error)
	RecentTransactions(ctx context.Context, workspaceID string, limit int32) ([]*equitytransactionpb.EquityTransaction, error)
}

// GetEquityDashboardPageDataUseCase orchestrates the equity dashboard.
type GetEquityDashboardPageDataUseCase struct {
	accounts     EquityAccountDashboardQueries
	transactions EquityTransactionDashboardQueries
}

// NewGetEquityDashboardPageDataUseCase constructs the use case.
func NewGetEquityDashboardPageDataUseCase(
	accounts EquityAccountDashboardQueries,
	transactions EquityTransactionDashboardQueries,
) *GetEquityDashboardPageDataUseCase {
	return &GetEquityDashboardPageDataUseCase{
		accounts:     accounts,
		transactions: transactions,
	}
}

// Execute assembles the equity dashboard proto response. Failures degrade gracefully.
func (uc *GetEquityDashboardPageDataUseCase) Execute(
	ctx context.Context,
	req *dashboardpb.GetEquityDashboardRequest,
) (*dashboardpb.GetEquityDashboardResponse, error) {
	now := time.Now()
	if req != nil && req.GetNowMillis() != 0 {
		now = time.UnixMilli(req.GetNowMillis())
	}

	workspaceID := ""
	if req != nil {
		workspaceID = req.GetWorkspaceId()
	}

	resp := &dashboardpb.GetEquityDashboardResponse{
		Success:   true,
		Stats:     &dashboardpb.EquityStats{},
		ByTypeYtd: map[string]int64{},
	}

	if uc.accounts != nil {
		if total, err := uc.accounts.SumContributedTotal(ctx, workspaceID); err == nil {
			resp.Stats.TotalContributed = total
		}
		if owners, err := uc.accounts.CountActive(ctx, workspaceID); err == nil {
			resp.Stats.ActiveOwners = owners
		}
		if top, err := uc.accounts.TopContributors(ctx, workspaceID, 5); err == nil {
			for _, c := range top {
				resp.TopContributors = append(resp.TopContributors, &dashboardpb.EquityAccountSlice{
					Id:          c.ID,
					Name:        c.Name,
					OwnerName:   c.OwnerName,
					AccountType: c.AccountType,
					Balance:     c.Balance,
				})
			}
		}
	}

	if uc.transactions != nil {
		if byType, err := uc.transactions.SumByTypeYTD(ctx, workspaceID, now.Year()); err == nil && byType != nil {
			resp.ByTypeYtd = byType
			resp.Stats.DistributionsYtd = byType["distribution"]
			// Net movement = contributions − withdrawals − distributions.
			resp.Stats.NetMovementYtd = byType["contribution"] - byType["withdrawal"] - byType["distribution"]
		}
		if recents, err := uc.transactions.RecentTransactions(ctx, workspaceID, 5); err == nil {
			resp.Recent = recents
		}
	}

	return resp, nil
}
