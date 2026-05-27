//go:build mysql

package core

import (
	"fmt"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// BuildOrderBy is the canonical sort-whitelist helper for MySQL list-page
// adapters. It mirrors contrib/postgres/internal/adapter/core.BuildOrderBy with
// one mechanical difference: identifiers are backtick-quoted (MySQL) instead of
// double-quoted (postgres). The quoting is delegated to MySQLDialect.QuoteIdent
// so the logic stays dialect-driven.
//
// allowedCols is the per-entity whitelist. It composes with the same slice that
// drives espynahttp.SortSpec.AllowedCols / espynahttp.ValidateSortColumns so the
// view layer and the adapter share one source of truth.
//
// Behavior:
//   - sort nil/empty (no fields, or first field blank) → return "ORDER BY " +
//     fallback verbatim (author-controlled constant, never caller-controlled).
//   - requested field NOT in allowedCols → return an error (fail closed). Callers
//     must propagate the error; silent fallback would hide probing / misconfigured
//     callers.
//   - direction is validated against the enum: SortDirection_DESC → "DESC",
//     everything else → "ASC".
//
// The returned fragment ("ORDER BY `col` DIR") is safe to interpolate into a
// query string with fmt.Sprintf because the column is both whitelist-validated
// and backtick-quoted, and the direction is enum-derived.
func BuildOrderBy(allowedCols []string, sort *commonpb.SortRequest, fallback string) (string, error) {
	field, dir, ok := firstSortField(sort)
	if !ok {
		return "ORDER BY " + fallback, nil
	}

	if !sortColAllowed(field, allowedCols) {
		return "", fmt.Errorf("unknown sort column %q (allowed: %v)", field, allowedCols)
	}

	d := NewMySQLDialect()
	return fmt.Sprintf("ORDER BY %s %s", d.QuoteIdent(field), dir), nil
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
