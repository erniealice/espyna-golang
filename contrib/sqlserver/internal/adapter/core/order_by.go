//go:build sqlserver

// Package core holds the SQL Server adapter's dialect primitive layer and generic
// CRUD helpers. It mirrors contrib/postgres/internal/adapter/core, but every
// SQL fragment uses SQL Server 2017+ syntax — square-bracket identifier quoting,
// @pN placeholders, CASE aggregation (no FILTER), and OFFSET/FETCH pagination.
package core

import (
	"fmt"
	"strings"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// BuildOrderBy is the canonical sort-whitelist helper for SQL Server list-page
// adapters. It is the fail-closed guard against unguarded ORDER BY interpolation
// (the A2 canonical bug). Every page-data method that lets the caller choose a
// sort column must route the column through this helper.
//
// SQL Server OFFSET/FETCH pagination requires an ORDER BY clause; a missing or
// defaulted sort is therefore especially important to handle safely. The fallback
// parameter is author-controlled (a trusted constant) and is used verbatim when
// no sort is requested.
//
// Behavior:
//   - sort nil/empty → "ORDER BY " + fallback (verbatim, trusted constant).
//   - field NOT in allowedCols → error (fail closed, no silent fallback).
//   - direction validated to ASC / DESC only; defaults to ASC.
//
// The returned fragment is `ORDER BY [col] DIR`. The column is square-bracket
// quoted via quoteSortIdent (SQL Server quoting), so the fragment is safe to
// interpolate into a query string with fmt.Sprintf after whitelist validation.
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

// quoteSortIdent wraps a SQL Server identifier in square brackets. Any embedded
// closing bracket is escaped by doubling it per SQL Server's quoting rule. The
// column is already whitelist-checked; this is defense in depth.
func quoteSortIdent(col string) string {
	return "[" + strings.ReplaceAll(col, "]", "]]") + "]"
}
