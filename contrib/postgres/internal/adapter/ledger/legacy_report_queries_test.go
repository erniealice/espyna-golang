//go:build postgresql

package ledger

// Pre-S1 baselines of the four receivables report queries, frozen verbatim from the commit before
// the application leg (20260927-usage-and-pass-through-charges). TestReportQueriesUnchangedWithoutApplications
// (AC-VERT-01) runs each baseline and the current query on the same database with no collection_application
// rows and requires identical result sets. Do not edit these to follow the live queries. The ONLY delta from
// the pre-S1 text is the report-type-drift fix (RD, 20260930): live revenue.due_date/revenue_date are
// timestamptz and treasury_collection.payment_date is text, so the same SQL now casts them explicitly.

import (
	"fmt"

	clientstmtpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/client_statement"
	agingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/receivables_aging"
	collsumpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/collection_summary"
	"time"
)

var _ = time.Now
var _ = fmt.Sprintf
var _ *agingpb.ReceivablesAgingRequest
var _ *clientstmtpb.ClientStatementRequest
var _ *collsumpb.CollectionSummaryRequest

func legacyBuildClientBalancesQuery(tc TableConfig) (string, []any) {
	return fmt.Sprintf(`
SELECT
    r.client_id,
    COALESCE(SUM(r.total_amount), 0) - COALESCE(SUM(received.total_received), 0) AS outstanding
FROM %s r
LEFT JOIN (
    SELECT tc.revenue_id, SUM(tc.amount) AS total_received
    FROM %s tc
    WHERE tc.active = true AND tc.status IN ('paid', 'completed')
    GROUP BY tc.revenue_id
) received ON received.revenue_id = r.id
WHERE r.active = true
  AND r.status NOT IN ('cancelled', 'draft')
  AND r.client_id IS NOT NULL
  AND ($1::text IS NULL OR r.workspace_id = $1)
GROUP BY r.client_id
HAVING COALESCE(SUM(r.total_amount), 0) - COALESCE(SUM(received.total_received), 0) != 0`,
		tc.Revenue, tc.TreasuryCollection), nil
}

func legacyBuildClientStatementQuery(tc TableConfig, req *clientstmtpb.ClientStatementRequest, workspaceID string) (string, []any) {
	args := []any{
		req.GetClientId(),
		nilIfEmpty(req.GetStartDate()),
		nilIfEmpty(req.GetEndDate()),
		nilIfEmpty(req.GetCurrency()),
		nilIfEmpty(workspaceID),
	}

	query := fmt.Sprintf(`
WITH statement AS (
    SELECT
        TO_CHAR(r.revenue_date::date, 'YYYY-MM-DD') AS date,
        'invoice' AS type,
        r.reference_number,
        r.name AS description,
        r.total_amount::bigint AS billed,
        0::bigint AS received,
        r.id AS entity_id,
        r.status
    FROM %s r
    WHERE r.client_id = $1
        AND r.status != 'cancelled'
        AND r.active = true
        AND ($2::text IS NULL OR r.revenue_date >= $2::date)
        AND ($3::text IS NULL OR r.revenue_date < ($3::date + interval '1 day'))
        AND ($4::text IS NULL OR r.currency = $4)
        AND ($5::text IS NULL OR r.workspace_id = $5)

    UNION ALL

    SELECT
        COALESCE(TO_CHAR(NULLIF(tc.payment_date, '')::date, 'YYYY-MM-DD'), '') AS date,
        'collection' AS type,
        tc.reference_number,
        tc.name AS description,
        0::bigint AS billed,
        tc.amount::bigint AS received,
        tc.id AS entity_id,
        tc.status
    FROM %s tc
    JOIN %s r ON r.id = tc.revenue_id
    WHERE r.client_id = $1
        AND r.status != 'cancelled'
        AND r.active = true
        AND ($2::text IS NULL OR NULLIF(tc.payment_date, '')::date >= $2::date)
        AND ($3::text IS NULL OR NULLIF(tc.payment_date, '')::date < ($3::date + interval '1 day'))
        AND ($4::text IS NULL OR tc.currency = $4)
        AND ($5::text IS NULL OR r.workspace_id = $5)
)
SELECT
    date,
    type,
    COALESCE(reference_number, '') AS reference_number,
    COALESCE(description, '') AS description,
    billed,
    received,
    SUM(billed - received) OVER (ORDER BY date, CASE type WHEN 'invoice' THEN 0 WHEN 'collection' THEN 1 END) AS balance,
    entity_id,
    status
FROM statement
ORDER BY date, CASE type WHEN 'invoice' THEN 0 WHEN 'collection' THEN 1 END`,
		tc.Revenue,
		tc.TreasuryCollection,
		tc.Revenue,
	)

	return query, args
}

