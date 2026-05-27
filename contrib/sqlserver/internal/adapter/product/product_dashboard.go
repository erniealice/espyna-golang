//go:build sqlserver

package product

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// CountByStatusAndKind returns a map of status (active|inactive) → count for
// products with the given product_kind value. Workspace-scoped.
//
// SQL Server: FILTER(WHERE ...) → SUM(CASE WHEN ... THEN 1 ELSE 0 END);
// active = 1 (BIT column); @pN positional params.
func (r *SQLServerProductRepository) CountByStatusAndKind(
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
			WHERE p.product_kind = @p2
			  AND (@p1 IS NULL OR @p1 = '' OR p.workspace_id = @p1)
		)
		SELECT
			SUM(CASE WHEN active = 1 THEN 1 ELSE 0 END) AS active_count,
			SUM(CASE WHEN active = 0 THEN 1 ELSE 0 END) AS inactive_count
		FROM base`

	var activeCount, inactiveCount int64
	row := r.db.QueryRowContext(ctx, query, workspaceID, kind)
	if err := row.Scan(&activeCount, &inactiveCount); err != nil {
		return nil, fmt.Errorf("failed to scan product count row: %w", err)
	}

	return map[string]int64{
		"active":   activeCount,
		"inactive": inactiveCount,
	}, nil
}

// CountByLine returns a map of line_id → count, restricted to the given
// product_kind. Workspace-scoped on product.
func (r *SQLServerProductRepository) CountByLine(
	ctx context.Context,
	workspaceID string,
	kind string,
) (map[string]int64, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT COALESCE(NULLIF(pl.line_id, ''), 'unassigned'),
		       CAST(COUNT(DISTINCT p.id) AS bigint)
		FROM product p
		LEFT JOIN product_line pl
		  ON pl.product_id = p.id AND pl.active = 1
		WHERE p.active = 1
		  AND p.product_kind = @p2
		  AND (@p1 IS NULL OR @p1 = '' OR p.workspace_id = @p1)
		GROUP BY pl.line_id`

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
//
// SQL Server: OFFSET 0 ROWS FETCH NEXT @p3 ROWS ONLY (instead of LIMIT $3);
// active = 1; individual column SELECT (instead of to_jsonb(p)).
func (r *SQLServerProductRepository) RecentlyListed(
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
		SELECT
			p.id,
			p.name,
			p.description,
			p.price,
			p.product_type,
			p.product_kind,
			p.tracking_mode,
			p.unit_of_measure,
			p.workspace_id,
			p.active,
			p.date_created,
			p.date_modified
		FROM product p
		WHERE p.active = 1
		  AND p.product_kind = @p2
		  AND (@p1 IS NULL OR @p1 = '' OR p.workspace_id = @p1)
		ORDER BY p.date_created DESC
		OFFSET 0 ROWS FETCH NEXT @p3 ROWS ONLY`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, kind, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recently listed products: %w", err)
	}
	defer rows.Close()

	out := make([]*productpb.Product, 0, limit)
	for rows.Next() {
		var (
			id             string
			name           string
			description    *string
			price          *int64
			productType    *string
			productKind    *string
			trackingMode   *string
			unitOfMeasure  *string
			workspaceIDVal *string
			active         bool
			dateCreated    *time.Time
			dateModified   *time.Time
		)
		if scanErr := rows.Scan(
			&id, &name, &description, &price, &productType, &productKind,
			&trackingMode, &unitOfMeasure, &workspaceIDVal, &active,
			&dateCreated, &dateModified,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan recently listed product row: %w", scanErr)
		}

		// Build a map and round-trip through protojson to leverage DiscardUnknown.
		rowMap := map[string]any{
			"id":     id,
			"name":   name,
			"active": active,
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
		if workspaceIDVal != nil {
			rowMap["workspaceId"] = *workspaceIDVal
		}
		if dateCreated != nil && !dateCreated.IsZero() {
			ms := dateCreated.UnixMilli()
			rowMap["dateCreated"] = ms
		}
		if dateModified != nil && !dateModified.IsZero() {
			ms := dateModified.UnixMilli()
			rowMap["dateModified"] = ms
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
