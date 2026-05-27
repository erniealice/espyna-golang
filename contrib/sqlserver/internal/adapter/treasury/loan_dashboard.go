//go:build sqlserver

package treasury

import (
	"context"
	"fmt"
	"time"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	treasurydash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/treasury"
)

// LoanSlice is aliased to the service-layer Go-only LoanSlice so the
// adapter's dashboard methods satisfy the service-layer
// LoanDashboardRepository interface exactly.
type LoanSlice = treasurydash.LoanSlice

// loanScalarAggregate is the ONE consolidated scalar-count CTE for the loan
// dashboard (Q-DASHBOARD-FAILOPEN, Option A). A single base CTE selects the
// active, workspace-scoped loan rows once; the outer SELECT derives every
// scalar metric via consolidated CASE-based aggregation (SQL Server has no
// FILTER (WHERE) clause).
type loanScalarAggregate struct {
	TotalOutstanding int64
	ActiveCount      int64
	CompletedCount   int64
	DefaultedCount   int64
}

// loanScalarAggregateQuery centralizes the consolidated CTE.
// workspace_id is @p1 (multi-tenancy guardrail).
// SQL Server translation:
//   - FILTER (WHERE status = 'ACTIVE') → SUM(CASE WHEN status = 'ACTIVE' THEN 1 END).
//   - active = 1 (BIT) instead of active = true.
//   - (@p1 = ” OR workspace_id = @p1) instead of $1::text IS NULL OR ... = $1.
const loanScalarAggregateQuery = `
	WITH base AS (
		SELECT l.status, l.remaining_balance
		FROM loan l
		WHERE l.active = 1
		  AND (@p1 = '' OR l.workspace_id = @p1)
	)
	SELECT
		COALESCE(SUM(remaining_balance), 0)                                          AS total_outstanding,
		COALESCE(SUM(CASE WHEN status = 'ACTIVE'    THEN 1 END), 0)                 AS active_count,
		COALESCE(SUM(CASE WHEN status = 'COMPLETED' THEN 1 END), 0)                 AS completed_count,
		COALESCE(SUM(CASE WHEN status = 'DEFAULTED' THEN 1 END), 0)                 AS defaulted_count
	FROM base`

// runLoanScalarAggregate executes the consolidated CTE once and returns the
// honest error. Workspace-scoped, centavos.
func (r *SQLServerLoanRepository) runLoanScalarAggregate(
	ctx context.Context,
	workspaceID string,
) (loanScalarAggregate, error) {
	var agg loanScalarAggregate
	if err := sqlserverCore.RunDashboardAggregate(
		ctx,
		r.db,
		loanScalarAggregateQuery,
		[]any{workspaceID},
		&agg.TotalOutstanding,
		&agg.ActiveCount,
		&agg.CompletedCount,
		&agg.DefaultedCount,
	); err != nil {
		return loanScalarAggregate{}, err
	}
	return agg, nil
}

// SumOutstanding returns the sum of remaining_balance across all active loans
// (centavos). Workspace-scoped.
func (r *SQLServerLoanRepository) SumOutstanding(
	ctx context.Context,
	workspaceID string,
) (int64, error) {
	agg, err := r.runLoanScalarAggregate(ctx, workspaceID)
	if err != nil {
		return 0, err
	}
	return agg.TotalOutstanding, nil
}

