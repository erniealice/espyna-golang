//go:build mysql

package treasury

import (
	"context"
	"fmt"
	"time"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	treasurydash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/treasury"
)

// TimeBucket is aliased to the service-layer Go-only TimeBucket so the adapter's
// dashboard methods satisfy the service-layer LoanDashboardRepository /
// CollectionDashboardRepository interfaces EXACTLY.
// See contrib/postgres/internal/adapter/treasury/loan_dashboard.go for the
// canonical doc-comment explaining why this alias is mandatory.
type TimeBucket = treasurydash.TimeBucket

// LoanSlice is aliased to the service-layer Go-only LoanSlice for the same reason.
type LoanSlice = treasurydash.LoanSlice

// loanScalarAggregate is the consolidated scalar-count CTE for the loan dashboard
// (Q-DASHBOARD-FAILOPEN, Option A).
type loanScalarAggregate struct {
	TotalOutstanding int64
	ActiveCount      int64
	CompletedCount   int64
	DefaultedCount   int64
}

// loanScalarAggregateQuery is the ONE multi-aggregate CTE for the loan dashboard.
//
// Dialect changes from postgres gold standard:
//   - $1::text IS NULL OR $1::text = ” → ? IS NULL OR ? = ” (MySQL)
//   - COUNT(*) FILTER (WHERE status = 'X') → SUM(CASE WHEN status = 'X' THEN 1 END)
//     MySQL has no FILTER clause; CASE is the portable alternative.
//   - ::bigint cast removed (MySQL returns BIGINT naturally for COUNT/SUM)
//
// Workspace-scoped, centavos, fail-honest (no swallowed errors).
const loanScalarAggregateQuery = `
	WITH base AS (
		SELECT l.status, l.remaining_balance
		FROM loan l
		WHERE l.active = 1
		  AND (? IS NULL OR ? = '' OR l.workspace_id = ?)
	)
	SELECT
		COALESCE(SUM(remaining_balance), 0)                                             AS total_outstanding,
		COALESCE(SUM(CASE WHEN status = 'ACTIVE'    THEN 1 END), 0)                    AS active_count,
		COALESCE(SUM(CASE WHEN status = 'COMPLETED' THEN 1 END), 0)                    AS completed_count,
		COALESCE(SUM(CASE WHEN status = 'DEFAULTED' THEN 1 END), 0)                    AS defaulted_count
	FROM base`

// runLoanScalarAggregate executes the consolidated CTE once and returns the honest error.
func (r *MySQLLoanRepository) runLoanScalarAggregate(
	ctx context.Context,
	workspaceID string,
) (loanScalarAggregate, error) {
	var agg loanScalarAggregate
	if err := mysqlCore.RunDashboardAggregate(
		ctx,
		r.db,
		loanScalarAggregateQuery,
		[]any{workspaceID, workspaceID, workspaceID},
		&agg.TotalOutstanding,
		&agg.ActiveCount,
		&agg.CompletedCount,
		&agg.DefaultedCount,
	); err != nil {
		return loanScalarAggregate{}, err
	}
	return agg, nil
}

// SumOutstanding returns the sum of remaining_balance across all active loans (centavos).
func (r *MySQLLoanRepository) SumOutstanding(ctx context.Context, workspaceID string) (int64, error) {
	agg, err := r.runLoanScalarAggregate(ctx, workspaceID)
	if err != nil {
		return 0, err
	}
	return agg.TotalOutstanding, nil
}

