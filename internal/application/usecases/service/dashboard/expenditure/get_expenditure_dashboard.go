package expenditure

import (
	"context"
	"sort"
	"time"

	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
	expendituredashpb "github.com/erniealice/esqyma/pkg/schema/v1/service/dashboard/expenditure"
)

// TimeBucket is a (period, value) tuple for time-series aggregates. Centavos.
//
// **Named-type contract (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS, LOCKED 2026-05-20):**
// the postgres `ExpenditureDashboardRepository` adapter MUST return EXACTLY
// this named type (via `type TimeBucket = expendituredash.TimeBucket` alias
// on the adapter side). Returning the adapter package's own
// `expenditure.TimeBucket` would silently fail the runtime type assertion in
// `initializers/service.go` (Go interface satisfaction requires exact named
// return type match). See
// `contrib/postgres/internal/adapter/expenditure/expenditure_dashboard_assertions.go`
// for the compile-time guard.
type TimeBucket struct {
	Period time.Time
	Value  int64
}

// TopSupplierRow is one row in the "top suppliers by spend" widget. Centavos.
// SupplierName falls back to SupplierID when no entity row exists.
//
// **Named-type contract (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS, LOCKED 2026-05-20):**
// the postgres adapter MUST return EXACTLY this named type via the
// `type TopSupplierRow = expendituredash.TopSupplierRow` alias.
type TopSupplierRow struct {
	SupplierID   string
	SupplierName string
	Total        int64
}

// ExpenditureDashboardRepository is the slice of the postgres expenditure
// adapter that the expenditure dashboard use case consumes. The postgres
// adapter `PostgresExpenditureRepository` satisfies it — see
// `contrib/postgres/internal/adapter/expenditure/expenditure_dashboard_assertions.go`
// for the compile-time guarantee.
//
// Extension interface — the aggregate methods live on the postgres
// expenditure adapter; this package surfaces them as a Go interface the
// composition root assembles via type assertion.
type ExpenditureDashboardRepository interface {
	CountByStatus(ctx context.Context, workspaceID string, kind string) (map[string]int64, error)
	SumOpenByStatus(ctx context.Context, workspaceID string, kind string) (map[string]int64, error)
	TopBySupplier(ctx context.Context, workspaceID string, kind string, limit int32) ([]TopSupplierRow, error)
	RecentByDate(ctx context.Context, workspaceID string, kind string, limit int32) ([]*expenditurepb.Expenditure, error)
	SumByMonth(ctx context.Context, workspaceID string, kind string, from, to time.Time) ([]TimeBucket, error)
	SumByCategory(ctx context.Context, workspaceID string, kind string, from, to time.Time) (map[string]int64, error)
}

// GetExpenditureDashboardRepositories groups the per-repository dependencies
// the service-layer expenditure dashboard composes. The single repository may
// be nil when the postgres build tag is inactive (or the type assertion in the
// initializer fails) — the Execute method tolerates nil and returns a
// zero-valued response.
type GetExpenditureDashboardRepositories struct {
	Expenditure ExpenditureDashboardRepository
}

// GetExpenditureDashboardUseCase composes the expenditure aggregate into the
// service-layer dashboard projection.
//
// Per Q-SDM-DASHBOARD-LAYOUT (LOCKED 2026-05-20) and Q-SDM-DASHBOARD-SHARED-TYPES
// (LOCKED 2026-05-20), this use case owns the expenditure-dashboard repository
// composition that previously lived at `usecases/expenditure/dashboard/`. The
// relocation moves the proto contract out of the Go-only Request/Response
// shape and into the service-driven category, where it sits alongside the
// other dashboard candidates (Admin, Location, Ledger, Equity, Treasury,
// Payroll, Schedule, etc.).
//
// **No authcheck.Check.** Per hexagonal-rules.md §8 service-driven domains
// take a conditional subset of layers; dashboard reads are authenticated by
// the upstream HTTP view middleware (the dashboard URL resolves only when the
// session is authenticated). This matches the Admin pilot pattern.
type GetExpenditureDashboardUseCase struct {
	repositories GetExpenditureDashboardRepositories
}

// NewGetExpenditureDashboardUseCase wires the use case from grouped dependencies.
func NewGetExpenditureDashboardUseCase(
	repositories GetExpenditureDashboardRepositories,
) *GetExpenditureDashboardUseCase {
	return &GetExpenditureDashboardUseCase{repositories: repositories}
}

