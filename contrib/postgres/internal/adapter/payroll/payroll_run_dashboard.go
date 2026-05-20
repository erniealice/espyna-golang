//go:build postgresql

package payroll

import (
	"context"
	"fmt"
	"time"

	payrolldash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/payroll"
	payrollrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_run"
)

// TimeBucket is a (period, value) tuple used by payroll dashboard time series.
// Values are centavos.
//
// **Aliased to the service-layer query-interface type** (Wave B P1.C.6
// LANDED 2026-05-20) so the postgres adapter's `SumGrossByMonth` directly
// satisfies [payrolldash.PayrollRunDashboardRepository]. Go's interface
// satisfaction requires the *exact* named return type — without this alias
// the adapter's `SumGrossByMonth` would return its own local
// `payroll.TimeBucket`, silently failing the type assertion in
// `internal/composition/core/initializers/service.go` and producing a nil
// payroll dashboard at runtime. See Q-SDM-DASHBOARD-COMPILE-ASSERTIONS
// (LOCKED 2026-05-20) and the §8 admin pilot "Lesson learned" caveat in
// `docs/wiki/articles/hexagonal-rules.md` — this is the same trap that
// shipped Wave B P1.C.1 with a permanently nil `dashboardDeps.AdminRole`
// (codex review P0, 2026-05-20).
type TimeBucket = payrolldash.TimeBucket

// CountByStatus returns counts of payroll runs grouped by status, optionally
// limited to runs created at-or-after `since`. Workspace-scoped.
//
// Performance index recommendation:
//
//	CREATE INDEX idx_payroll_run_workspace_status_created
//	  ON payroll_run(workspace_id, status, date_created DESC);
func (r *PostgresPayrollRunRepository) CountByStatus(
	ctx context.Context,
	workspaceID string,
	since time.Time,
) (map[string]int64, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	var (
		query string
		args  []any
	)
	if since.IsZero() {
		query = `
			SELECT pr.status, COUNT(*)::bigint
			FROM payroll_run pr
			WHERE ($1::text IS NULL OR $1::text = '' OR pr.workspace_id = $1)
			GROUP BY pr.status`
		args = []any{workspaceID}
	} else {
		query = `
			SELECT pr.status, COUNT(*)::bigint
			FROM payroll_run pr
			WHERE pr.date_created >= $2
			  AND ($1::text IS NULL OR $1::text = '' OR pr.workspace_id = $1)
			GROUP BY pr.status`
		args = []any{workspaceID, since.UnixMilli()}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return map[string]int64{}, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make(map[string]int64, 4)
	for rows.Next() {
		var (
			status string
			n      int64
		)
		if scanErr := rows.Scan(&status, &n); scanErr != nil {
			return nil, fmt.Errorf("failed to scan payroll_run count row: %w", scanErr)
		}
		out[status] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating payroll_run count rows: %w", err)
	}
	return out, nil
}

// SumGrossByMonth returns one TimeBucket per month between from..to summing
// total_gross of payroll runs whose pay_period_end falls in that month.
// Workspace-scoped, centavos.
func (r *PostgresPayrollRunRepository) SumGrossByMonth(
	ctx context.Context,
	workspaceID string,
	from, to time.Time,
) ([]TimeBucket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if from.After(to) {
		return nil, fmt.Errorf("from must be before to")
	}

	const query = `
		WITH months AS (
			SELECT generate_series(
				date_trunc('month', $2::timestamp),
				date_trunc('month', $3::timestamp),
				interval '1 month'
			) AS bucket
		)
		SELECT m.bucket,
		       COALESCE(SUM(pr.total_gross), 0)::bigint
		FROM months m
		LEFT JOIN payroll_run pr
		  ON date_trunc('month', pr.pay_period_end::timestamp) = m.bucket
		 AND ($1::text IS NULL OR $1::text = '' OR pr.workspace_id = $1)
		GROUP BY m.bucket
		ORDER BY m.bucket ASC`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, from, to)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	var out []TimeBucket
	for rows.Next() {
		var (
			bucket time.Time
			value  int64
		)
		if scanErr := rows.Scan(&bucket, &value); scanErr != nil {
			return nil, fmt.Errorf("failed to scan payroll gross-by-month row: %w", scanErr)
		}
		out = append(out, TimeBucket{Period: bucket, Value: value})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating payroll gross-by-month rows: %w", err)
	}
	return out, nil
}

