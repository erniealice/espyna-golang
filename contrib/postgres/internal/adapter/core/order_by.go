//go:build postgresql

package core

import (
	"fmt"
	"strings"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// BuildOrderBy is the canonical sort-whitelist helper for postgres list-page
// adapters. It is the fail-closed guard against unguarded `ORDER BY %s %s`
// interpolation (the A2 canonical bug): every page-data method that lets the
// caller pick a sort column must route the column through this helper rather
// than trusting the request field verbatim.
//
// allowedCols is the per-entity whitelist (e.g. supplierSortableSQLCols). It
// composes with the same slice that drives espynahttp.SortSpec.AllowedCols /
// espynahttp.ValidateSortColumns, so the view layer and the adapter share one
// source of truth.
//
// Behavior:
//   - sort nil/empty (no fields, or first field blank) → use fallback verbatim.
//   - requested field NOT in allowedCols → return an error (fail closed). The
//     caller must propagate the error; it must NOT silently fall back, because
//     a silent fallback hides a probing/misconfigured caller.
//   - requested direction is validated to ASC/DESC only (defaults to ASC for
//     SortDirection_ASC, DESC for SortDirection_DESC).
//
// The returned fragment is `ORDER BY <col> <DIR>` with the column safely
// double-quoted. Because the column is whitelist-validated AND quoted, and the
// direction is enum-derived, the fragment is safe to interpolate into a query
// string with fmt.Sprintf.
//
// fallback is the default ORDER BY body when no sort is requested, e.g.
// "date_created DESC". It is interpolated verbatim (it is author-controlled,
// never caller-controlled) so authors must keep it a trusted constant.
func BuildOrderBy(allowedCols []string, sort *commonpb.SortRequest, fallback string) (string, error) {
	field, dir, ok := firstSortField(sort)
	if !ok {
		return "ORDER BY " + fallback, nil
	}

	if !sortColAllowed(field, allowedCols) {
		return "", fmt.Errorf("unknown sort column %q (allowed: %v)", field, allowedCols)
	}

	return fmt.Sprintf("ORDER BY %s %s", quoteSortIdent(field), dir), nil
}

// firstSortField extracts the first non-empty sort field + normalized direction
// from the request. ok=false means "no usable sort field" (nil request, no
// fields, or a blank field) and the caller should use its fallback.
func firstSortField(sort *commonpb.SortRequest) (field, dir string, ok bool) {
	if sort == nil {
		return "", "", false
	}
	for _, f := range sort.GetFields() {
		col := f.GetField()
		if col == "" {
			continue
		}
		direction := "ASC"
		if f.GetDirection() == commonpb.SortDirection_DESC {
			direction = "DESC"
		}
		return col, direction, true
	}
	return "", "", false
}

// sortColAllowed reports whether col is present in the whitelist.
func sortColAllowed(col string, allowedCols []string) bool {
	for _, c := range allowedCols {
		if c == col {
			return true
		}
	}
	return false
}

// quoteSortIdent double-quotes a postgres identifier so the validated column is
// interpolated safely. Any embedded double quote is escaped per the SQL rule of
// doubling it. The column is already whitelist-checked; this is defense in depth.
func quoteSortIdent(col string) string {
	return `"` + strings.ReplaceAll(col, `"`, `""`) + `"`
}
