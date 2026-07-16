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
// The returned fragment is `ORDER BY <col> <DIR>, <tie> ASC` with the whitelisted
// column safely double-quoted. Because the column is whitelist-validated AND
// quoted, and the direction is enum-derived, the fragment is safe to interpolate
// into a query string with fmt.Sprintf.
//
// fallback is the default ORDER BY body when no sort is requested, e.g.
// "date_created DESC". It is interpolated verbatim (it is author-controlled,
// never caller-controlled) so authors must keep it a trusted constant.
//
// DETERMINISTIC-PAGINATION TIEBREAKER: the emitted clause always terminates in a
// UNIQUE final key so the LIMIT/OFFSET pages these hand-written list adapters
// produce are stable, disjoint, and complete even when rows TIE on the leading
// sort key (e.g. many rows bulk-created under a single date_created). Without a
// unique final key Postgres may return tied rows in a different order per fetch,
// silently dropping some rows from every page and duplicating others — the same
// bug the base List.buildListOrderByClause fix closed. The tiebreaker is appended
// to BOTH the fallback and the caller-sort paths, unless the resolved ORDER BY
// already sorts by an id key (no redundant double-append; case/qualifier-aware).
//
// tiebreaker is optional and defaults to the primary key `id`, which is correct
// for the overwhelming majority of callers because their ORDER BY sits on a
// single-relation outer query — a bare table, a single aliased table, or a
// CTE-wrapped subquery alias (`SELECT ... FROM enriched e ORDER BY ...`) whose
// sole exposed `id` column resolves unambiguously. A caller whose ORDER BY sits
// on a JOINED outer query (two+ relations each carrying `id` directly in scope)
// must pass its primary table's alias-qualified id (e.g. "rb.id") so the appended
// key is unambiguous — an empty string keeps the "id" default. Direction is always
// ASC: stability needs a unique final key, not a particular orientation.
//
// The tiebreaker crosses the same fail-closed boundary as a whitelisted sort
// column: it is validated as a dotted identifier PATH whose FINAL component is
// exactly `id` (an injection string or a non-id key is rejected — the latter would
// silently defeat the uniqueness invariant) and emitted double-quoted PER
// COMPONENT (`"rb"."id"`, never the single identifier `"rb.id"`).
func BuildOrderBy(allowedCols []string, sort *commonpb.SortRequest, fallback string, tiebreaker ...string) (string, error) {
	tie := "id"
	if len(tiebreaker) > 0 && tiebreaker[0] != "" {
		tie = tiebreaker[0]
	}
	quotedTie, err := validateAndQuoteTiebreaker(tie)
	if err != nil {
		return "", err
	}

	field, dir, ok := firstSortField(sort)
	if !ok {
		return "ORDER BY " + appendTiebreaker(fallback, quotedTie), nil
	}

	if !sortColAllowed(field, allowedCols) {
		return "", fmt.Errorf("unknown sort column %q (allowed: %v)", field, allowedCols)
	}

	body := fmt.Sprintf("%s %s", quoteSortIdent(field), dir)
	return "ORDER BY " + appendTiebreaker(body, quotedTie), nil
}

// validateAndQuoteTiebreaker fail-closes the caller-supplied tiebreaker expression
// (same posture as ValidateSQLIdent for sort columns): every dot-separated
// component must be a bare SQL identifier, and the FINAL component must be exactly
// `id` (case-insensitive) so the appended key is always the unique primary key —
// a non-id key would silently defeat the deterministic-pagination invariant. The
// returned form is double-quoted per component (`"rb"."id"`) so it is safe to
// interpolate verbatim. The "id" default and the three constant alias-qualified
// callers ("cp.id"/"rb.id"/"rem.id") pass; a request-derived injection string such
// as `id ASC; DROP TABLE x; --` is rejected.
func validateAndQuoteTiebreaker(tie string) (string, error) {
	comps := strings.Split(tie, ".")
	for _, c := range comps {
		if err := ValidateSQLIdent(c); err != nil {
			return "", fmt.Errorf("invalid tiebreaker %q: %w", tie, err)
		}
	}
	if !strings.EqualFold(comps[len(comps)-1], "id") {
		return "", fmt.Errorf("invalid tiebreaker %q: final path component must be \"id\" (uniqueness invariant)", tie)
	}
	for i, c := range comps {
		comps[i] = `"` + strings.ReplaceAll(c, `"`, `""`) + `"`
	}
	return strings.Join(comps, "."), nil
}

