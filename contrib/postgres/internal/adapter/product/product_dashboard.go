//go:build postgresql

package product

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// CountByStatusAndKind returns a map of status (active|inactive) → count for
// products with the given product_kind value (e.g. "service"). Workspace-
// scoped.
//
// Q-DASHBOARD-FAILOPEN (Option A): the active/inactive split is the dashboard's
// single fixed-shape scalar metric, so it is consolidated into ONE single-row
// CTE — a `base` pass over the kind-filtered, workspace-scoped product rows,
// then `COUNT(*) FILTER (WHERE active)` / `FILTER (WHERE NOT active)` over that
// pass — and scanned through core.RunDashboardAggregate so a DB fault is
// returned honestly instead of being swallowed as an empty all-zeros map.
//
// Performance index recommendation:
//
//	CREATE INDEX idx_product_workspace_kind_active
//	  ON product(workspace_id, product_kind, active);
func (r *PostgresProductRepository) CountByStatusAndKind(
	ctx context.Context,
	workspaceID string,
	kind string,
) (map[string]int64, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	const query = `
		WITH base AS (
			SELECT p.active
			FROM product p
			WHERE p.product_kind = $2
			  AND ($1::text IS NULL OR $1::text = '' OR p.workspace_id = $1)
		)
		SELECT
			COUNT(*) FILTER (WHERE active)::bigint     AS active_count,
			COUNT(*) FILTER (WHERE NOT active)::bigint AS inactive_count
		FROM base`

	var activeCount, inactiveCount int64
	if err := postgresCore.RunDashboardAggregate(
		ctx,
		r.db,
		query,
		[]any{workspaceID, kind},
		&activeCount,
		&inactiveCount,
	); err != nil {
		return nil, err
	}

	return map[string]int64{
		"active":   activeCount,
		"inactive": inactiveCount,
	}, nil
}

// CountByLine returns a map of line_id → count, restricted to the given
// product_kind. The link table is product_line (rows mapping product_id →
// line_id); we LEFT JOIN it so products with no line membership still
// surface as 'unassigned'. Workspace-scoped on product.
func (r *PostgresProductRepository) CountByLine(
	ctx context.Context,
	workspaceID string,
	kind string,
) (map[string]int64, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT COALESCE(NULLIF(pl.line_id, ''), 'unassigned'),
		       COUNT(DISTINCT p.id)::bigint
		FROM product p
		LEFT JOIN product_line pl
		  ON pl.product_id = p.id AND pl.active = true
		WHERE p.active = true
		  AND p.product_kind = $2
		  AND ($1::text IS NULL OR $1::text = '' OR p.workspace_id = $1)
		GROUP BY 1`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, kind)
	if err != nil {
		return nil, fmt.Errorf("failed to query product-by-line counts: %w", err)
	}
	defer rows.Close()

	out := make(map[string]int64, 8)
	for rows.Next() {
		var (
			lineID string
			n      int64
		)
		if scanErr := rows.Scan(&lineID, &n); scanErr != nil {
			return nil, fmt.Errorf("failed to scan product-by-line row: %w", scanErr)
		}
		out[lineID] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product-by-line rows: %w", err)
	}
	return out, nil
}

// RecentlyListed returns the most recently created products of the given
// kind (newest-first). Workspace-scoped.
func (r *PostgresProductRepository) RecentlyListed(
	ctx context.Context,
	workspaceID string,
	kind string,
	limit int32,
) ([]*productpb.Product, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if limit <= 0 {
		limit = 5
	}

	const query = `
		SELECT to_jsonb(p) AS row
		FROM product p
		WHERE p.active = true
		  AND p.product_kind = $2
		  AND ($1::text IS NULL OR $1::text = '' OR p.workspace_id = $1)
		ORDER BY p.date_created DESC
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, kind, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recently listed products: %w", err)
	}
	defer rows.Close()

	out := make([]*productpb.Product, 0, limit)
	for rows.Next() {
		var rowJSON []byte
		if scanErr := rows.Scan(&rowJSON); scanErr != nil {
			return nil, fmt.Errorf("failed to scan recently listed product row: %w", scanErr)
		}
		var rowMap map[string]any
		if err := json.Unmarshal(rowJSON, &rowMap); err != nil {
			log.Printf("WARN: unmarshal recent product row: %v", err)
			continue
		}
		clean, err := json.Marshal(rowMap)
		if err != nil {
			log.Printf("WARN: re-marshal recent product row: %v", err)
			continue
		}
		p := &productpb.Product{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(clean, p); err != nil {
			log.Printf("WARN: protojson unmarshal recent product: %v", err)
			continue
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recently listed products: %w", err)
	}
	return out, nil
}
