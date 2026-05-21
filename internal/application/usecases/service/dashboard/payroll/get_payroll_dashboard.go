package payroll

import (
	"context"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	payrollremittancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_remittance"
	payrollrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_run"
	payrolldashpb "github.com/erniealice/esqyma/pkg/schema/v1/service/dashboard/payroll"
)

// TimeBucket mirrors the postgres-adapter TimeBucket — kept as a Go-only
// type because it is the output of `PayrollRunDashboardRepository.SumGrossByMonth`.
// Values are centavos.
//
// Per Q-SDM-DASHBOARD-SHARED-TYPES (LOCKED 2026-05-20), TimeBucket stays a
// Go-internal query-interface type per dashboard package — the proto response
// flattens it into parallel `gross_trend_labels` + `gross_trend_values`
// series so view consumers do not depend on a proto TimeBucket shape.
//
// **The postgres adapter `PostgresPayrollRunRepository.SumGrossByMonth`
// aliases its local `TimeBucket` to this type via
// `type TimeBucket = payrolldash.TimeBucket`** so the adapter directly
// satisfies [PayrollRunDashboardRepository] without a type-assertion drift
// (see Q-SDM-DASHBOARD-COMPILE-ASSERTIONS LOCKED 2026-05-20).
type TimeBucket struct {
	Period time.Time
	Value  int64
}

// PayrollRunDashboardRepository is the slice of the postgres payroll_run
// adapter that the payroll dashboard use case consumes. The postgres adapter
// `PostgresPayrollRunRepository` satisfies it — see
// `contrib/postgres/internal/adapter/entity/payroll_dashboard_assertions.go`
// for the compile-time guarantee.
type PayrollRunDashboardRepository interface {
	CountByStatus(ctx context.Context, workspaceID string, since time.Time) (map[string]int64, error)
	SumGrossByMonth(ctx context.Context, workspaceID string, from, to time.Time) ([]TimeBucket, error)
	LatestRun(ctx context.Context, workspaceID string) (*payrollrunpb.PayrollRun, error)
	RecentRuns(ctx context.Context, workspaceID string, limit int32) ([]*payrollrunpb.PayrollRun, error)
	SumTotalGrossInPeriod(ctx context.Context, workspaceID string, from, to time.Time) (int64, error)
}

// PayrollRemittanceDashboardRepository is the slice of the postgres
// payroll_remittance adapter that the payroll dashboard use case consumes.
// The postgres adapter `PostgresPayrollRemittanceRepository` satisfies it —
// see `contrib/postgres/internal/adapter/entity/payroll_dashboard_assertions.go`
// for the compile-time guarantee.
type PayrollRemittanceDashboardRepository interface {
	CountDueWithin(ctx context.Context, workspaceID string, days int) (int64, error)
	UpcomingDeadlines(ctx context.Context, workspaceID string, limit int32) ([]*payrollremittancepb.PayrollRemittance, error)
}

// GetPayrollDashboardRepositories groups the per-repository dependencies the
// service-layer payroll dashboard composes. Any sub-repository may be nil
// when the postgres build tag is inactive (or the type assertion in the
// initializer fails) — the Execute method tolerates nil repositories and
// returns a zero-valued response section for the missing concern.
type GetPayrollDashboardRepositories struct {
	PayrollRun        PayrollRunDashboardRepository
	PayrollRemittance PayrollRemittanceDashboardRepository
}

// GetPayrollDashboardServices groups application services. Translator
// formats error messages. No Authorizer — the dashboard is rendered
// for the active workspace context and the upstream HTTP route is gated by
// session middleware rather than per-entity authcheck (matches the Admin
// pilot at `service/dashboard/admin/`).
type GetPayrollDashboardServices struct {
	Translator ports.Translator
}