// appendTiebreaker appends ", <quotedTie> ASC" to an ORDER BY body unless the body
// already sorts by an id key (avoid a redundant duplicate final key). quotedTie is
// the already-validated, per-component-quoted tiebreaker (never caller-derived raw).
func appendTiebreaker(body, quotedTie string) string {
	if orderByHasIDKey(body) {
		return body
	}
	return body + ", " + quotedTie + " ASC"
}

// orderByHasIDKey reports whether any sort segment of an ORDER BY body already
// sorts by an id key, so the tiebreaker is not double-appended. It is quote-aware
// (mirrors the per-component quoting quoteSortIdent now emits): segments are split
// on commas OUTSIDE double quotes (so a column literally named `x,id` is not torn
// into a phantom `id` segment), and each segment's leading path token has its FINAL
// component unquoted and matched case-insensitively to `id` — recognizing the
// canonical `"id"` and `"rb"."id"` quoted forms as well as bare `id` / `alias.id`.
func orderByHasIDKey(body string) bool {
	for _, seg := range splitOutsideQuotes(body, ',') {
		tok := leadingSortToken(seg)
		if tok == "" {
			continue
		}
		if finalPathComponent(tok) == "id" {
			return true
		}
	}
	return false
}

// splitOutsideQuotes splits s on sep, ignoring sep bytes that appear inside a
// double-quoted region. Per the SQL rule, a doubled "" within a quoted region is
// an escaped quote, not a region close. Used to walk ORDER BY commas and dotted
// path dots without being fooled by a quoted identifier that embeds them.
func splitOutsideQuotes(s string, sep byte) []string {
	var parts []string
	var cur strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' {
			if inQuote && i+1 < len(s) && s[i+1] == '"' {
				cur.WriteByte(c)
				cur.WriteByte(s[i+1])
				i++
				continue
			}
			inQuote = !inQuote
			cur.WriteByte(c)
			continue
		}
		if c == sep && !inQuote {
			parts = append(parts, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	parts = append(parts, cur.String())
	return parts
}

// leadingSortToken returns the column/path expression at the head of an ORDER BY
// segment — everything up to the first whitespace OUTSIDE double quotes (which
// strips the trailing ASC/DESC/NULLS ... options).
func leadingSortToken(seg string) string {
	seg = strings.TrimSpace(seg)
	inQuote := false
	for i := 0; i < len(seg); i++ {
		c := seg[i]
		if c == '"' {
			if inQuote && i+1 < len(seg) && seg[i+1] == '"' {
				i++
				continue
			}
			inQuote = !inQuote
			continue
		}
		if !inQuote && (c == ' ' || c == '\t' || c == '\n' || c == '\r') {
			return seg[:i]
		}
	}
	return seg
}

// finalPathComponent returns the last dot-separated component of a (possibly
// per-component double-quoted) path token, unquoted and lowercased — e.g.
// `"rb"."id"` → `id`, `j.date_created` → `date_created`, `"x,id"` → `x,id`.
func finalPathComponent(token string) string {
	comps := splitOutsideQuotes(token, '.')
	last := strings.TrimSpace(comps[len(comps)-1])
	if len(last) >= 2 && last[0] == '"' && last[len(last)-1] == '"' {
		last = strings.ReplaceAll(last[1:len(last)-1], `""`, `"`)
	}
	return strings.ToLower(last)
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

// quoteSortIdent double-quotes a whitelisted postgres sort column so it is
// interpolated safely. It quotes PER DOT-SEPARATED COMPONENT — a qualified column
// `alias.col` becomes `"alias"."col"`, NOT the single identifier `"alias.col"`
// (which references a nonexistent column literally named `alias.col` and breaks
// any dotted whitelist entry such as rate_band's `rb.id`). A bare column becomes
// `"col"`. Any embedded double quote is escaped per the SQL rule of doubling it.
// The column is already whitelist-checked; this is defense in depth.
func quoteSortIdent(col string) string {
	comps := splitOutsideQuotes(col, '.')
	for i, c := range comps {
		comps[i] = `"` + strings.ReplaceAll(c, `"`, `""`) + `"`
	}
	return strings.Join(comps, ".")
}
