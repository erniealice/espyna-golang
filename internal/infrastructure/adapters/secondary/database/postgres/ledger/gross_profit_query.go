//go:build postgres

package ledger

import (
	"fmt"
	"strings"
	"time"

	reportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/gross_profit"
)

// groupByConfig defines the SQL fragments for each grouping dimension.
type groupByConfig struct {
	// revenueSelect is the SELECT expression for the group key/id in the revenue CTE.
	revenueSelect string
	// revenueGroupBy is the GROUP BY clause for the revenue CTE.
	revenueGroupBy string
	// cogsSelect is the SELECT expression for the grouping key in the COGS CTE.
	cogsSelect string
	// cogsGroupBy is the GROUP BY clause for the COGS CTE.
	cogsGroupBy string
	// joinCondition is the JOIN condition between revenue_summary and cogs_summary.
	joinCondition string
	// extraJoins are additional JOINs needed in the revenue CTE (e.g., location).
	extraRevenueJoins string
}

// validGroupBy is a whitelist of allowed group_by values to prevent SQL injection.
var validGroupBy = map[string]bool{
	"product":  true,
	"location": true,
	"category": true,
	"period":   true,
}

// validGranularity maps user-facing granularity to PostgreSQL date_trunc intervals.
var validGranularity = map[string]string{
	"daily":   "day",
	"weekly":  "week",
	"monthly": "month",
	"yearly":  "year",
}

// buildGrossProfitQuery constructs the CTE-based SQL query and its parameter args.
// Table names come from TableConfig (developer-configured, safe for fmt.Sprintf).
// All user-provided values use parameterized queries ($1, $2, ...).
func buildGrossProfitQuery(tc TableConfig, req *reportpb.GrossProfitReportRequest) (string, []any) {
	groupBy := "product"
	if req.GroupBy != nil && validGroupBy[req.GetGroupBy()] {
		groupBy = req.GetGroupBy()
	}

	cfg := getGroupByConfig(tc, groupBy, req.GetPeriodGranularity())

	// Build parameter list.
	// $1 = start_date (timestamptz or NULL)
	// $2 = end_date (timestamptz or NULL)
	// $3 = product_id (text or NULL)
	// $4 = location_id (text or NULL)
	// $5 = revenue_category_id (text or NULL)
	args := make([]any, 5)
	if req.StartDate != nil {
		args[0] = time.Unix(req.GetStartDate(), 0).UTC()
	}
	if req.EndDate != nil {
		args[1] = time.Unix(req.GetEndDate(), 0).UTC()
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

	var sb strings.Builder

	// Revenue summary CTE
	sb.WriteString(fmt.Sprintf(`WITH revenue_summary AS (
    SELECT
        %s,
        SUM(rli.total_price) AS total_revenue,
        SUM(COALESCE(rli.discount_amount, 0)) AS total_discount,
        SUM(rli.total_price) - SUM(COALESCE(rli.discount_amount, 0)) AS net_revenue,
        SUM(rli.quantity) AS units_sold,
        COUNT(DISTINCT r.id) AS transaction_count
    FROM %s rli
    JOIN %s r ON r.id = rli.revenue_id
    LEFT JOIN %s p ON p.id = rli.product_id
    %s
    WHERE r.active = true
      AND r.status = 'completed'
      AND ($1::timestamptz IS NULL OR r.revenue_date >= $1::timestamptz)
      AND ($2::timestamptz IS NULL OR r.revenue_date <= $2::timestamptz)
      AND ($3::text IS NULL OR rli.product_id = $3)
      AND ($4::text IS NULL OR r.location_id = $4)
      AND ($5::text IS NULL OR r.revenue_category_id = $5)
    GROUP BY %s
)`,
		cfg.revenueSelect,
		tc.RevenueLineItem,
		tc.Revenue,
		tc.Product,
		cfg.extraRevenueJoins,
		cfg.revenueGroupBy,
	))

	// COGS summary CTE
	sb.WriteString(fmt.Sprintf(`,
cogs_summary AS (
    SELECT
        %s,
        SUM(ABS(it.quantity) * it.unit_cost) AS cost_of_goods_sold
    FROM %s it
    JOIN %s ii ON ii.id = it.inventory_item_id
    WHERE it.active = true
      AND it.transaction_type IN ('sold', 'OUT')
      AND it.unit_cost > 0
      AND ($1::timestamptz IS NULL OR it.transaction_date::timestamptz >= $1::timestamptz)
      AND ($2::timestamptz IS NULL OR it.transaction_date::timestamptz <= $2::timestamptz)
      AND ($3::text IS NULL OR ii.product_id = $3)
      AND ($4::text IS NULL OR ii.location_id = $4)
    GROUP BY %s
)`,
		cfg.cogsSelect,
		tc.InventoryTransaction,
		tc.InventoryItem,
		cfg.cogsGroupBy,
	))

	// Final SELECT joining both CTEs
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

	return sb.String(), args
}

// getGroupByConfig returns the SQL fragments for the requested grouping dimension.
func getGroupByConfig(tc TableConfig, groupBy string, granularity string) groupByConfig {
	switch groupBy {
	case "location":
		return groupByConfig{
			revenueSelect:     fmt.Sprintf("l.name AS group_key, r.location_id AS group_id"),
			revenueGroupBy:    "r.location_id, l.name",
			cogsSelect:        "ii.location_id AS group_id",
			cogsGroupBy:       "ii.location_id",
			joinCondition:     "cs.group_id = rs.group_id",
			extraRevenueJoins: fmt.Sprintf("LEFT JOIN %s l ON l.id = r.location_id", tc.Location),
		}
	case "category":
		return groupByConfig{
			revenueSelect:     fmt.Sprintf("rc.name AS group_key, r.revenue_category_id AS group_id"),
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
		dateTrunc := fmt.Sprintf("date_trunc('%s', r.revenue_date)", interval)
		cogsDateTrunc := fmt.Sprintf("date_trunc('%s', it.transaction_date::timestamptz)", interval)
		return groupByConfig{
			revenueSelect:  fmt.Sprintf("%s::text AS group_key, %s::text AS group_id", dateTrunc, dateTrunc),
			revenueGroupBy: dateTrunc,
			cogsSelect:     fmt.Sprintf("%s::text AS group_id", cogsDateTrunc),
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
