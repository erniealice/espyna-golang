// Package equity_dashboard implements the read-only Equity Dashboard use case
// (Phase 2 — Pyeza dashboard block + per-app live dashboards plan).
//
// Wiring deferred: the orchestrator must construct
// *GetEquityDashboardPageDataUseCase from the postgres ledger equity_account
// + equity_transaction adapters and add it to LedgerUseCases.
package equity_dashboard

import (
	"context"
	"time"

	equitytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_transaction"
)

// EquityAccountSlice mirrors ledger.EquityAccountSlice in shape.
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

// EquityStats are the four stat values for the dashboard. Centavos.
type EquityStats struct {
	TotalContributed int64
	ActiveOwners     int64
	DistributionsYTD int64
	NetMovementYTD   int64
}

// GetEquityDashboardPageDataRequest is the request shape.
type GetEquityDashboardPageDataRequest struct {
	WorkspaceID string
	Now         time.Time
}

// GetEquityDashboardPageDataResponse is the view-layer projection.
type GetEquityDashboardPageDataResponse struct {
	Stats           EquityStats
	ByTypeYTD       map[string]int64 // contribution / withdrawal / distribution / transfer
	TopContributors []EquityAccountSlice
	Recent          []*equitytransactionpb.EquityTransaction
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

// Execute assembles the equity dashboard response. Failures degrade gracefully.
func (uc *GetEquityDashboardPageDataUseCase) Execute(
	ctx context.Context,
	req *GetEquityDashboardPageDataRequest,
) (*GetEquityDashboardPageDataResponse, error) {
	if req == nil {
		req = &GetEquityDashboardPageDataRequest{}
	}
	if req.Now.IsZero() {
		req.Now = time.Now()
	}
	resp := &GetEquityDashboardPageDataResponse{
		ByTypeYTD: map[string]int64{},
	}

	if uc.accounts != nil {
		if total, err := uc.accounts.SumContributedTotal(ctx, req.WorkspaceID); err == nil {
			resp.Stats.TotalContributed = total
		}
		if owners, err := uc.accounts.CountActive(ctx, req.WorkspaceID); err == nil {
			resp.Stats.ActiveOwners = owners
		}
		if top, err := uc.accounts.TopContributors(ctx, req.WorkspaceID, 5); err == nil {
			resp.TopContributors = top
		}
	}

	if uc.transactions != nil {
		if byType, err := uc.transactions.SumByTypeYTD(ctx, req.WorkspaceID, req.Now.Year()); err == nil && byType != nil {
			resp.ByTypeYTD = byType
			resp.Stats.DistributionsYTD = byType["distribution"]
			// Net movement = contributions − withdrawals − distributions.
			resp.Stats.NetMovementYTD = byType["contribution"] - byType["withdrawal"] - byType["distribution"]
		}
		if recents, err := uc.transactions.RecentTransactions(ctx, req.WorkspaceID, 5); err == nil {
			resp.Recent = recents
		}
	}

	return resp, nil
}
