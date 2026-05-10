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
// container pattern.
//
// Phase 0i: Execute takes/returns proto types (GetLedgerDashboardRequest /
// GetLedgerDashboardResponse). The old Go-struct Request/Response/LedgerStats
// are deleted — proto-generated types replace them.
package dashboard

import (
	"context"
	"time"

	journalentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_entry"
	dashboardpb    "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/dashboard"
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

// Execute runs the aggregate queries and assembles the proto response. Failures of
// individual aggregates degrade gracefully: missing data renders empty stats
// rather than blocking the dashboard.
func (uc *GetLedgerDashboardPageDataUseCase) Execute(
	ctx context.Context,
	req *dashboardpb.GetLedgerDashboardRequest,
) (*dashboardpb.GetLedgerDashboardResponse, error) {
	// Resolve "now" from proto millis or server time.
	now := time.Now()
	if req != nil && req.GetNowMillis() != 0 {
		now = time.UnixMilli(req.GetNowMillis())
	}

	workspaceID := ""
	if req != nil {
		workspaceID = req.GetWorkspaceId()
	}

	resp := &dashboardpb.GetLedgerDashboardResponse{
		Success:       true,
		Stats:         &dashboardpb.LedgerStats{},
		BalanceByType: map[string]int64{},
	}

	if uc.accounts != nil {
		if balanceByType, err := uc.accounts.SumBalanceByType(ctx, workspaceID); err == nil && balanceByType != nil {
			resp.BalanceByType = balanceByType
			resp.Stats.TotalAssets = balanceByType["asset"]
			resp.Stats.TotalLiabilities = balanceByType["liability"]
			resp.Stats.TotalEquity = balanceByType["equity"]
			// Net income MTD ≈ revenue − expense (ledger-side approximation;
			// the proper figure comes from an income-statement use case).
			resp.Stats.NetIncomeMtd = balanceByType["revenue"] - balanceByType["expense"]
		}
	}

	if uc.journals != nil {
		// Last 30 days for "recent" status counts.
		since := now.AddDate(0, 0, -30)
		if statusCounts, err := uc.journals.CountByStatus(ctx, workspaceID, since); err == nil {
			resp.Stats.UnpostedJournals = statusCounts["DRAFT"]
			resp.Stats.PostedRecentCount = statusCounts["POSTED"]
		}
		if recents, err := uc.journals.RecentEntries(ctx, workspaceID, 5); err == nil {
			resp.RecentEntries = recents
		}
		if unposted, err := uc.journals.RecentEntries(ctx, workspaceID, 5); err == nil {
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
