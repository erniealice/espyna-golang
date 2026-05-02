//go:build postgresql

package payroll

import (
	"context"
	"fmt"
	"time"

	payrollremittancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_remittance"
)

// CountDueWithin returns the count of pending remittances whose due_date is
// within [now, now+days]. Workspace-scoped (filtered via the parent payroll_run).
//
// Performance index recommendation:
//
//	CREATE INDEX idx_payroll_remittance_due_status
//	  ON payroll_remittance(status, due_date);
func (r *PostgresPayrollRemittanceRepository) CountDueWithin(
	ctx context.Context,
	workspaceID string,
	days int,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}
	if days <= 0 {
		days = 30
	}

	now := time.Now().Format("2006-01-02")
	until := time.Now().AddDate(0, 0, days).Format("2006-01-02")

	const query = `
		SELECT COUNT(*)::bigint
		FROM payroll_remittance prm
		LEFT JOIN payroll_run pr ON pr.id = prm.payroll_run_id
		WHERE prm.due_date >= $2
		  AND prm.due_date <= $3
		  AND ($1::text IS NULL OR $1::text = '' OR pr.workspace_id = $1)`

	var n int64
	if err := r.db.QueryRowContext(ctx, query, workspaceID, now, until).Scan(&n); err != nil {
		return 0, nil //nolint:nilerr
	}
	return n, nil
}

// UpcomingDeadlines returns the next N remittances ordered by due_date ASC.
// Workspace-scoped via the parent payroll_run.
func (r *PostgresPayrollRemittanceRepository) UpcomingDeadlines(
	ctx context.Context,
	workspaceID string,
	limit int32,
) ([]*payrollremittancepb.PayrollRemittance, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if limit <= 0 {
		limit = 5
	}

	const query = `
		SELECT
			prm.id,
			prm.payroll_run_id,
			prm.remittance_type,
			prm.amount,
			prm.due_date,
			prm.status
		FROM payroll_remittance prm
		LEFT JOIN payroll_run pr ON pr.id = prm.payroll_run_id
		WHERE ($1::text IS NULL OR $1::text = '' OR pr.workspace_id = $1)
		ORDER BY prm.due_date ASC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, limit)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	var out []*payrollremittancepb.PayrollRemittance
	for rows.Next() {
		var (
			id           string
			payrollRunID string
			rType        string
			amount       int64
			dueDate      string
			status       string
		)
		if scanErr := rows.Scan(&id, &payrollRunID, &rType, &amount, &dueDate, &status); scanErr != nil {
			return nil, fmt.Errorf("failed to scan upcoming remittance row: %w", scanErr)
		}
		prm := &payrollremittancepb.PayrollRemittance{
			Id:           id,
			PayrollRunId: payrollRunID,
			Amount:       amount,
			DueDate:      dueDate,
		}
		if val, ok := payrollremittancepb.RemittanceType_value[rType]; ok {
			prm.RemittanceType = payrollremittancepb.RemittanceType(val)
		}
		if val, ok := payrollremittancepb.RemittanceStatus_value[status]; ok {
			prm.Status = payrollremittancepb.RemittanceStatus(val)
		}
		out = append(out, prm)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating upcoming remittance rows: %w", err)
	}
	return out, nil
}
