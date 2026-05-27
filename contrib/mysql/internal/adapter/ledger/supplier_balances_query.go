//go:build mysql

package ledger

import "fmt"

// buildSupplierBalancesQuery returns a SQL query that computes the outstanding
// balance (total billed minus total paid) for every supplier that has at least
// one non-cancelled, non-draft expenditure. Suppliers with a net-zero balance
// are excluded via the HAVING clause.
//
// Parameters:
//
//	? = workspace_id (text or NULL)
//
// Dialect differences from postgres gold standard:
//   - $1::text IS NULL → ? IS NULL (MySQL driver passes nil directly)
//
// Returns two columns: supplier_id (text), outstanding (bigint centavos).
func buildSupplierBalancesQuery(tc TableConfig) (string, []any) {
	return fmt.Sprintf(`
SELECT
    e.supplier_id,
    COALESCE(SUM(e.total_amount), 0) - COALESCE(SUM(paid.total_paid), 0) AS outstanding
FROM %s e
LEFT JOIN (
    SELECT d.expenditure_id, SUM(d.amount) AS total_paid
    FROM %s d
    WHERE d.active = 1 AND d.status IN ('paid', 'completed')
    GROUP BY d.expenditure_id
) paid ON paid.expenditure_id = e.id
WHERE e.active = 1
  AND e.status NOT IN ('cancelled', 'draft')
  AND e.supplier_id IS NOT NULL
  AND (? IS NULL OR e.workspace_id = ?)
GROUP BY e.supplier_id
HAVING COALESCE(SUM(e.total_amount), 0) - COALESCE(SUM(paid.total_paid), 0) != 0`,
		tc.Expenditure, tc.TreasuryDisbursement), nil
}
