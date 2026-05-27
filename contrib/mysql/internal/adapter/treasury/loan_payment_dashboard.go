//go:build mysql

package treasury

import (
	"context"
	"fmt"
	"time"

	loanpaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan_payment"
)

// SumDueWithin returns an estimate of payments coming due in the next `days`
// days, derived from loans with maturity_date in the window (centavos).
// Workspace-scoped via parent loan.workspace_id.
//
// Dialect changes: $N → ?; $1::text IS NULL → ? IS NULL; ::bigint removed.
func (r *MySQLLoanPaymentRepository) SumDueWithin(
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

	// Args: now, until, workspaceID x3
	const query = `
		SELECT COALESCE(SUM(l.remaining_balance), 0)
		FROM loan l
		WHERE l.active = 1
		  AND l.maturity_date >= ?
		  AND l.maturity_date <= ?
		  AND (? IS NULL OR ? = '' OR l.workspace_id = ?)`

	var total int64
	if err := r.db.QueryRowContext(ctx, query, now, until, workspaceID, workspaceID, workspaceID).Scan(&total); err != nil {
		return 0, nil //nolint:nilerr
	}
	return total, nil
}

// RecentByLoan returns the latest loan payments across all loans in the
// workspace, newest-first. Workspace-scoped via parent loan.workspace_id.
//
// Dialect changes: $1/$2 → ?/?; $1::text IS NULL OR ... → ? IS NULL OR ...
func (r *MySQLLoanPaymentRepository) RecentByLoan(
	ctx context.Context,
	workspaceID string,
	limit int32,
) ([]*loanpaymentpb.LoanPayment, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if limit <= 0 {
		limit = 5
	}

	// Args: workspaceID x3, limit
	const query = `
		SELECT
			lp.id,
			lp.loan_id,
			lp.payment_number,
			lp.payment_date,
			lp.principal_amount,
			lp.interest_amount,
			lp.fee_amount,
			lp.total_amount,
			lp.remaining_balance,
			lp.date_created
		FROM loan_payment lp
		JOIN loan l ON l.id = lp.loan_id
		WHERE (? IS NULL OR ? = '' OR l.workspace_id = ?)
		ORDER BY lp.payment_date DESC, lp.date_created DESC
		LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, workspaceID, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent loan payments: %w", err)
	}
	defer rows.Close()

	var out []*loanpaymentpb.LoanPayment
	for rows.Next() {
		var (
			id               string
			loanID           string
			paymentNumber    string
			paymentDate      string
			principalAmount  int64
			interestAmount   int64
			feeAmount        int64
			totalAmount      int64
			remainingBalance int64
			dateCreated      time.Time
		)
		if scanErr := rows.Scan(
			&id, &loanID, &paymentNumber, &paymentDate,
			&principalAmount, &interestAmount, &feeAmount,
			&totalAmount, &remainingBalance, &dateCreated,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan recent loan payment row: %w", scanErr)
		}
		p := &loanpaymentpb.LoanPayment{
			Id:               id,
			LoanId:           loanID,
			PaymentNumber:    paymentNumber,
			PaymentDate:      paymentDate,
			PrincipalAmount:  principalAmount,
			InterestAmount:   interestAmount,
			FeeAmount:        feeAmount,
			TotalAmount:      totalAmount,
			RemainingBalance: remainingBalance,
		}
		if !dateCreated.IsZero() {
			ms := dateCreated.UnixMilli()
			p.DateCreated = &ms
			s := dateCreated.Format(time.RFC3339)
			p.DateCreatedString = &s
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recent loan payment rows: %w", err)
	}
	return out, nil
}
