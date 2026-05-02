// Package dashboard implements the read-only Payroll Dashboard use case
// (Phase 2 — Pyeza dashboard block + per-app live dashboards plan).
//
// Wiring deferred: the orchestrator must construct
// *GetPayrollDashboardPageDataUseCase from the postgres payroll adapters and
// add it to PayrollUseCases.
package dashboard

import (
	"context"
	"time"

	payrollremittancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_remittance"
	payrollrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_run"
)

// TimeBucket mirrors payroll.TimeBucket.
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

// PayrollStats are the four dashboard stats for payroll.
type PayrollStats struct {
	CurrentRunStatus    string // "draft" | "calculated" | "approved" | "posted" | "" (no run)
	EmployeesInCurrent  int32
	TotalGrossMTD       int64 // centavos
	RemittancesDue30Cnt int64
}

// GetPayrollDashboardPageDataRequest is the request shape.
type GetPayrollDashboardPageDataRequest struct {
	WorkspaceID string
	Now         time.Time
}

// GetPayrollDashboardPageDataResponse is the view-layer projection.
type GetPayrollDashboardPageDataResponse struct {
	Stats              PayrollStats
	LatestRun          *payrollrunpb.PayrollRun
	RecentRuns         []*payrollrunpb.PayrollRun
	UpcomingDeadlines  []*payrollremittancepb.PayrollRemittance
	GrossTrendLabels   []string
	GrossTrendValues   []float64
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

// Execute assembles the dashboard response. Failures degrade gracefully.
func (uc *GetPayrollDashboardPageDataUseCase) Execute(
	ctx context.Context,
	req *GetPayrollDashboardPageDataRequest,
) (*GetPayrollDashboardPageDataResponse, error) {
	if req == nil {
		req = &GetPayrollDashboardPageDataRequest{}
	}
	if req.Now.IsZero() {
		req.Now = time.Now()
	}
	resp := &GetPayrollDashboardPageDataResponse{}

	if uc.runs != nil {
		if latest, err := uc.runs.LatestRun(ctx, req.WorkspaceID); err == nil && latest != nil {
			resp.LatestRun = latest
			resp.Stats.CurrentRunStatus = latest.GetStatus().String()
			resp.Stats.EmployeesInCurrent = latest.GetEmployeeCount()
		}
		if recent, err := uc.runs.RecentRuns(ctx, req.WorkspaceID, 5); err == nil {
			resp.RecentRuns = recent
		}

		// MTD gross.
		mtdStart := time.Date(req.Now.Year(), req.Now.Month(), 1, 0, 0, 0, 0, time.UTC)
		if gross, err := uc.runs.SumTotalGrossInPeriod(ctx, req.WorkspaceID, mtdStart, req.Now); err == nil {
			resp.Stats.TotalGrossMTD = gross
		}

		// 12-month trend ending current month.
		from := req.Now.AddDate(0, -11, 0)
		from = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(req.Now.Year(), req.Now.Month(), 1, 0, 0, 0, 0, time.UTC)
		if buckets, err := uc.runs.SumGrossByMonth(ctx, req.WorkspaceID, from, to); err == nil {
			resp.GrossTrendLabels = make([]string, 0, len(buckets))
			resp.GrossTrendValues = make([]float64, 0, len(buckets))
			for _, b := range buckets {
				resp.GrossTrendLabels = append(resp.GrossTrendLabels, b.Period.Format("Jan"))
				resp.GrossTrendValues = append(resp.GrossTrendValues, float64(b.Value))
			}
		}
	}

	if uc.remittances != nil {
		if n, err := uc.remittances.CountDueWithin(ctx, req.WorkspaceID, 30); err == nil {
			resp.Stats.RemittancesDue30Cnt = n
		}
		if upcoming, err := uc.remittances.UpcomingDeadlines(ctx, req.WorkspaceID, 5); err == nil {
			resp.UpcomingDeadlines = upcoming
		}
	}

	return resp, nil
}
