//go:build sqlserver

// Package core holds the SQL Server dialect-primitive layer. It is the single
// source of truth for every SQL-syntax difference between the postgres gold
// standard and SQL Server (Transact-SQL). Domain adapters (MS-2/3/4) translate
// the finished postgres queries by composing these primitives rather than
// hand-rolling per-call T-SQL.
//
// See docs/plan/20260527-multi-dialect-adapter-alignment/brief.md for the full
// dialect translation table.
package core

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Dialect is the per-adapter SQL-syntax primitive interface. The postgres,
// mysql, and sqlserver adapters each provide one implementation; callers
// compose these primitives so the surrounding query-building code stays
// dialect-agnostic.
//
// This declaration mirrors the canonical interface in the coordination brief.
// It is declared here (rather than imported from a shared package) so the
// sqlserver module self-resolves without depending on a not-yet-extracted
// common dialect package; the method set is identical across dialects.
type Dialect interface {
	Placeholder(n int) string      // "$1" | "?" | "@p1"
	QuoteIdent(name string) string // "x" | `x` | [x]
	BoolLiteral(b bool) string     // true/false | 1/0
	Paginate(sql, orderBy string, limit, offset int) string
	ConditionalSum(expr, cond string) string // FILTER vs CASE
	Now() string
}

// SQLServerDialect implements Dialect for Microsoft SQL Server 2017+ (T-SQL).
//
// Mapping summary (vs postgres gold standard):
//   - Placeholders: $1,$2,… → @p1,@p2,…
//   - Identifier quote: "ident" → [ident]
//   - Boolean literal: true/false → 1/0 (bit)
//   - Conditional aggregate: SUM(x) FILTER (WHERE c) → SUM(CASE WHEN c THEN x END)
//   - Pagination: LIMIT n OFFSET m → ORDER BY … OFFSET m ROWS FETCH NEXT n ROWS ONLY
//     (SQL Server REQUIRES an ORDER BY for OFFSET/FETCH)
//   - Now(): SYSUTCDATETIME() (UTC, matches postgres now()/CURRENT_TIMESTAMP usage)
type SQLServerDialect struct{}

// Compile-time assertion that SQLServerDialect satisfies Dialect.
var _ Dialect = SQLServerDialect{}

// DefaultDialect is the package-level singleton domain adapters compose with.
var DefaultDialect SQLServerDialect

// Placeholder returns the positional parameter marker for argument n (1-based),
// e.g. Placeholder(1) == "@p1". SQL Server uses named "@pN" markers (the
// convention emitted by github.com/microsoft/go-mssqldb for positional args).
func (SQLServerDialect) Placeholder(n int) string {
	return "@p" + strconv.Itoa(n)
}

// QuoteIdent wraps a SQL Server identifier in square brackets, escaping any
// embedded closing bracket by doubling it (the T-SQL rule). The result is safe
// to interpolate into a query string for an already-whitelisted column.
func (SQLServerDialect) QuoteIdent(name string) string {
	return "[" + strings.ReplaceAll(name, "]", "]]") + "]"
}

// BoolLiteral renders a boolean as a SQL Server bit literal: "1" / "0".
func (SQLServerDialect) BoolLiteral(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// ConditionalSum renders a conditional aggregate. SQL Server has no
// `FILTER (WHERE …)` clause, so it is expressed with a CASE expression:
//
//	SUM(CASE WHEN <cond> THEN <expr> END)
//
// The implicit ELSE NULL is correct: SUM ignores NULLs, matching the
// postgres FILTER semantics where non-matching rows contribute nothing.
func (SQLServerDialect) ConditionalSum(expr, cond string) string {
	return fmt.Sprintf("SUM(CASE WHEN %s THEN %s END)", cond, expr)
}

// stableOrderKey is the fallback ORDER BY used when Paginate is called with an
// empty orderBy. SQL Server REQUIRES an ORDER BY for OFFSET/FETCH, so a stable
// deterministic key must always be present. "(SELECT NULL)" is the canonical
// T-SQL idiom for "order is irrelevant but syntactically required" — it yields
// a stable (insertion-order) pass without forcing a specific column to exist.
const stableOrderKey = "(SELECT NULL)"

// Paginate appends SQL Server window pagination to a SELECT body. Unlike
// postgres `LIMIT n OFFSET m`, SQL Server uses
//
//	ORDER BY <orderBy> OFFSET <offset> ROWS FETCH NEXT <limit> ROWS ONLY
//
// and MANDATES an ORDER BY clause. If orderBy is empty we substitute a stable
// key so the statement remains valid. orderBy must be author-controlled /
// whitelist-validated (never raw caller input) because it is interpolated
// verbatim; offset and limit are integers and are safe to interpolate.
//
// The returned string is sql + a single trailing pagination fragment. Callers
// must pass a SELECT body that does NOT already contain its own ORDER BY /
// OFFSET clause.
func (SQLServerDialect) Paginate(sql, orderBy string, limit, offset int) string {
	order := strings.TrimSpace(orderBy)
	if order == "" {
		order = stableOrderKey
	}
	if offset < 0 {
		offset = 0
	}
	body := strings.TrimRight(strings.TrimSpace(sql), ";")
	return fmt.Sprintf(
		"%s ORDER BY %s OFFSET %d ROWS FETCH NEXT %d ROWS ONLY",
		body, order, offset, limit,
	)
}

// Now returns the SQL Server expression for the current UTC timestamp.
// SYSUTCDATETIME() (datetime2, microsecond precision, UTC) is the closest
// analogue to postgres now()/CURRENT_TIMESTAMP as used for created/updated
// timestamps in the gold-standard adapters.
func (SQLServerDialect) Now() string {
	return "SYSUTCDATETIME()"
}

// pgPlaceholderRE matches postgres positional placeholders ($1, $2, …). The
// negative lookbehind-free pattern relies on $ + digits; a trailing word
// boundary prevents "$12" being partially matched as "$1".
var pgPlaceholderRE = regexp.MustCompile(`\$(\d+)`)

// RewritePlaceholders converts a postgres-authored SQL string's positional
// placeholders ($1, $2, …) into SQL Server markers (@p1, @p2, …). This lets a
// domain adapter author a single query body in the postgres gold standard and
// mechanically translate it for SQL Server without renumbering arguments.
//
// Only the placeholder tokens are rewritten; the argument ORDER is preserved,
// so the existing args slice can be reused unchanged. Identifier-quoting,
// pagination, and conditional-aggregate differences are NOT handled here — use
// the Dialect primitives for those.
//
// Note: this is a lexical rewrite. It assumes $N tokens appear only as
// placeholders (the convention in the generated adapter queries) and not, e.g.,
// inside string literals containing a literal "$1".
func RewritePlaceholders(pgSQL string) string {
	return pgPlaceholderRE.ReplaceAllString(pgSQL, "@p$1")
}