// GetPayrollDashboardUseCase composes the two payroll aggregates
// (payroll_run + payroll_remittance) into the service-layer payroll
// dashboard projection.
//
// Per Q-SDM-DASHBOARD-LAYOUT (LOCKED 2026-05-20) and Q-SDM-DASHBOARD-SHARED-TYPES
// (LOCKED 2026-05-20), this use case owns the payroll-dashboard repository
// composition that previously lived at `usecases/payroll/dashboard/`. The
// relocation moves the proto contract out of the entity-driven category
// and into the service-driven category, where it sits alongside the other
// dashboard candidates (Admin, Location, Ledger, Equity, Treasury,
// Schedule, etc.).
type GetPayrollDashboardUseCase struct {
	repositories GetPayrollDashboardRepositories
	services     GetPayrollDashboardServices
}

// NewGetPayrollDashboardUseCase wires the use case from grouped dependencies.
func NewGetPayrollDashboardUseCase(
	repositories GetPayrollDashboardRepositories,
	services GetPayrollDashboardServices,
) *GetPayrollDashboardUseCase {
	return &GetPayrollDashboardUseCase{repositories: repositories, services: services}
}

// Execute assembles the proto response. Each branch is nil-safe so the
// dashboard degrades gracefully on non-postgres builds and missing aggregates.
//
// The Q-SDM-DASHBOARD-COMPILE-ASSERTIONS guard rail (postgres assertion file)
// ensures that whenever the postgres adapter ships, the type assertion in
// the initializer succeeds — the historical Wave B P1.C.1 silent-nil bug
// (codex review P0, 2026-05-20) is structurally prevented for payroll.
func (uc *GetPayrollDashboardUseCase) Execute(
	ctx context.Context,
	req *payrolldashpb.GetPayrollDashboardRequest,
) (*payrolldashpb.GetPayrollDashboardResponse, error) {
	now := time.Now()
	workspaceID := ""
	if req != nil {
		workspaceID = req.GetWorkspaceId()
		if req.GetNowMillis() != 0 {
			now = time.UnixMilli(req.GetNowMillis())
		}
	}

	resp := &payrolldashpb.GetPayrollDashboardResponse{
		Success: true,
		Stats:   &payrolldashpb.PayrollStats{},
	}

	if uc.repositories.PayrollRun != nil {
		if latest, err := uc.repositories.PayrollRun.LatestRun(ctx, workspaceID); err == nil && latest != nil {
			resp.LatestRun = latest
			resp.Stats.CurrentRunStatus = latest.GetStatus().String()
			resp.Stats.EmployeesInCurrent = latest.GetEmployeeCount()
		}
		if recent, err := uc.repositories.PayrollRun.RecentRuns(ctx, workspaceID, 5); err == nil {
			resp.RecentRuns = recent
		}

		// MTD gross.
		mtdStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		if gross, err := uc.repositories.PayrollRun.SumTotalGrossInPeriod(ctx, workspaceID, mtdStart, now); err == nil {
			resp.Stats.TotalGrossMtd = gross
		}

		// 12-month trend ending current month.
		from := now.AddDate(0, -11, 0)
		from = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		if buckets, err := uc.repositories.PayrollRun.SumGrossByMonth(ctx, workspaceID, from, to); err == nil {
			resp.GrossTrendLabels = make([]string, 0, len(buckets))
			resp.GrossTrendValues = make([]float64, 0, len(buckets))
			for _, b := range buckets {
				resp.GrossTrendLabels = append(resp.GrossTrendLabels, b.Period.Format("Jan"))
				resp.GrossTrendValues = append(resp.GrossTrendValues, float64(b.Value))
			}
		}
	}

	if uc.repositories.PayrollRemittance != nil {
		if n, err := uc.repositories.PayrollRemittance.CountDueWithin(ctx, workspaceID, 30); err == nil {
			resp.Stats.RemittancesDue30Cnt = n
		}
		if upcoming, err := uc.repositories.PayrollRemittance.UpcomingDeadlines(ctx, workspaceID, 5); err == nil {
			resp.UpcomingDeadlines = upcoming
		}
	}

	return resp, nil
}
