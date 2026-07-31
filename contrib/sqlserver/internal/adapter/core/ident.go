//go:build sqlserver

package core

import (
	"fmt"
	"regexp"
)

// sqlIdentRe matches a bare or single-qualified SQL identifier
// (`column` or `alias.column`). Anything else — spaces, quotes, brackets,
// parens, semicolons, operators — is rejected.
//
// The pattern is byte-identical to contrib/postgres/internal/adapter/core/ident.go
// and contrib/mysql/internal/adapter/core/ident.go ON PURPOSE. It is
// dialect-neutral (it accepts only [A-Za-z0-9_] plus one dot), and a "local
// improvement" here would reintroduce exactly the per-dialect drift that
// docs/plan/20260728-sql-injection-dialect-parity/plan.md exists to close. Any
// change must be made in all three files at once.
var sqlIdentRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$`)

// ValidateSQLIdent is the fail-closed guard for every request-supplied column
// name that gets interpolated into query text (filter fields, search fields,
// sort fields). Values are always bound via @pN placeholders, but identifiers
// cannot be bound — so any identifier that reaches an fmt.Sprintf/concat site
// MUST pass this first. Callers must propagate the error, not silently drop
// the field: a dropped filter widens the result set (fail-open), and a silent
// fallback hides a probing caller (the A2 lesson — see BuildOrderBy).
//
// Note the empty string is REJECTED: sqlIdentRe requires a leading
// [A-Za-z_], so "" does not match. An unvalidated empty field would otherwise
// interpolate a zero-width identifier into query text (the empty-string trap in
// docs/wiki/articles/infra-sql-no-direct-sql-rule.md).
func ValidateSQLIdent(field string) error {
	if !sqlIdentRe.MatchString(field) {
		return fmt.Errorf("invalid SQL identifier %q", field)
	}
	return nil
}
