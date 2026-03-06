
package reference

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

// Checker provides batch FK reference checking for deletable state.
// Each method returns a map where true = ID is in use and should NOT be deleted.
type Checker struct {
	db *sql.DB
}

func NewChecker(db *sql.DB) *Checker {
	return &Checker{db: db}
}

func (c *Checker) GetLocationInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `
		SELECT DISTINCT ref_id FROM (
			SELECT location_id AS ref_id FROM revenue WHERE location_id = ANY($1) AND active = true
			UNION ALL
			SELECT location_id AS ref_id FROM expenditure WHERE location_id = ANY($1) AND active = true
			UNION ALL
			SELECT location_id AS ref_id FROM inventory_item WHERE location_id = ANY($1) AND active = true
			UNION ALL
			SELECT location_id AS ref_id FROM price_list WHERE location_id = ANY($1) AND active = true
		) AS refs`
	return queryInUseIDs(ctx, c.db, query, ids)
}

func (c *Checker) GetRoleInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `SELECT DISTINCT role_id FROM workspace_user_role WHERE role_id = ANY($1) AND active = true`
	return queryInUseIDs(ctx, c.db, query, ids)
}

func (c *Checker) GetCategoryInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `SELECT DISTINCT category_id FROM client_category WHERE category_id = ANY($1) AND active = true`
	return queryInUseIDs(ctx, c.db, query, ids)
}

// GetClientInUseIDs checks if clients are referenced as vendors in expenditure.
func (c *Checker) GetClientInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `SELECT DISTINCT vendor_id FROM expenditure WHERE vendor_id = ANY($1) AND active = true`
	return queryInUseIDs(ctx, c.db, query, ids)
}

func (c *Checker) GetProductInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `
		SELECT DISTINCT ref_id FROM (
			SELECT product_id AS ref_id FROM revenue_line_item WHERE product_id = ANY($1) AND active = true
			UNION ALL
			SELECT product_id AS ref_id FROM price_product WHERE product_id = ANY($1) AND active = true
			UNION ALL
			SELECT product_id AS ref_id FROM inventory_item WHERE product_id = ANY($1) AND active = true
		) AS refs`
	return queryInUseIDs(ctx, c.db, query, ids)
}

// GetPriceListInUseIDs checks if price lists are referenced by price products.
func (c *Checker) GetPriceListInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `SELECT DISTINCT price_list_id FROM price_product WHERE price_list_id = ANY($1) AND active = true`
	return queryInUseIDs(ctx, c.db, query, ids)
}

func (c *Checker) GetAssetCategoryInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `SELECT DISTINCT asset_category_id AS ref_id FROM asset WHERE asset_category_id = ANY($1) AND active = true`
	return queryInUseIDs(ctx, c.db, query, ids)
}

func queryInUseIDs(ctx context.Context, db *sql.DB, query string, ids []string) (map[string]bool, error) {
	if len(ids) == 0 {
		return make(map[string]bool), nil
	}

	rows, err := db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("reference check query failed: %w", err)
	}
	defer rows.Close()

	result := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("reference check scan failed: %w", err)
		}
		result[id] = true
	}
	return result, rows.Err()
}
