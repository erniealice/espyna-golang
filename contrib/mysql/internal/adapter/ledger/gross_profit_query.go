//go:build mysql

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

// validGranularity maps user-facing granularity to MySQL DATE_FORMAT intervals.
var validGranularity = map[string]string{
	"daily":   "day",
	"weekly":  "week",
	"monthly": "month",
	"yearly":  "year",
}

// buildGrossProfitQuery constructs the CTE-based SQL query and its parameter args.
// Table names come from TableConfig (developer-configured, safe for fmt.Sprintf).
// All user-provided values use parameterized queries (?).
//
// Dialect differences from postgres gold standard:
//   - $N → ? (positional, same left-to-right order)
//   - $N::timestamptz IS NULL → ? IS NULL
//   - date_trunc('month', ...) → DATE_FORMAT(..., '%Y-%m-01') for period grouping
//   - ::bigint casts removed
//   - active = true → active = 1
func buildGrossProfitQuery(tc TableConfig, req *reportpb.GrossProfitReportRequest, workspaceID string) (string, []any) {
	groupBy := "product"
	if req.GroupBy != nil && validGroupBy[req.GetGroupBy()] {
		groupBy = req.GetGroupBy()
	}

	cfg := getGroupByConfig(tc, groupBy, req.GetPeriodGranularity())

	// Args in same order as postgres ($1=start, $2=end, $3=product, $4=location, $5=cat, $6=workspace).
	// MySQL: each ? IS NULL OR col = ? pair is expanded below.
	var start, end, productID, locationID, categoryID, wsID any
	if req.StartDate != nil {
		if t, err := time.Parse("2006-01-02", req.GetStartDate()); err == nil {
			start = t.UTC()
		}
	}
	if req.EndDate != nil {
		if t, err := time.Parse("2006-01-02", req.GetEndDate()); err == nil {
			end = t.UTC()
		}
	}
	if req.ProductId != nil {
		productID = req.GetProductId()
	}
	if req.LocationId != nil {
		locationID = req.GetLocationId()
	}
	if req.RevenueCategoryId != nil {
		categoryID = req.GetRevenueCategoryId()
	}
	if workspaceID != "" {
		wsID = workspaceID
	}

	var sb strings.Builder

	// Revenue summary CTE — no discount column yet (same as postgres TODO comment).
	sb.WriteString(fmt.Sprintf(`WITH revenue_summary AS (
    SELECT
        %s,
        SUM(rli.total_price) AS total_revenue,
        0 AS total_discount,
        SUM(rli.total_price) AS net_revenue,
        SUM(rli.quantity) AS units_sold,
        COUNT(DISTINCT r.id) AS transaction_count
    FROM %s rli
    JOIN %s r ON r.id = rli.revenue_id
    LEFT JOIN %s p ON p.id = rli.product_id
    %s
    WHERE r.active = 1
      AND r.status != 'cancelled'
      AND (? IS NULL OR r.revenue_date >= ?)
      AND (? IS NULL OR r.revenue_date <= ?)
      AND (? IS NULL OR rli.product_id = ?)
      AND (? IS NULL OR r.location_id = ?)
      AND (? IS NULL OR r.revenue_category_id = ?)
      AND (? IS NULL OR r.workspace_id = ?)
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
      AND (? IS NULL OR it.transaction_date >= ?)
      AND (? IS NULL OR it.transaction_date <= ?)
      AND (? IS NULL OR ii.product_id = ?)
      AND (? IS NULL OR ii.location_id = ?)
    GROUP BY %s
)`,
		cfg.cogsSelect,
		tc.InventoryTransaction,
		tc.InventoryItem,
		cfg.cogsGroupBy,
	))

	// Final SELECT.
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
         THEN ((rs.net_revenue - COALESCE(cs.cost_of_goods_sold, 0)) / rs.net_revenue) * 100
         ELSE 0
    END AS gross_profit_margin,
    rs.units_sold,
    rs.transaction_count
FROM revenue_summary rs
LEFT JOIN cogs_summary cs ON %s
ORDER BY rs.net_revenue DESC`,
		cfg.joinCondition,
	))

	// Expand each (? IS NULL OR col = ?) pair once per arg.
	args := []any{
		start, start,
		end, end,
		productID, productID,
		locationID, locationID,
		categoryID, categoryID,
		wsID, wsID,
		// COGS WHERE
		start, start,
		end, end,
		productID, productID,
		locationID, locationID,
	}

	return sb.String(), args
}

// getGroupByConfig returns the SQL fragments for the requested grouping dimension.
// MySQL dialect: date_trunc → DATE_FORMAT; EXTRACT → YEAR()/MONTH()/QUARTER();
// ::text casts removed.
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
		// MySQL date truncation: use DATE_FORMAT to snap to period start.
		interval := "month"
		if g, ok := validGranularity[granularity]; ok {
			interval = g
		}
		var dateFmt string
		switch interval {
		case "day":
			dateFmt = "%Y-%m-%d"
		case "week":
			dateFmt = "%Y-%u" // ISO week
		case "year":
			dateFmt = "%Y"
		default: // month
			dateFmt = "%Y-%m"
		}
		revExpr := fmt.Sprintf("DATE_FORMAT(r.revenue_date, '%s')", dateFmt)
		cogsExpr := fmt.Sprintf("DATE_FORMAT(it.transaction_date, '%s')", dateFmt)
		return groupByConfig{
			revenueSelect:  fmt.Sprintf("%s AS group_key, %s AS group_id", revExpr, revExpr),
			revenueGroupBy: revExpr,
			cogsSelect:     fmt.Sprintf("%s AS group_id", cogsExpr),
			cogsGroupBy:    cogsExpr,
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