func legacyBuildReceivablesAgingQuery(tc TableConfig, req *agingpb.ReceivablesAgingRequest, workspaceID string) (string, []any) {
	// Validate and normalise dimension.
	rowDim := normalizeAgingDimension(req.GetRowDimension())
	if !validAgingDimensions[rowDim] {
		rowDim = "client"
	}

	dimConfig := getAgingDimensionConfig(tc, rowDim)

	// Default as_of_date to today if not provided.
	asOfDate := req.GetAsOfDate()
	if asOfDate == "" {
		asOfDate = time.Now().Format("2006-01-02")
	}

	// Build parameter list.
	// $1 = as_of_date (text, YYYY-MM-DD)
	// $2 = client_id (text or NULL)
	// $3 = location_id (text or NULL)
	// $4 = revenue_category_id (text or NULL)
	// $5 = currency (text or NULL)
	// $6 = start_date (text or NULL) — filter revenues created after
	// $7 = end_date (text or NULL) — filter revenues created before
	// $8 = workspace_id (text or NULL)
	args := []any{
		asOfDate,
		nilIfEmpty(req.GetClientId()),
		nilIfEmpty(req.GetLocationId()),
		nilIfEmpty(req.GetRevenueCategoryId()),
		nilIfEmpty(req.GetCurrency()),
		nilIfEmpty(req.GetStartDate()),
		nilIfEmpty(req.GetEndDate()),
		nilIfEmpty(workspaceID),
	}

	query := fmt.Sprintf(`
WITH outstanding AS (
    SELECT
        r.id AS revenue_id,
        %s,
        %s,
        r.total_amount,
        COALESCE(SUM(tc.amount), 0) AS total_paid,
        r.total_amount - COALESCE(SUM(tc.amount), 0) AS balance,
        CASE
            WHEN r.due_date IS NULL THEN 0
            ELSE (($1::date) - r.due_date::date)
        END AS days_overdue
    FROM %s r
    LEFT JOIN %s tc
        ON tc.revenue_id = r.id
        AND NULLIF(tc.payment_date, '')::date < ($1::date + interval '1 day')
    %s
    WHERE r.status != 'cancelled'
        AND r.active = true
        AND ($2::text IS NULL OR r.client_id = $2)
        AND ($3::text IS NULL OR r.location_id = $3)
        AND ($4::text IS NULL OR r.revenue_category_id = $4)
        AND ($5::text IS NULL OR r.currency = $5)
        AND ($6::text IS NULL OR r.revenue_date >= $6::date)
        AND ($7::text IS NULL OR r.revenue_date < ($7::date + interval '1 day'))
        AND ($8::text IS NULL OR r.workspace_id = $8)
    GROUP BY r.id, r.total_amount, r.due_date, %s
    HAVING r.total_amount - COALESCE(SUM(tc.amount), 0) > 0
)
SELECT
    row_key,
    row_id,
    COALESCE(SUM(balance) FILTER (WHERE days_overdue <= 0), 0)::bigint AS current_amount,
    COALESCE(SUM(balance) FILTER (WHERE days_overdue BETWEEN 1 AND 30), 0)::bigint AS days_1_30,
    COALESCE(SUM(balance) FILTER (WHERE days_overdue BETWEEN 31 AND 60), 0)::bigint AS days_31_60,
    COALESCE(SUM(balance) FILTER (WHERE days_overdue BETWEEN 61 AND 90), 0)::bigint AS days_61_90,
    COALESCE(SUM(balance) FILTER (WHERE days_overdue > 90), 0)::bigint AS days_over_90,
    SUM(balance)::bigint AS total_outstanding,
    COUNT(*)::int AS invoice_count
FROM outstanding
GROUP BY row_key, row_id
ORDER BY row_key`,
		dimConfig.selectKey, dimConfig.selectID,
		tc.Revenue,
		tc.TreasuryCollection,
		dimConfig.extraJoins,
		dimConfig.groupBy,
	)

	return query, args
}

