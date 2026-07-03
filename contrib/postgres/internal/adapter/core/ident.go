//go:build postgresql

package core

import (
	"fmt"
	"regexp"
)

// sqlIdentRe matches a bare or single-qualified postgres identifier
// (`column` or `alias.column`). Anything else — spaces, quotes, parens,
// semicolons, operators — is rejected.
var sqlIdentRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$`)

// ValidateSQLIdent is the fail-closed guard for every request-supplied column
// name that gets interpolated into query text (filter fields, search fields,
// sort fields). Values are always bound via $N placeholders, but identifiers
// cannot be bound — so any identifier that reaches an fmt.Sprintf/concat site
// MUST pass this first. Callers must propagate the error, not silently drop
// the field: a dropped filter widens the result set (fail-open), and a silent
// fallback hides a probing caller (the A2 lesson — see BuildOrderBy).
func ValidateSQLIdent(field string) error {
	if !sqlIdentRe.MatchString(field) {
		return fmt.Errorf("invalid SQL identifier %q", field)
	}
	return nil
}