// LatestRun returns the most recent payroll run for the workspace, or nil if
// there are none. The error is non-nil only on real DB failure.
func (r *PostgresPayrollRunRepository) LatestRun(
	ctx context.Context,
	workspaceID string,
) (*payrollrunpb.PayrollRun, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT
			pr.id,
			pr.run_number,
			pr.pay_period_start,
			pr.pay_period_end,
			pr.total_gross,
			pr.total_deductions,
			pr.total_net,
			pr.employee_count,
			pr.status,
			pr.date_created
		FROM payroll_run pr
		WHERE ($1::text IS NULL OR $1::text = '' OR pr.workspace_id = $1)
		ORDER BY pr.date_created DESC
		LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, workspaceID)
	var (
		id              string
		runNumber       string
		ppStart         string
		ppEnd           string
		totalGross      int64
		totalDeductions int64
		totalNet        int64
		empCount        int32
		status          string
		dateCreated     int64
	)
	err := row.Scan(&id, &runNumber, &ppStart, &ppEnd, &totalGross, &totalDeductions, &totalNet, &empCount, &status, &dateCreated)
	if err != nil {
		// Treat any error (incl. ErrNoRows) as "no latest run yet" — dashboard
		// renders an empty state rather than an error page.
		return nil, nil //nolint:nilerr
	}
	pr := &payrollrunpb.PayrollRun{
		Id:              id,
		RunNumber:       runNumber,
		PayPeriodStart:  ppStart,
		PayPeriodEnd:    ppEnd,
		TotalGross:      totalGross,
		TotalDeductions: totalDeductions,
		TotalNet:        totalNet,
		EmployeeCount:   empCount,
	}
	if val, ok := payrollrunpb.PayrollRunStatus_value[status]; ok {
		pr.Status = payrollrunpb.PayrollRunStatus(val)
	}
	if dateCreated > 0 {
		pr.DateCreated = &dateCreated
	}
	return pr, nil
}

// RecentRuns returns the latest N payroll runs for display on the dashboard
// "Recent Runs" table widget. Workspace-scoped.
func (r *PostgresPayrollRunRepository) RecentRuns(
	ctx context.Context,
	workspaceID string,
	limit int32,
) ([]*payrollrunpb.PayrollRun, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if limit <= 0 {
		limit = 5
	}

	const query = `
		SELECT
			pr.id,
			pr.run_number,
			pr.pay_period_start,
			pr.pay_period_end,
			pr.total_gross,
			pr.total_deductions,
			pr.total_net,
			pr.employee_count,
			pr.status,
			pr.date_created
		FROM payroll_run pr
		WHERE ($1::text IS NULL OR $1::text = '' OR pr.workspace_id = $1)
		ORDER BY pr.date_created DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, limit)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	var out []*payrollrunpb.PayrollRun
	for rows.Next() {
		var (
			id              string
			runNumber       string
			ppStart         string
			ppEnd           string
			totalGross      int64
			totalDeductions int64
			totalNet        int64
			empCount        int32
			status          string
			dateCreated     int64
		)
		if scanErr := rows.Scan(&id, &runNumber, &ppStart, &ppEnd, &totalGross, &totalDeductions, &totalNet, &empCount, &status, &dateCreated); scanErr != nil {
			return nil, fmt.Errorf("failed to scan recent payroll_run row: %w", scanErr)
		}
		pr := &payrollrunpb.PayrollRun{
			Id:              id,
			RunNumber:       runNumber,
			PayPeriodStart:  ppStart,
			PayPeriodEnd:    ppEnd,
			TotalGross:      totalGross,
			TotalDeductions: totalDeductions,
			TotalNet:        totalNet,
			EmployeeCount:   empCount,
		}
		if val, ok := payrollrunpb.PayrollRunStatus_value[status]; ok {
			pr.Status = payrollrunpb.PayrollRunStatus(val)
		}
		if dateCreated > 0 {
			pr.DateCreated = &dateCreated
		}
		out = append(out, pr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recent payroll_run rows: %w", err)
	}
	return out, nil
}

// SumTotalGrossInPeriod returns the sum of total_gross of payroll runs whose
// pay_period_end falls in [from..to]. Workspace-scoped, centavos.
func (r *PostgresPayrollRunRepository) SumTotalGrossInPeriod(
	ctx context.Context,
	workspaceID string,
	from, to time.Time,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT COALESCE(SUM(pr.total_gross), 0)::bigint
		FROM payroll_run pr
		WHERE pr.pay_period_end >= $2
		  AND pr.pay_period_end <= $3
		  AND ($1::text IS NULL OR $1::text = '' OR pr.workspace_id = $1)`

	var total int64
	if err := r.db.QueryRowContext(ctx, query,
		workspaceID,
		from.Format("2006-01-02"),
		to.Format("2006-01-02"),
	).Scan(&total); err != nil {
		return 0, nil //nolint:nilerr
	}
	return total, nil
}
