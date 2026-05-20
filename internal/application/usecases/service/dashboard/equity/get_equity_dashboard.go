package equity

import (
	"context"
	"time"

	equitydashpb "github.com/erniealice/esqyma/pkg/schema/v1/service/dashboard/equity"
	equitytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_transaction"
)

// EquityAccountSlice mirrors the equity_account row shape for the dashboard
// queries. Kept as a Go-only repository return type — the service-layer use
// case projects it onto the proto `EquityAccountSlice` message.
type EquityAccountSlice struct {
	ID          string
	Name        string
	OwnerName   string
	AccountType string
	Balance     int64
}

// EquityAccountDashboardRepository is satisfied by PostgresEquityAccountRepository.
//
// Extension interface — the aggregate Sum/Count/TopContributors methods live
// on the postgres equity_account adapter; this package surfaces them as a
// Go interface the composition root assembles via type assertion.
type EquityAccountDashboardRepository interface {
	SumContributedTotal(ctx context.Context, workspaceID string) (int64, error)
	CountActive(ctx context.Context, workspaceID string) (int64, error)
	TopContributors(ctx context.Context, workspaceID string, limit int32) ([]EquityAccountSlice, error)
}

// EquityTransactionDashboardRepository is satisfied by
// PostgresEquityTransactionRepository.
type EquityTransactionDashboardRepository interface {
	SumByTypeYTD(ctx context.Context, workspaceID string, year int) (map[string]int64, error)
	RecentTransactions(ctx context.Context, workspaceID string, limit int32) ([]*equitytransactionpb.EquityTransaction, error)
}

// GetEquityDashboardRepositories groups the per-repository dependencies the
// service-layer equity dashboard composes. Any sub-repository may be nil
// when the postgres build tag is inactive (or the type assertion in the
// initializer fails) — the Execute method tolerates nil repositories and
// returns a zero-valued response section for the missing concern.
type GetEquityDashboardRepositories struct {
	EquityAccount     EquityAccountDashboardRepository
	EquityTransaction EquityTransactionDashboardRepository
}

// GetEquityDashboardUseCase composes the two ledger aggregates
// (equity_account + equity_transaction) into the service-layer equity
// dashboard projection.
//
// Per Q-SDM-DASHBOARD-LAYOUT (LOCKED 2026-05-20) and Q-SDM-DASHBOARD-SHARED-TYPES
// (LOCKED 2026-05-20), this use case owns the equity-dashboard repository
// composition that previously lived at `usecases/ledger/equity_dashboard/`.
// The relocation moves the proto contract out of the entity/ledger-driven
// category and into the service-driven category, where it sits alongside
// the other dashboard candidates (Admin, Location, Ledger, Treasury, Payroll).
//
// **No authcheck.Check.** Per hexagonal-rules.md §8 service-driven domains
// take a conditional subset of layers; dashboard reads are authenticated by
// the upstream HTTP view middleware (the dashboard URL resolves only when
// the session is authenticated). This matches the Admin pilot pattern.
type GetEquityDashboardUseCase struct {
	repositories GetEquityDashboardRepositories
}

// NewGetEquityDashboardUseCase wires the use case from grouped dependencies.
func NewGetEquityDashboardUseCase(
	repositories GetEquityDashboardRepositories,
) *GetEquityDashboardUseCase {
	return &GetEquityDashboardUseCase{repositories: repositories}
}

// Execute fans out the two aggregate queries and assembles the proto response.
// Each branch is nil-safe so the dashboard degrades gracefully on non-postgres
// builds.
func (uc *GetEquityDashboardUseCase) Execute(
	ctx context.Context,
	req *equitydashpb.GetEquityDashboardRequest,
) (*equitydashpb.GetEquityDashboardResponse, error) {
	now := time.Now()
	if req != nil && req.GetNowMillis() != 0 {
		now = time.UnixMilli(req.GetNowMillis())
	}

	workspaceID := ""
	if req != nil {
		workspaceID = req.GetWorkspaceId()
	}

	resp := &equitydashpb.GetEquityDashboardResponse{
		Success:   true,
		Stats:     &equitydashpb.EquityStats{},
		ByTypeYtd: map[string]int64{},
	}

	if uc.repositories.EquityAccount != nil {
		if total, err := uc.repositories.EquityAccount.SumContributedTotal(ctx, workspaceID); err == nil {
			resp.Stats.TotalContributed = total
		}
		if owners, err := uc.repositories.EquityAccount.CountActive(ctx, workspaceID); err == nil {
			resp.Stats.ActiveOwners = owners
		}
		if top, err := uc.repositories.EquityAccount.TopContributors(ctx, workspaceID, 5); err == nil {
			for _, c := range top {
				resp.TopContributors = append(resp.TopContributors, &equitydashpb.EquityAccountSlice{
					Id:          c.ID,
					Name:        c.Name,
					OwnerName:   c.OwnerName,
					AccountType: c.AccountType,
					Balance:     c.Balance,
				})
			}
		}
	}

	if uc.repositories.EquityTransaction != nil {
		if byType, err := uc.repositories.EquityTransaction.SumByTypeYTD(ctx, workspaceID, now.Year()); err == nil && byType != nil {
			resp.ByTypeYtd = byType
			resp.Stats.DistributionsYtd = byType["distribution"]
			// Net movement = contributions − withdrawals − distributions.
			resp.Stats.NetMovementYtd = byType["contribution"] - byType["withdrawal"] - byType["distribution"]
		}
		if recents, err := uc.repositories.EquityTransaction.RecentTransactions(ctx, workspaceID, 5); err == nil {
			resp.Recent = recents
		}
	}

	return resp, nil
}