// CountByStatus returns a map of loan status (ACTIVE/COMPLETED/DEFAULTED) to
// count. Workspace-scoped.
func (r *SQLServerLoanRepository) CountByStatus(
	ctx context.Context,
	workspaceID string,
) (map[string]int64, error) {
	agg, err := r.runLoanScalarAggregate(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return map[string]int64{
		"ACTIVE":    agg.ActiveCount,
		"COMPLETED": agg.CompletedCount,
		"DEFAULTED": agg.DefaultedCount,
	}, nil
}

// SumInterestAccruedYTD sums recorded loan_payment.interest_amount year-to-date
// (centavos). Workspace-scoped.
func (r *SQLServerLoanRepository) SumInterestAccruedYTD(
	ctx context.Context,
	workspaceID string,
	year int,
) (int64, error) {
	yearStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	yearEnd := time.Date(year, time.December, 31, 23, 59, 59, 0, time.UTC).Format("2006-01-02")

	// SQL Server: @pN placeholders; no ::text or CAST needed for string params;
	// active = 1 on the joined loan table.
	const query = `
		SELECT COALESCE(SUM(lp.interest_amount), 0)
		FROM loan_payment lp
		JOIN loan l ON l.id = lp.loan_id
		WHERE lp.payment_date >= @p2
		  AND lp.payment_date <= @p3
		  AND (@p1 = '' OR l.workspace_id = @p1)`

	var total int64
	if err := sqlserverCore.RunDashboardAggregate(
		ctx,
		r.db,
		query,
		[]any{workspaceID, yearStart, yearEnd},
		&total,
	); err != nil {
		return 0, err
	}
	return total, nil
}

// TopByOutstanding returns active loans ranked by remaining_balance DESC.
// Workspace-scoped. List-shaped (multi-row), returned as a slice.
func (r *SQLServerLoanRepository) TopByOutstanding(
	ctx context.Context,
	workspaceID string,
	limit int32,
) ([]LoanSlice, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if limit <= 0 {
		limit = 5
	}

	// SQL Server: TOP @p2 instead of LIMIT $2; no NULLS LAST (SQL Server puts
	// NULLs last for DESC by default — same behaviour, no syntax needed).
	query := fmt.Sprintf(`
		SELECT TOP %d
			l.id,
			l.loan_number,
			l.lender_name,
			l.remaining_balance,
			l.principal_amount,
			l.status
		FROM loan l
		WHERE l.active = 1
		  AND (@p1 = '' OR l.workspace_id = @p1)
		ORDER BY l.remaining_balance DESC`, limit)

	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query top loans: %w", err)
	}
	defer rows.Close()

	out := make([]LoanSlice, 0, limit)
	for rows.Next() {
		var row LoanSlice
		if scanErr := rows.Scan(
			&row.ID,
			&row.LoanNumber,
			&row.LenderName,
			&row.RemainingBalance,
			&row.PrincipalAmount,
			&row.Status,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan top loan row: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating top loan rows: %w", err)
	}
	return out, nil
}

// OutstandingPrincipalByMonth returns one TimeBucket per month in the range.
// SQL Server: recursive CTE for month generation (no generate_series);
// OUTER APPLY replaces LEFT JOIN LATERAL.
func (r *SQLServerLoanRepository) OutstandingPrincipalByMonth(
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

	// Truncate to month start.
	fromMonth := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	toMonth := time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Estimate max recursion depth (months between fromMonth and toMonth + 1).
	months := int(toMonth.Sub(fromMonth).Hours()/24/28) + 2
	if months < 2 {
		months = 2
	}

	query := fmt.Sprintf(`
		WITH months AS (
			SELECT CAST(@p2 AS date) AS bucket
			UNION ALL
			SELECT DATEADD(month, 1, bucket) FROM months
			WHERE bucket < CAST(@p3 AS date)
		),
		active_principal AS (
			SELECT COALESCE(SUM(l.remaining_balance), 0) AS now_balance
			FROM loan l
			WHERE l.active = 1
			  AND (@p1 = '' OR l.workspace_id = @p1)
		)
		SELECT m.bucket,
		       ap.now_balance + COALESCE(paid.paid_principal, 0) AS value
		FROM months m
		CROSS JOIN active_principal ap
		OUTER APPLY (
			SELECT COALESCE(SUM(lp.principal_amount), 0) AS paid_principal
			FROM loan_payment lp
			JOIN loan l ON l.id = lp.loan_id
			WHERE lp.payment_date >= DATEADD(month, 1, m.bucket)
			  AND (@p1 = '' OR l.workspace_id = @p1)
		) paid
		ORDER BY m.bucket ASC
		OPTION (MAXRECURSION %d)`, months)

	rows, err := r.db.QueryContext(ctx, query, workspaceID, fromMonth, toMonth)
	if err != nil {
		return nil, fmt.Errorf("failed to query outstanding-by-month: %w", err)
	}
	defer rows.Close()

	var out []TimeBucket
	for rows.Next() {
		var (
			bucket time.Time
			value  int64
		)
		if scanErr := rows.Scan(&bucket, &value); scanErr != nil {
			return nil, fmt.Errorf("failed to scan outstanding-by-month row: %w", scanErr)
		}
		out = append(out, TimeBucket{Period: bucket, Value: value})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating outstanding-by-month rows: %w", err)
	}
	return out, nil
}
