//go:build mysql

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
// Aliased to the service-layer query-interface type so MySQLPayrollRunRepository's
// SumGrossByMonth satisfies payrolldash.PayrollRunDashboardRepository. Mirrors the
// postgres gold standard's alias pattern (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS).
type TimeBucket = payrolldash.TimeBucket

// CountByStatus returns counts of payroll runs grouped by status, optionally
// limited to runs created at-or-after `since`. Workspace-scoped.
//
// Dialect changes from postgres gold standard:
//   - $1::text IS NULL → (? IS NULL OR ? = ” OR pr.workspace_id = ?)
//   - COUNT(*)::bigint → CAST(COUNT(*) AS SIGNED) (MySQL)
//   - pr.date_created >= $2 with UnixMilli arg stays identical
func (r *MySQLPayrollRunRepository) CountByStatus(
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
		// Dialect: (? IS NULL OR ? = '' OR pr.workspace_id = ?) replaces $1::text IS NULL check.
		// CAST(COUNT(*) AS SIGNED) replaces COUNT(*)::bigint.
		query = `
			SELECT pr.status, CAST(COUNT(*) AS SIGNED)
			FROM payroll_run pr
			WHERE (? IS NULL OR ? = '' OR pr.workspace_id = ?)
			GROUP BY pr.status`
		args = []any{workspaceID, workspaceID, workspaceID}
	} else {
		query = `
			SELECT pr.status, CAST(COUNT(*) AS SIGNED)
			FROM payroll_run pr
			WHERE pr.date_created >= ?
			  AND (? IS NULL OR ? = '' OR pr.workspace_id = ?)
			GROUP BY pr.status`
		args = []any{since.UnixMilli(), workspaceID, workspaceID, workspaceID}
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
//
// Dialect changes from postgres gold standard:
//   - generate_series() → recursive CTE with DATE_ADD to expand months (MySQL 8.0+)
//   - date_trunc('month', ...) → DATE_FORMAT(date, '%Y-%m-01') (MySQL)
//   - $N → ? (positional, re-sequenced)
//   - ::timestamp casts removed (MySQL infers types)
func (r *MySQLPayrollRunRepository) SumGrossByMonth(
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

	// MySQL has no generate_series(). Use a recursive CTE to expand months.
	// DATE_FORMAT(date, '%Y-%m-01') truncates to the first day of the month,
	// matching postgres date_trunc('month', ...).
	const query = `
		WITH RECURSIVE months AS (
			SELECT DATE_FORMAT(?, '%Y-%m-01') AS bucket
			UNION ALL
			SELECT DATE_FORMAT(DATE_ADD(bucket, INTERVAL 1 MONTH), '%Y-%m-01')
			FROM months
			WHERE DATE_ADD(bucket, INTERVAL 1 MONTH) <= DATE_FORMAT(?, '%Y-%m-01')
		)
		SELECT m.bucket,
		       COALESCE(SUM(pr.total_gross), 0)
		FROM months m
		LEFT JOIN payroll_run pr
		  ON DATE_FORMAT(pr.pay_period_end, '%Y-%m-01') = m.bucket
		 AND (? IS NULL OR ? = '' OR pr.workspace_id = ?)
		GROUP BY m.bucket
		ORDER BY m.bucket ASC`

	rows, err := r.db.QueryContext(ctx, query,
		from.Format("2006-01-02"),
		to.Format("2006-01-02"),
		workspaceID, workspaceID, workspaceID,
	)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	var out []TimeBucket
	for rows.Next() {
		var (
			bucketStr string
			value     int64
		)
		if scanErr := rows.Scan(&bucketStr, &value); scanErr != nil {
			return nil, fmt.Errorf("failed to scan payroll gross-by-month row: %w", scanErr)
		}
		bucket, err := time.Parse("2006-01-02", bucketStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse month bucket %q: %w", bucketStr, err)
		}
		out = append(out, TimeBucket{Period: bucket, Value: value})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating payroll gross-by-month rows: %w", err)
	}
	return out, nil
}

// LatestRun returns the most recent payroll run for the workspace, or nil if none.
//
// Dialect changes: $1::text IS NULL → (? IS NULL OR ? = ” OR pr.workspace_id = ?);
// ? placeholder replaces $1/$2.
func (r *MySQLPayrollRunRepository) LatestRun(
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
		WHERE (? IS NULL OR ? = '' OR pr.workspace_id = ?)
		ORDER BY pr.date_created DESC
		LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, workspaceID, workspaceID, workspaceID)
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
//
// Dialect changes: $1::text IS NULL → (? IS NULL OR ? = ” OR pr.workspace_id = ?);
// LIMIT $2 → LIMIT ?.
func (r *MySQLPayrollRunRepository) RecentRuns(
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
		WHERE (? IS NULL OR ? = '' OR pr.workspace_id = ?)
		ORDER BY pr.date_created DESC
		LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, workspaceID, workspaceID, limit)
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
//
// Dialect changes: COALESCE(SUM(...), 0)::bigint → COALESCE(SUM(...), 0);
// $1::text IS NULL → (? IS NULL OR ? = ” OR pr.workspace_id = ?).
func (r *MySQLPayrollRunRepository) SumTotalGrossInPeriod(
	ctx context.Context,
	workspaceID string,
	from, to time.Time,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT COALESCE(SUM(pr.total_gross), 0)
		FROM payroll_run pr
		WHERE pr.pay_period_end >= ?
		  AND pr.pay_period_end <= ?
		  AND (? IS NULL OR ? = '' OR pr.workspace_id = ?)`

	var total int64
	if err := r.db.QueryRowContext(ctx, query,
		from.Format("2006-01-02"),
		to.Format("2006-01-02"),
		workspaceID, workspaceID, workspaceID,
	).Scan(&total); err != nil {
		return 0, nil //nolint:nilerr
	}
	return total, nil
}