func legacyBuildCollectionSummaryQuery(tc TableConfig, req *collsumpb.CollectionSummaryRequest, workspaceID string) (string, []any) {
	// Validate and normalise dimensions.
	primaryDim := normalizeCollectionDimension(req.GetPrimaryDimension())
	if !validCollectionDimensions[primaryDim] {
		primaryDim = "monthly"
	}
	rowDim := normalizeCollectionDimension(req.GetRowDimension())
	if !validCollectionDimensions[rowDim] {
		rowDim = "client"
	}

	colConfig := getCollectionPivotDimensionConfig(tc, primaryDim)
	rowConfig := getCollectionPivotDimensionConfig(tc, rowDim)

	// Combine extra JOINs from both dimensions, deduplicating shared table aliases.
	extraJoins := mergeJoins(rowConfig.extraJoins, colConfig.extraJoins)

	// Build parameter list.
	// $1 = start_date (text or NULL)
	// $2 = end_date   (text or NULL)
	// $3 = client_id (text or NULL)
	// $4 = location_id (text or NULL)
	// $5 = collection_method_id (text or NULL)
	// $6 = currency (text or NULL)
	// $7 = collection_type (text or NULL)
	// $8 = workspace_id (text or NULL)
	args := []any{
		nilIfEmpty(req.GetStartDate()),
		nilIfEmpty(req.GetEndDate()),
		nilIfEmpty(req.GetClientId()),
		nilIfEmpty(req.GetLocationId()),
		nilIfEmpty(req.GetCollectionMethodId()),
		nilIfEmpty(req.GetCurrency()),
		nilIfEmpty(req.GetCollectionType()),
		nilIfEmpty(workspaceID),
	}

	query := fmt.Sprintf(`
WITH collection_pivot AS (
    SELECT
        %s AS row_key,
        %s AS row_id,
        %s AS col_key,
        %s AS col_id,
        SUM(tc.amount)::bigint   AS total_collected,
        COUNT(tc.id)::bigint     AS transaction_count
    FROM %s tc
    JOIN %s r ON r.id = tc.revenue_id
    %s
    WHERE tc.active = true
      AND ($1::text IS NULL OR NULLIF(tc.payment_date, '')::date >= $1::date)
      AND ($2::text IS NULL OR NULLIF(tc.payment_date, '')::date < ($2::date + interval '1 day'))
      AND ($3::text IS NULL OR r.client_id = $3)
      AND ($4::text IS NULL OR r.location_id = $4)
      AND ($5::text IS NULL OR tc.collection_method_id = $5)
      AND ($6::text IS NULL OR tc.currency = $6)
      AND ($7::text IS NULL OR tc.collection_type = $7)
      AND ($8::text IS NULL OR r.workspace_id = $8)
    GROUP BY %s, %s
)
SELECT row_key, row_id, col_key, col_id,
       total_collected, transaction_count
FROM collection_pivot
ORDER BY row_key, col_key`,
		rowConfig.selectKey, rowConfig.selectID,
		colConfig.selectKey, colConfig.selectID,
		tc.TreasuryCollection,
		tc.Revenue,
		extraJoins,
		rowConfig.groupBy, colConfig.groupBy,
	)

	return query, args
}