// Execute assembles the expenditure dashboard proto response. One use case
// serves both purchase (Kind="purchase") and expense (Kind="expense") surfaces
// — the request's Kind discriminator selects which slice of the aggregate
// the postgres adapter scans. Failures degrade gracefully — a nil
// expenditure adapter returns a zero-valued response.
//
// Per the codex review pattern (Schedule pilot P1 2026-05-20), the
// CategoryLabels/CategoryValues parallel slices come from a Go map; the keys
// are sorted before emission to keep proto output deterministic for cross-
// language clients.
func (uc *GetExpenditureDashboardUseCase) Execute(
	ctx context.Context,
	req *expendituredashpb.GetExpenditureDashboardRequest,
) (*expendituredashpb.GetExpenditureDashboardResponse, error) {
	now := time.Now()
	workspaceID := ""
	kind := "purchase"
	if req != nil {
		workspaceID = req.GetWorkspaceId()
		if req.GetKind() != "" {
			kind = req.GetKind()
		}
		if req.GetNowMillis() != 0 {
			now = time.UnixMilli(req.GetNowMillis())
		}
	}

	resp := &expendituredashpb.GetExpenditureDashboardResponse{
		Success: true,
		Stats:   &expendituredashpb.ExpenditureStats{},
	}
	if uc.repositories.Expenditure == nil {
		return resp, nil
	}

	if byStatus, err := uc.repositories.Expenditure.CountByStatus(ctx, workspaceID, kind); err == nil {
		// Open: pending_approval + draft + approved (not yet paid). Awaiting receipt: approved.
		resp.Stats.OpenCount = byStatus["pending_approval"] + byStatus["draft"] + byStatus["approved"]
		resp.Stats.AwaitingCount = byStatus["approved"]
	}

	// 12-month trend ending current month.
	monthEnd := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthStart := monthEnd.AddDate(0, -11, 0)
	if buckets, err := uc.repositories.Expenditure.SumByMonth(ctx, workspaceID, kind, monthStart, monthEnd); err == nil {
		resp.MonthLabels = make([]string, 0, len(buckets))
		resp.MonthValues = make([]float64, 0, len(buckets))
		for _, b := range buckets {
			resp.MonthLabels = append(resp.MonthLabels, b.Period.Format("Jan"))
			resp.MonthValues = append(resp.MonthValues, float64(b.Value))
		}
	}

	// Spent MTD: current month bucket.
	mtdStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	mtdEnd := mtdStart.AddDate(0, 1, 0)
	if mtdBuckets, err := uc.repositories.Expenditure.SumByMonth(ctx, workspaceID, kind, mtdStart, mtdEnd); err == nil {
		var total int64
		for _, b := range mtdBuckets {
			total += b.Value
		}
		resp.Stats.TotalMtd = total
	}

	if top, err := uc.repositories.Expenditure.TopBySupplier(ctx, workspaceID, kind, 5); err == nil {
		for _, r := range top {
			resp.TopSuppliers = append(resp.TopSuppliers, &expendituredashpb.TopSupplierRow{
				SupplierId:   r.SupplierID,
				SupplierName: r.SupplierName,
				Total:        r.Total,
			})
		}
		if len(top) > 0 {
			resp.Stats.TopSupplierName = top[0].SupplierName
			resp.Stats.TopSupplierTotal = top[0].Total
		}
	}

	if recents, err := uc.repositories.Expenditure.RecentByDate(ctx, workspaceID, kind, 5); err == nil {
		resp.Recent = recents
	}

	// Category breakdown — populated for both surfaces (purchase view ignores).
	// Sorted by key to keep proto output deterministic across language clients.
	if cats, err := uc.repositories.Expenditure.SumByCategory(ctx, workspaceID, kind, mtdStart, mtdEnd); err == nil {
		resp.Stats.CategoryCount = int64(len(cats))
		catKeys := make([]string, 0, len(cats))
		for k := range cats {
			catKeys = append(catKeys, k)
		}
		sort.Strings(catKeys)
		resp.CategoryLabels = make([]string, 0, len(catKeys))
		resp.CategoryValues = make([]float64, 0, len(catKeys))
		for _, k := range catKeys {
			v := cats[k]
			resp.CategoryLabels = append(resp.CategoryLabels, k)
			resp.CategoryValues = append(resp.CategoryValues, float64(v))
			if kind == "expense" {
				resp.Stats.ReimbursableMtd += v
			}
		}
	}

	return resp, nil
}
