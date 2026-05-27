//go:build sqlserver

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
// Aliased to the service-layer query-interface type so SQLServerPayrollRunRepository
// directly satisfies [payrolldash.PayrollRunDashboardRepository].
type TimeBucket = payrolldash.TimeBucket

// CountByStatus returns counts of payroll runs grouped by status, optionally
// limited to runs created at-or-after `since`. Workspace-scoped.
func (r *SQLServerPayrollRunRepository) CountByStatus(
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
		// Dialect: @p1 placeholder; workspace_id guard in WHERE, not JOIN.
		query = `
			SELECT pr.status, COUNT(*) AS n
			FROM [payroll_run] pr
			WHERE (@p1 = '' OR pr.workspace_id = @p1)
			GROUP BY pr.status`
		args = []any{workspaceID}
	} else {
		query = `
			SELECT pr.status, COUNT(*) AS n
			FROM [payroll_run] pr
			WHERE pr.date_created >= @p2
			  AND (@p1 = '' OR pr.workspace_id = @p1)
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
//
// SQL Server equivalent of Postgres generate_series uses a recursive CTE to
// produce the month spine. SQL Server 2017+ supports recursive CTEs.
func (r *SQLServerPayrollRunRepository) SumGrossByMonth(
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

	// Recursive CTE builds month spine equivalent to Postgres generate_series.
	// DATEFROMPARTS truncates to the 1st of each month.
	// @p1 = workspaceID, @p2 = from (time.Time), @p3 = to (time.Time).
	const query = `
		WITH months AS (
			SELECT DATEFROMPARTS(YEAR(@p2), MONTH(@p2), 1) AS bucket
			UNION ALL
			SELECT DATEADD(MONTH, 1, bucket)
			FROM months
			WHERE DATEADD(MONTH, 1, bucket) <= DATEFROMPARTS(YEAR(@p3), MONTH(@p3), 1)
		)
		SELECT m.bucket,
		       COALESCE(SUM(pr.total_gross), 0)
		FROM months m
		LEFT JOIN [payroll_run] pr
		  ON DATEFROMPARTS(YEAR(CAST(pr.pay_period_end AS date)), MONTH(CAST(pr.pay_period_end AS date)), 1) = m.bucket
		 AND (@p1 = '' OR pr.workspace_id = @p1)
		GROUP BY m.bucket
		ORDER BY m.bucket ASC
		OPTION (MAXRECURSION 36)`

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
//
// Dialect: TOP 1 instead of LIMIT 1; @p1 placeholder.
func (r *SQLServerPayrollRunRepository) LatestRun(
	ctx context.Context,
	workspaceID string,
) (*payrollrunpb.PayrollRun, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT TOP 1
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
		FROM [payroll_run] pr
		WHERE (@p1 = '' OR pr.workspace_id = @p1)
		ORDER BY pr.date_created DESC`

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
		// Treat any error (incl. ErrNoRows) as "no latest run yet".
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

// RecentRuns returns the latest N payroll runs for display on the dashboard.
// Workspace-scoped.
//
// Dialect: TOP @p2 — SQL Server supports parameterised TOP via subquery form;
// using a fixed-format query with TOP N directly from a safe-cast parameter.
func (r *SQLServerPayrollRunRepository) RecentRuns(
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

	// SQL Server does not support parameterised TOP directly; use OFFSET/FETCH instead.
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
		FROM [payroll_run] pr
		WHERE (@p1 = '' OR pr.workspace_id = @p1)
		ORDER BY pr.date_created DESC
		OFFSET 0 ROWS FETCH NEXT @p2 ROWS ONLY`

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
func (r *SQLServerPayrollRunRepository) SumTotalGrossInPeriod(
	ctx context.Context,
	workspaceID string,
	from, to time.Time,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT COALESCE(SUM(pr.total_gross), 0)
		FROM [payroll_run] pr
		WHERE pr.pay_period_end >= @p2
		  AND pr.pay_period_end <= @p3
		  AND (@p1 = '' OR pr.workspace_id = @p1)`

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
