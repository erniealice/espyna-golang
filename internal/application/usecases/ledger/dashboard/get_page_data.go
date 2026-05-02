// Package dashboard implements the read-only Ledger Dashboard use case
// (Phase 2 — Pyeza dashboard block + per-app live dashboards plan).
//
// The use case orchestrates aggregate queries across the Account and
// JournalEntry repositories to produce the four-stat / three-widget projection
// the fycha ledger dashboard view consumes. It is intentionally read-only and
// does not depend on the auth / transaction / translation services.
//
// Wiring: the orchestrator must construct *GetLedgerDashboardPageDataUseCase
// from the postgres ledger account + journal_entry repositories (which expose
// the dashboard methods as concrete-type methods). See `packages/espyna-golang/
// internal/application/usecases/ledger/usecases.go` for the existing
// container pattern; this dashboard use case is **not yet** wired into
// LedgerUseCases — wiring is deferred to the orchestrator follow-up.
package dashboard

import (
	"context"
	"time"

	journalentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_entry"
)

// AccountDashboardQueries is the slice of the postgres account adapter the
// dashboard use case needs. Implemented by *PostgresAccountRepository in
// contrib/postgres/internal/adapter/ledger/account_dashboard.go.
type AccountDashboardQueries interface {
	SumBalanceByType(ctx context.Context, workspaceID string) (map[string]int64, error)
	CountByStatus(ctx context.Context, workspaceID string) (map[string]int64, error)
}

// JournalEntryDashboardQueries is the slice of the postgres journal_entry
// adapter the dashboard use case needs. Implemented by
// *PostgresJournalEntryRepository in contrib/postgres/internal/adapter/ledger/
// journal_entry_dashboard.go.
type JournalEntryDashboardQueries interface {
	CountByStatus(ctx context.Context, workspaceID string, since time.Time) (map[string]int64, error)
	RecentEntries(ctx context.Context, workspaceID string, limit int32) ([]*journalentrypb.JournalEntry, error)
}

// LedgerStats is the typed stat-card payload the view layer consumes.
// Centavos for monetary values.
type LedgerStats struct {
	TotalAssets      int64
	TotalLiabilities int64
	TotalEquity      int64
	NetIncomeMTD     int64

	UnpostedJournals  int64
	PostedRecentCount int64
}

// GetLedgerDashboardPageDataRequest is the request shape.
type GetLedgerDashboardPageDataRequest struct {
	WorkspaceID string
	Now         time.Time
}

// GetLedgerDashboardPageDataResponse is the projection the view layer reads.
type GetLedgerDashboardPageDataResponse struct {
	Stats          LedgerStats
	BalanceByType  map[string]int64 // keyed by element: asset/liability/equity/revenue/expense
	UnpostedTop    []*journalentrypb.JournalEntry
	RecentEntries  []*journalentrypb.JournalEntry
}

// GetLedgerDashboardPageDataUseCase orchestrates the read-only dashboard
// projection.
type GetLedgerDashboardPageDataUseCase struct {
	accounts AccountDashboardQueries
	journals JournalEntryDashboardQueries
}

// NewGetLedgerDashboardPageDataUseCase constructs the use case.
func NewGetLedgerDashboardPageDataUseCase(
	accounts AccountDashboardQueries,
	journals JournalEntryDashboardQueries,
) *GetLedgerDashboardPageDataUseCase {
	return &GetLedgerDashboardPageDataUseCase{
		accounts: accounts,
		journals: journals,
	}
}

// Execute runs the aggregate queries and assembles the response. Failures of
// individual aggregates degrade gracefully: missing data renders empty stats
// rather than blocking the dashboard.
func (uc *GetLedgerDashboardPageDataUseCase) Execute(
	ctx context.Context,
	req *GetLedgerDashboardPageDataRequest,
) (*GetLedgerDashboardPageDataResponse, error) {
	if req == nil {
		req = &GetLedgerDashboardPageDataRequest{}
	}
	if req.Now.IsZero() {
		req.Now = time.Now()
	}
	resp := &GetLedgerDashboardPageDataResponse{
		BalanceByType: map[string]int64{},
	}

	if uc.accounts != nil {
		if balanceByType, err := uc.accounts.SumBalanceByType(ctx, req.WorkspaceID); err == nil && balanceByType != nil {
			resp.BalanceByType = balanceByType
			resp.Stats.TotalAssets = balanceByType["asset"]
			resp.Stats.TotalLiabilities = balanceByType["liability"]
			resp.Stats.TotalEquity = balanceByType["equity"]
			// Net income MTD ≈ revenue − expense (ledger-side approximation;
			// the proper figure comes from an income-statement use case).
			resp.Stats.NetIncomeMTD = balanceByType["revenue"] - balanceByType["expense"]
		}
	}

	if uc.journals != nil {
		// Last 30 days for "recent" status counts.
		since := req.Now.AddDate(0, 0, -30)
		if statusCounts, err := uc.journals.CountByStatus(ctx, req.WorkspaceID, since); err == nil {
			resp.Stats.UnpostedJournals = statusCounts["DRAFT"]
			resp.Stats.PostedRecentCount = statusCounts["POSTED"]
		}
		if recents, err := uc.journals.RecentEntries(ctx, req.WorkspaceID, 5); err == nil {
			resp.RecentEntries = recents
		}
		if unposted, err := uc.journals.RecentEntries(ctx, req.WorkspaceID, 5); err == nil {
			// Filter unposted from the recents — placeholder until adapter
			// adds a status-filtered variant.
			for _, e := range unposted {
				if e.GetStatus() == journalentrypb.JournalEntryStatus_JOURNAL_ENTRY_STATUS_DRAFT {
					resp.UnpostedTop = append(resp.UnpostedTop, e)
					if len(resp.UnpostedTop) >= 5 {
						break
					}
				}
			}
		}
	}

	return resp, nil
}
