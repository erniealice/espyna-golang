//go:build postgresql

package ledger

// Application leg of the receivables report queries (20260927-usage-and-pass-through-charges,
// build-spec §6.1/§6.3, C4, AC-VERT-01). The four report queries gain an ADDITIVE leg over
// collection_application: `Σ collection_application.amount` for CASH + APPLIED applications on
// the revenue target, plus (statement only) recovery documents as billed rows and every cash
// application as a received row. With no collection_application rows every added term is
// COALESCE(...,0)/an empty UNION branch, so results are identical to the pre-S1 queries.
//
// Workspace scoping of the new leg is strict and adds no `$n IS NULL OR` wildcard: the revenue-target
// subtotal carries no workspace parameter at all and is tied to its parent row by the caller's join
// (`<alias>.workspace_id = r.workspace_id`); the standalone application/receipt/document rows require
// a non-NULL workspace parameter, and their joins to referenced rows repeat the workspace predicate.
//
// Table names come from TableConfig (CollectionApplication, RecoveryDocument), never literals.
//
// The enum literals below are the stored proto names (CHECK-constrained in migration
// 20260930110000); they are constants, never caller input.

const (
	appStatusApplied = "APPLICATION_STATUS_APPLIED"
	appKindCash      = "APPLICATION_KIND_CASH"
	appTargetRevenue = "APPLICATION_TARGET_KIND_REVENUE"

	recoveryDocStatusIssued   = "RECOVERY_DOCUMENT_STATUS_ISSUED"
	recoveryDocTypeCreditNote = "RECOVERY_DOCUMENT_TYPE_CREDIT_NOTE"

	// receiptCollectionType marks treasury_collection rows created by ReceiveAndApplyCollection
	// (collection_application.ReceiptCollectionType); they carry revenue_id NULL.
	receiptCollectionType = "receipt"
)

// appliedToRevenueSubquery returns a per-revenue, per-workspace subtotal of CASH APPLIED applications
// from table (TableConfig.CollectionApplication). extraWhere (may be empty) is appended with AND. It
// takes no workspace parameter: the caller must join it on `workspace_id = r.workspace_id`.
func appliedToRevenueSubquery(table, extraWhere string) string {
	q := `(
    SELECT ca.revenue_id, ca.workspace_id, SUM(ca.amount) AS total_applied
    FROM ` + table + ` ca
    WHERE ca.active = true
      AND ca.status = '` + appStatusApplied + `'
      AND ca.application_kind = '` + appKindCash + `'
      AND ca.target_kind = '` + appTargetRevenue + `'`
	if extraWhere != "" {
		q += "\n      AND " + extraWhere
	}
	q += `
    GROUP BY ca.revenue_id, ca.workspace_id
)`
	return q
}