// CountByStatus returns a map of loan status → count. Workspace-scoped.
func (r *MySQLLoanRepository) CountByStatus(ctx context.Context, workspaceID string) (map[string]int64, error) {
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

// SumInterestAccruedYTD sums recorded loan_payment.interest_amount YTD (centavos).
//
// Dialect changes:
//   - $1/$2/$3 → ?/?/?
//   - $1::text IS NULL OR $1::text = ” → ? IS NULL OR ? = ”
//   - ::bigint removed
func (r *MySQLLoanRepository) SumInterestAccruedYTD(
	ctx context.Context,
	workspaceID string,
	year int,
) (int64, error) {
	yearStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	yearEnd := time.Date(year, time.December, 31, 23, 59, 59, 0, time.UTC).Format("2006-01-02")

	// Args: yearStart, yearEnd, workspaceID x3
	const query = `
		SELECT COALESCE(SUM(lp.interest_amount), 0)
		FROM loan_payment lp
		JOIN loan l ON l.id = lp.loan_id
		WHERE lp.payment_date >= ?
		  AND lp.payment_date <= ?
		  AND (? IS NULL OR ? = '' OR l.workspace_id = ?)`

	var total int64
	if err := mysqlCore.RunDashboardAggregate(
		ctx, r.db, query,
		[]any{yearStart, yearEnd, workspaceID, workspaceID, workspaceID},
		&total,
	); err != nil {
		return 0, err
	}
	return total, nil
}

// TopByOutstanding returns active loans ranked by remaining_balance DESC.
//
// Dialect changes: $1/$2 → ?/?; $1::text IS NULL → ? IS NULL;
// NULLS LAST → ORDER BY remaining_balance IS NULL ASC (MySQL 8.0+ does not support NULLS LAST
// syntax — NULL ordering is achieved via IS NULL trick).
func (r *MySQLLoanRepository) TopByOutstanding(
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

	// Args: workspaceID x3, limit
	const query = `
		SELECT
			l.id,
			l.loan_number,
			l.lender_name,
			l.remaining_balance,
			l.principal_amount,
			l.status
		FROM loan l
		WHERE l.active = 1
		  AND (? IS NULL OR ? = '' OR l.workspace_id = ?)
		ORDER BY l.remaining_balance IS NULL ASC, l.remaining_balance DESC
		LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, workspaceID, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top loans: %w", err)
	}
	defer rows.Close()

	out := make([]LoanSlice, 0, limit)
	for rows.Next() {
		var row LoanSlice
		if scanErr := rows.Scan(
			&row.ID, &row.LoanNumber, &row.LenderName,
			&row.RemainingBalance, &row.PrincipalAmount, &row.Status,
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
//
// Dialect changes:
//   - generate_series → recursive CTE calendar
//   - date_trunc('month', ...) → DATE_FORMAT(..., '%Y-%m-01')
//   - interval '1 month' → INTERVAL 1 MONTH
//   - $N::text IS NULL → ? IS NULL
//   - lp.payment_date::timestamp → lp.payment_date (MySQL is already datetime)
func (r *MySQLLoanRepository) OutstandingPrincipalByMonth(
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

	// Recursive CTE: generate one row per month between from and to.
	// Args: from (anchor), to (termination), workspaceID x3, workspaceID x3
	const query = `
		WITH RECURSIVE months AS (
			SELECT DATE_FORMAT(?, '%Y-%m-01') + INTERVAL 0 MONTH AS bucket
			UNION ALL
			SELECT bucket + INTERVAL 1 MONTH FROM months
			WHERE bucket + INTERVAL 1 MONTH <= DATE_FORMAT(?, '%Y-%m-01')
		),
		paid_after AS (
			SELECT m.bucket,
			       COALESCE(SUM(lp.principal_amount), 0) AS paid_principal
			FROM months m
			LEFT JOIN loan_payment lp
			  ON lp.payment_date >= m.bucket + INTERVAL 1 MONTH
			LEFT JOIN loan l ON l.id = lp.loan_id
			WHERE (? IS NULL OR ? = '' OR l.workspace_id = ? OR l.workspace_id IS NULL)
			GROUP BY m.bucket
		),
		active_principal AS (
			SELECT COALESCE(SUM(l.remaining_balance), 0) AS now_balance
			FROM loan l
			WHERE l.active = 1
			  AND (? IS NULL OR ? = '' OR l.workspace_id = ?)
		)
		SELECT m.bucket,
		       (ap.now_balance + COALESCE(p.paid_principal, 0))
		FROM months m
		CROSS JOIN active_principal ap
		LEFT JOIN paid_after p ON p.bucket = m.bucket
		ORDER BY m.bucket ASC`

	rows, err := r.db.QueryContext(ctx, query,
		from, to,
		workspaceID, workspaceID, workspaceID,
		workspaceID, workspaceID, workspaceID,
	)
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
