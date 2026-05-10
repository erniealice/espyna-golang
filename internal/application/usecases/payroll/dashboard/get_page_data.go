// Package dashboard implements the read-only Payroll Dashboard use case
// (Phase 2 — Pyeza dashboard block + per-app live dashboards plan).
//
// Wiring deferred: the orchestrator must construct
// *GetPayrollDashboardPageDataUseCase from the postgres payroll adapters and
// add it to PayrollUseCases.
//
// Phase 0i: Execute takes/returns proto types (GetPayrollDashboardRequest /
// GetPayrollDashboardResponse). The old Go-struct Request/Response/PayrollStats/
// TimeBucket are deleted — proto-generated types replace them.
package dashboard

import (
	"context"
	"time"

	payrollremittancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_remittance"
	payrollrunpb       "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_run"
	dashboardpb        "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/dashboard"
)

// TimeBucket mirrors payroll.TimeBucket — kept as a Go-only type because it
// is the output of PayrollRunDashboardQueries.SumGrossByMonth.
type TimeBucket struct {
	Period time.Time
	Value  int64
}

// PayrollRunDashboardQueries is the slice of the postgres payroll_run adapter
// the dashboard use case needs.
type PayrollRunDashboardQueries interface {
	CountByStatus(ctx context.Context, workspaceID string, since time.Time) (map[string]int64, error)
	SumGrossByMonth(ctx context.Context, workspaceID string, from, to time.Time) ([]TimeBucket, error)
	LatestRun(ctx context.Context, workspaceID string) (*payrollrunpb.PayrollRun, error)
	RecentRuns(ctx context.Context, workspaceID string, limit int32) ([]*payrollrunpb.PayrollRun, error)
	SumTotalGrossInPeriod(ctx context.Context, workspaceID string, from, to time.Time) (int64, error)
}

// PayrollRemittanceDashboardQueries is the slice of the postgres
// payroll_remittance adapter the dashboard use case needs.
type PayrollRemittanceDashboardQueries interface {
	CountDueWithin(ctx context.Context, workspaceID string, days int) (int64, error)
	UpcomingDeadlines(ctx context.Context, workspaceID string, limit int32) ([]*payrollremittancepb.PayrollRemittance, error)
}

// GetPayrollDashboardPageDataUseCase orchestrates the payroll dashboard.
type GetPayrollDashboardPageDataUseCase struct {
	runs        PayrollRunDashboardQueries
	remittances PayrollRemittanceDashboardQueries
}

// NewGetPayrollDashboardPageDataUseCase constructs the use case.
func NewGetPayrollDashboardPageDataUseCase(
	runs PayrollRunDashboardQueries,
	remittances PayrollRemittanceDashboardQueries,
) *GetPayrollDashboardPageDataUseCase {
	return &GetPayrollDashboardPageDataUseCase{
		runs:        runs,
		remittances: remittances,
	}
}

// Execute assembles the dashboard proto response. Failures degrade gracefully.
func (uc *GetPayrollDashboardPageDataUseCase) Execute(
	ctx context.Context,
	req *dashboardpb.GetPayrollDashboardRequest,
) (*dashboardpb.GetPayrollDashboardResponse, error) {
	now := time.Now()
	if req != nil && req.GetNowMillis() != 0 {
		now = time.UnixMilli(req.GetNowMillis())
	}

	workspaceID := ""
	if req != nil {
		workspaceID = req.GetWorkspaceId()
	}

	resp := &dashboardpb.GetPayrollDashboardResponse{
		Success: true,
		Stats:   &dashboardpb.PayrollStats{},
	}

	if uc.runs != nil {
		if latest, err := uc.runs.LatestRun(ctx, workspaceID); err == nil && latest != nil {
			resp.LatestRun = latest
			resp.Stats.CurrentRunStatus = latest.GetStatus().String()
			resp.Stats.EmployeesInCurrent = latest.GetEmployeeCount()
		}
		if recent, err := uc.runs.RecentRuns(ctx, workspaceID, 5); err == nil {
			resp.RecentRuns = recent
		}

		// MTD gross.
		mtdStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		if gross, err := uc.runs.SumTotalGrossInPeriod(ctx, workspaceID, mtdStart, now); err == nil {
			resp.Stats.TotalGrossMtd = gross
		}

		// 12-month trend ending current month.
		from := now.AddDate(0, -11, 0)
		from = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		if buckets, err := uc.runs.SumGrossByMonth(ctx, workspaceID, from, to); err == nil {
			resp.GrossTrendLabels = make([]string, 0, len(buckets))
			resp.GrossTrendValues = make([]float64, 0, len(buckets))
			for _, b := range buckets {
				resp.GrossTrendLabels = append(resp.GrossTrendLabels, b.Period.Format("Jan"))
				resp.GrossTrendValues = append(resp.GrossTrendValues, float64(b.Value))
			}
		}
	}

	if uc.remittances != nil {
		if n, err := uc.remittances.CountDueWithin(ctx, workspaceID, 30); err == nil {
			resp.Stats.RemittancesDue30Cnt = n
		}
		if upcoming, err := uc.remittances.UpcomingDeadlines(ctx, workspaceID, 5); err == nil {
			resp.UpcomingDeadlines = upcoming
		}
	}

	return resp, nil
}
