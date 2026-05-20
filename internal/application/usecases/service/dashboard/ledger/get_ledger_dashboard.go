package ledger

import (
	"context"
	"time"

	journalentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_entry"
	ledgerdashpb "github.com/erniealice/esqyma/pkg/schema/v1/service/dashboard/ledger"
)

// AccountDashboardRepository is satisfied by PostgresAccountRepository.
//
// Extension interface — the aggregate Sum/Count methods live on the postgres
// account adapter; this package surfaces them as a Go interface the
// composition root assembles via type assertion.
type AccountDashboardRepository interface {
	SumBalanceByType(ctx context.Context, workspaceID string) (map[string]int64, error)
	CountByStatus(ctx context.Context, workspaceID string) (map[string]int64, error)
}

// JournalEntryDashboardRepository is satisfied by PostgresJournalEntryRepository.
type JournalEntryDashboardRepository interface {
	CountByStatus(ctx context.Context, workspaceID string, since time.Time) (map[string]int64, error)
	RecentEntries(ctx context.Context, workspaceID string, limit int32) ([]*journalentrypb.JournalEntry, error)
}

// GetLedgerDashboardRepositories groups the per-repository dependencies the
// service-layer ledger dashboard composes. Any sub-repository may be nil when
// the postgres build tag is inactive (or the type assertion in the
// initializer fails) — the Execute method tolerates nil repositories and
// returns a zero-valued response section for the missing concern.
type GetLedgerDashboardRepositories struct {
	Account      AccountDashboardRepository
	JournalEntry JournalEntryDashboardRepository
}

// GetLedgerDashboardUseCase composes the account + journal_entry aggregates
// into the service-layer ledger dashboard projection.
//
// Per Q-SDM-DASHBOARD-LAYOUT (LOCKED 2026-05-20) and Q-SDM-DASHBOARD-SHARED-TYPES
// (LOCKED 2026-05-20), this use case owns the ledger-dashboard repository
// composition that previously lived at `usecases/ledger/dashboard/`. The
// relocation moves the proto contract out of the ledger-domain category and
// into the service-driven category, where it sits alongside the other
// dashboard candidates (Admin, Location, Equity, Treasury, Payroll).
//
// **No authcheck.Check.** Per hexagonal-rules.md §8 service-driven domains
// take a conditional subset of layers; dashboard reads are authenticated by
// the upstream HTTP view middleware (the dashboard URL resolves only when
// the session is authenticated). This matches the Admin/Location/Equity
// pilot pattern.
type GetLedgerDashboardUseCase struct {
	repositories GetLedgerDashboardRepositories
}

// NewGetLedgerDashboardUseCase wires the use case from grouped dependencies.
func NewGetLedgerDashboardUseCase(
	repositories GetLedgerDashboardRepositories,
) *GetLedgerDashboardUseCase {
	return &GetLedgerDashboardUseCase{repositories: repositories}
}

// Execute fans out the two aggregate queries and assembles the proto response.
// Each branch is nil-safe so the dashboard degrades gracefully on non-postgres
// builds.
func (uc *GetLedgerDashboardUseCase) Execute(
	ctx context.Context,
	req *ledgerdashpb.GetLedgerDashboardRequest,
) (*ledgerdashpb.GetLedgerDashboardResponse, error) {
	now := time.Now()
	if req != nil && req.GetNowMillis() != 0 {
		now = time.UnixMilli(req.GetNowMillis())
	}

	workspaceID := ""
	if req != nil {
		workspaceID = req.GetWorkspaceId()
	}

	resp := &ledgerdashpb.GetLedgerDashboardResponse{
		Success:       true,
		Stats:         &ledgerdashpb.LedgerStats{},
		BalanceByType: map[string]int64{},
	}

	if uc.repositories.Account != nil {
		if balanceByType, err := uc.repositories.Account.SumBalanceByType(ctx, workspaceID); err == nil && balanceByType != nil {
			resp.BalanceByType = balanceByType
			resp.Stats.TotalAssets = balanceByType["asset"]
			resp.Stats.TotalLiabilities = balanceByType["liability"]
			resp.Stats.TotalEquity = balanceByType["equity"]
			// Net income MTD ≈ revenue − expense (ledger-side approximation;
			// the proper figure comes from an income-statement use case).
			resp.Stats.NetIncomeMtd = balanceByType["revenue"] - balanceByType["expense"]
		}
	}

	if uc.repositories.JournalEntry != nil {
		// Last 30 days for "recent" status counts.
		since := now.AddDate(0, 0, -30)
		if statusCounts, err := uc.repositories.JournalEntry.CountByStatus(ctx, workspaceID, since); err == nil {
			resp.Stats.UnpostedJournals = statusCounts["DRAFT"]
			resp.Stats.PostedRecentCount = statusCounts["POSTED"]
		}
		if recents, err := uc.repositories.JournalEntry.RecentEntries(ctx, workspaceID, 5); err == nil {
			resp.RecentEntries = recents
		}
		if unposted, err := uc.repositories.JournalEntry.RecentEntries(ctx, workspaceID, 5); err == nil {
			// Filter unposted from the recents — placeholder until adapter
			// adds a status-filtered variant. Mirrors the entity-layer logic
			// that this package absorbs.
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
