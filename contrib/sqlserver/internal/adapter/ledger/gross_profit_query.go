//go:build sqlserver

package ledger

import (
	"fmt"
	"strings"
	"time"

	reportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/gross_profit"
)

// groupByConfig defines the SQL fragments for each grouping dimension.
type groupByConfig struct {
	revenueSelect     string
	revenueGroupBy    string
	cogsSelect        string
	cogsGroupBy       string
	joinCondition     string
	extraRevenueJoins string
}

// validGroupBy is a whitelist of allowed group_by values to prevent SQL injection.
var validGroupBy = map[string]bool{
	"product":  true,
	"location": true,
	"category": true,
	"period":   true,
}

// validGranularity maps user-facing granularity to SQL Server DATETRUNC-compatible intervals.
var validGranularity = map[string]string{
	"daily":   "day",
	"weekly":  "week",
	"monthly": "month",
	"yearly":  "year",
}

// buildGrossProfitQuery constructs the CTE-based SQL query and its parameter args.
// Table names come from TableConfig (developer-configured, safe for fmt.Sprintf).
// All user-provided values use parameterized queries (@p1, @p2, ...).
//
// SQL Server differences from the postgres gold standard:
//   - $N → @pN placeholders.
//   - ::timestamptz IS NULL → @pN IS NULL (SQL Server does not use postgres cast syntax).
//   - date_trunc('month', col) → FORMAT(col, 'yyyy-MM-01') or DATEFROMPARTS for period grouping.
//   - ::bigint, ::text → CAST(… AS bigint), CAST(… AS nvarchar).
//   - FILTER (WHERE c) aggregate → SUM(CASE WHEN c THEN x END).
func buildGrossProfitQuery(tc TableConfig, req *reportpb.GrossProfitReportRequest, workspaceID string) (string, []any) {
	groupBy := "product"
	if req.GroupBy != nil && validGroupBy[req.GetGroupBy()] {
		groupBy = req.GetGroupBy()
	}

	cfg := getGroupByConfig(tc, groupBy, req.GetPeriodGranularity())

	// Build parameter list (same order/semantics as postgres gold standard).
	// @p1 = start_date (datetime2 or NULL)
	// @p2 = end_date   (datetime2 or NULL)
	// @p3 = product_id (text or NULL)
	// @p4 = location_id (text or NULL)
	// @p5 = revenue_category_id (text or NULL)
	// @p6 = workspace_id (text or NULL)
	args := make([]any, 6)
	if req.StartDate != nil {
		if t, err := time.Parse("2006-01-02", req.GetStartDate()); err == nil {
			args[0] = t.UTC()
		}
	}
	if req.EndDate != nil {
		if t, err := time.Parse("2006-01-02", req.GetEndDate()); err == nil {
			args[1] = t.UTC()
		}
	}
	if req.ProductId != nil {
		args[2] = req.GetProductId()
	}
	if req.LocationId != nil {
		args[3] = req.GetLocationId()
	}
	if req.RevenueCategoryId != nil {
		args[4] = req.GetRevenueCategoryId()
	}
	if workspaceID != "" {
		args[5] = workspaceID
	}

	var sb strings.Builder

	// Revenue summary CTE.
	// Note: discount tracking at line level is a future migration; total_discount is 0.
	sb.WriteString(fmt.Sprintf(`WITH revenue_summary AS (
    SELECT
        %s,
        CAST(SUM(rli.total_price) AS bigint) AS total_revenue,
        CAST(0 AS bigint) AS total_discount,
        CAST(SUM(rli.total_price) AS bigint) AS net_revenue,
        CAST(SUM(rli.quantity) AS bigint) AS units_sold,
        COUNT(DISTINCT r.id) AS transaction_count
    FROM %s rli
    JOIN %s r ON r.id = rli.revenue_id
    LEFT JOIN %s p ON p.id = rli.product_id
    %s
    WHERE r.active = 1
      AND r.status != 'cancelled'
      AND (@p1 IS NULL OR r.revenue_date >= @p1)
      AND (@p2 IS NULL OR r.revenue_date <= @p2)
      AND (@p3 IS NULL OR rli.product_id = @p3)
      AND (@p4 IS NULL OR r.location_id = @p4)
      AND (@p5 IS NULL OR r.revenue_category_id = @p5)
      AND (@p6 IS NULL OR r.workspace_id = @p6)
    GROUP BY %s
)`,
		cfg.revenueSelect,
		tc.RevenueLineItem,
		tc.Revenue,
		tc.Product,
		cfg.extraRevenueJoins,
		cfg.revenueGroupBy,
	))

	// COGS summary CTE.
	sb.WriteString(fmt.Sprintf(`,
cogs_summary AS (
    SELECT
        %s,
        SUM(ABS(it.quantity) * it.unit_cost) AS cost_of_goods_sold
    FROM %s it
    JOIN %s ii ON ii.id = it.inventory_item_id
    WHERE it.active = 1
      AND it.transaction_type IN ('sold', 'OUT')
      AND it.unit_cost > 0
      AND (@p1 IS NULL OR it.transaction_date >= @p1)
      AND (@p2 IS NULL OR it.transaction_date <= @p2)
      AND (@p3 IS NULL OR ii.product_id = @p3)
      AND (@p4 IS NULL OR ii.location_id = @p4)
    GROUP BY %s
)`,
		cfg.cogsSelect,
		tc.InventoryTransaction,
		tc.InventoryItem,
		cfg.cogsGroupBy,
	))

	// Final SELECT joining both CTEs.
	sb.WriteString(fmt.Sprintf(`
SELECT
    rs.group_key,
    rs.group_id,
    rs.total_revenue,
    rs.total_discount,
    rs.net_revenue,
    COALESCE(cs.cost_of_goods_sold, 0) AS cost_of_goods_sold,
    rs.net_revenue - COALESCE(cs.cost_of_goods_sold, 0) AS gross_profit,
    CASE WHEN rs.net_revenue > 0
         THEN ((rs.net_revenue - COALESCE(cs.cost_of_goods_sold, 0)) * 1.0 / rs.net_revenue) * 100
         ELSE 0
    END AS gross_profit_margin,
    rs.units_sold,
    rs.transaction_count
FROM revenue_summary rs
LEFT JOIN cogs_summary cs ON %s
ORDER BY rs.net_revenue DESC`,
		cfg.joinCondition,
	))

	return sb.String(), args
}

