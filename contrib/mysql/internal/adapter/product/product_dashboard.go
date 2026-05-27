//go:build mysql

package product

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/erniealice/espyna-golang/consumer"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	"google.golang.org/protobuf/encoding/protojson"
)

// CountByStatusAndKind returns a map of status (active|inactive) → count for
// products with the given product_kind value (e.g. "service"). Workspace-scoped.
//
// Dialect translation from postgres gold standard:
//   - $1, $2 → ? (MySQL positional placeholders, same left-to-right arg order)
//   - ($1::text IS NULL OR ...) → (? = ” OR ...) — MySQL has no ::text cast;
//     empty-string check replaces the NULL/empty-string guard
//   - CASE WHEN p.active THEN 'active' ELSE 'inactive' END is portable SQL
//   - COUNT(*)::bigint → COUNT(*) — MySQL COUNT() returns BIGINT natively
//
// CRITICAL: Always filters by workspace_id for multi-tenancy.
func (r *MySQLProductRepository) CountByStatusAndKind(
	ctx context.Context,
	workspaceID string,
	kind string,
) (map[string]int64, error) {
	// Fall back to context workspace_id if not provided directly.
	if workspaceID == "" {
		workspaceID = consumer.GetWorkspaceIDFromContext(ctx)
	}

	// Consolidated CASE-based aggregate CTE (A5 dashboard pattern, one round-trip).
	// MySQL has no FILTER (WHERE ...) on aggregates; use SUM(CASE WHEN ...) instead.
	// Dialect: $2, $1 → ? (args: kind, workspaceID) — reordered to match positional ? sequence.
	const query = `
		SELECT
			CASE WHEN p.active = 1 THEN 'active' ELSE 'inactive' END AS status,
			COUNT(*) AS cnt
		FROM product p
		WHERE p.product_kind = ?
		  AND (? = '' OR p.workspace_id = ?)
		GROUP BY status`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, kind, workspaceID, workspaceID)
	if err != nil {
		return map[string]int64{}, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make(map[string]int64, 2)
	for rows.Next() {
		var (
			status string
			n      int64
		)
		if scanErr := rows.Scan(&status, &n); scanErr != nil {
			return nil, fmt.Errorf("failed to scan product status row: %w", scanErr)
		}
		out[status] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product status rows: %w", err)
	}
	return out, nil
}

// CountByLine returns a map of line_id → count, restricted to the given
// product_kind. The link table is product_line (rows mapping product_id →
// line_id); we LEFT JOIN it so products with no line membership still
// surface as 'unassigned'. Workspace-scoped on product.
//
// Dialect translation from postgres gold standard:
//   - $1, $2 → ? (workspace_id first, kind second — same positional order)
//   - ($1::text IS NULL OR ...) → (? = ” OR ...) — MySQL empty-string guard
//   - NULLIF(pl.line_id, ”) stays portable
//   - COUNT(DISTINCT p.id)::bigint → COUNT(DISTINCT p.id)
//   - active = true → active = 1 (MySQL TINYINT(1) boolean)
//
// CRITICAL: Always filters by workspace_id for multi-tenancy.
func (r *MySQLProductRepository) CountByLine(
	ctx context.Context,
	workspaceID string,
	kind string,
) (map[string]int64, error) {
	if workspaceID == "" {
		workspaceID = consumer.GetWorkspaceIDFromContext(ctx)
	}

	const query = `
		SELECT COALESCE(NULLIF(pl.line_id, ''), 'unassigned'),
		       COUNT(DISTINCT p.id)
		FROM product p
		LEFT JOIN product_line pl
		  ON pl.product_id = p.id AND pl.active = 1
		WHERE p.active = 1
		  AND p.product_kind = ?
		  AND (? = '' OR p.workspace_id = ?)
		GROUP BY 1`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, kind, workspaceID, workspaceID)
	if err != nil {
		return map[string]int64{}, nil //nolint:nilerr
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
//
// Dialect translation from postgres gold standard:
//   - $1, $2, $3 → ? (workspace_id first, kind second, limit third)
//   - to_jsonb(p) AS row → explicit column list with JSON_OBJECT or individual
//     columns + standard protojson marshal. We select individual scalar columns
//     to avoid a MySQL-side to_jsonb equivalent; protojson still reconstructs
//     the Product proto from the map round-trip.
//   - ($1::text IS NULL OR ...) → (? = ” OR ...) — MySQL empty-string guard
//   - active = true → active = 1 (MySQL TINYINT(1) boolean)
//   - LIMIT $3 → LIMIT ? (positional)
//
// CRITICAL: Always filters by workspace_id for multi-tenancy.
func (r *MySQLProductRepository) RecentlyListed(
	ctx context.Context,
	workspaceID string,
	kind string,
	limit int32,
) ([]*productpb.Product, error) {
	if workspaceID == "" {
		workspaceID = consumer.GetWorkspaceIDFromContext(ctx)
	}
	if limit <= 0 {
		limit = 5
	}

	// Select the full scalar product row; we marshal it into a map ourselves
	// rather than relying on to_jsonb (postgres-only). The column set covers
	// all fields needed by the Product proto.
	const query = `
		SELECT
			p.id,
			p.name,
			p.description,
			p.active,
			p.price,
			p.product_type,
			p.product_kind,
			p.tracking_mode,
			p.unit_of_measure,
			p.currency,
			p.workspace_id,
			p.date_created,
			p.date_modified
		FROM product p
		WHERE p.active = 1
		  AND p.product_kind = ?
		  AND (? = '' OR p.workspace_id = ?)
		ORDER BY p.date_created DESC
		LIMIT ?`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, kind, workspaceID, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recently listed products: %w", err)
	}
	defer rows.Close()

	out := make([]*productpb.Product, 0, limit)
	for rows.Next() {
		var (
			id            string
			name          *string
			description   *string
			active        bool
			price         *int64
			productType   *string
			productKind   *string
			trackingMode  *string
			unitOfMeasure *string
			currency      *string
			wsID          *string
			dateCreated   time.Time
			dateModified  time.Time
		)

		if scanErr := rows.Scan(
			&id,
			&name,
			&description,
			&active,
			&price,
			&productType,
			&productKind,
			&trackingMode,
			&unitOfMeasure,
			&currency,
			&wsID,
			&dateCreated,
			&dateModified,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan recently listed product row: %w", scanErr)
		}

		// Build a map and round-trip via protojson — same approach as the postgres
		// gold standard's to_jsonb path, but without a postgres dependency.
		rowMap := map[string]any{
			"id":     id,
			"active": active,
		}
		if name != nil {
			rowMap["name"] = *name
		}
		if description != nil {
			rowMap["description"] = *description
		}
		if price != nil {
			rowMap["price"] = *price
		}
		if productType != nil {
			rowMap["productType"] = *productType
		}
		if productKind != nil {
			rowMap["productKind"] = *productKind
		}
		if trackingMode != nil {
			rowMap["trackingMode"] = *trackingMode
		}
		if unitOfMeasure != nil {
			rowMap["unitOfMeasure"] = *unitOfMeasure
		}
		if currency != nil {
			rowMap["currency"] = *currency
		}
		if wsID != nil {
			rowMap["workspaceId"] = *wsID
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			rowMap["dateCreated"] = ts
			s := dateCreated.Format(time.RFC3339)
			rowMap["dateCreatedString"] = s
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			rowMap["dateModified"] = ts
			s := dateModified.Format(time.RFC3339)
			rowMap["dateModifiedString"] = s
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