// getGroupByConfig returns the SQL fragments for the requested grouping dimension.
// SQL Server period expressions use FORMAT() for YYYYMM bucketing.
func getGroupByConfig(tc TableConfig, groupBy string, granularity string) groupByConfig {
	switch groupBy {
	case "location":
		return groupByConfig{
			revenueSelect:     "l.name AS group_key, r.location_id AS group_id",
			revenueGroupBy:    "r.location_id, l.name",
			cogsSelect:        "ii.location_id AS group_id",
			cogsGroupBy:       "ii.location_id",
			joinCondition:     "cs.group_id = rs.group_id",
			extraRevenueJoins: fmt.Sprintf("LEFT JOIN %s l ON l.id = r.location_id", tc.Location),
		}
	case "category":
		return groupByConfig{
			revenueSelect:     "rc.name AS group_key, r.revenue_category_id AS group_id",
			revenueGroupBy:    "r.revenue_category_id, rc.name",
			cogsSelect:        "ii.product_id AS group_id",
			cogsGroupBy:       "ii.product_id",
			joinCondition:     "cs.group_id = rs.group_id",
			extraRevenueJoins: fmt.Sprintf("LEFT JOIN %s rc ON rc.id = r.revenue_category_id", tc.RevenueCategory),
		}
	case "period":
		interval := "month"
		if g, ok := validGranularity[granularity]; ok {
			interval = g
		}
		// SQL Server 2017+ uses DATEADD/DATEDIFF to truncate to a period boundary.
		// For month: DATEFROMPARTS(YEAR(col), MONTH(col), 1)
		// For year:  DATEFROMPARTS(YEAR(col), 1, 1)
		// For day:   CAST(col AS date)
		// For week:  DATEADD(day, -(DATEPART(weekday, col) - 1), CAST(col AS date))
		var dateTrunc, cogsDateTrunc string
		switch interval {
		case "day":
			dateTrunc = "CAST(r.revenue_date AS date)"
			cogsDateTrunc = "CAST(it.transaction_date AS date)"
		case "week":
			dateTrunc = "DATEADD(day, -(DATEPART(weekday, r.revenue_date) - 1), CAST(r.revenue_date AS date))"
			cogsDateTrunc = "DATEADD(day, -(DATEPART(weekday, it.transaction_date) - 1), CAST(it.transaction_date AS date))"
		case "year":
			dateTrunc = "DATEFROMPARTS(YEAR(r.revenue_date), 1, 1)"
			cogsDateTrunc = "DATEFROMPARTS(YEAR(it.transaction_date), 1, 1)"
		default: // month
			dateTrunc = "DATEFROMPARTS(YEAR(r.revenue_date), MONTH(r.revenue_date), 1)"
			cogsDateTrunc = "DATEFROMPARTS(YEAR(it.transaction_date), MONTH(it.transaction_date), 1)"
		}
		return groupByConfig{
			revenueSelect:  fmt.Sprintf("CAST(%s AS nvarchar(30)) AS group_key, CAST(%s AS nvarchar(30)) AS group_id", dateTrunc, dateTrunc),
			revenueGroupBy: dateTrunc,
			cogsSelect:     fmt.Sprintf("CAST(%s AS nvarchar(30)) AS group_id", cogsDateTrunc),
			cogsGroupBy:    cogsDateTrunc,
			joinCondition:  "cs.group_id = rs.group_id",
		}
	default: // "product"
		return groupByConfig{
			revenueSelect:  "p.name AS group_key, rli.product_id AS group_id",
			revenueGroupBy: "rli.product_id, p.name",
			cogsSelect:     "ii.product_id AS group_id",
			cogsGroupBy:    "ii.product_id",
			joinCondition:  "cs.group_id = rs.group_id",
		}
	}
}
