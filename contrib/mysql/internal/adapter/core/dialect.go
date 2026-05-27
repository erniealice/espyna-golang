//go:build mysql

// Package core holds the MySQL adapter's dialect primitive layer and generic
// CRUD helpers. It mirrors contrib/postgres/internal/adapter/core, but every
// SQL fragment is emitted using MySQL 8.0+ syntax.
package core

import (
	"strings"
)

// Dialect abstracts the per-RDBMS SQL fragments that differ between postgres
// (the gold standard), MySQL, and SQL Server. Postgres SQL is authored first;
// this MySQL implementation translates those fragments via the rules in
// docs/plan/20260527-multi-dialect-adapter-alignment/brief.md (the dialect
// translation table). Centralizing the differences here keeps the entity
// adapters dialect-agnostic.
type Dialect interface {
	// Placeholder returns the bound-parameter placeholder for argument n
	// (1-based). Postgres returns "$1"; MySQL returns "?"; SQL Server "@p1".
	Placeholder(n int) string

	// QuoteIdent wraps an identifier in the dialect's quoting characters.
	// Postgres uses "ident"; MySQL uses `ident`; SQL Server uses [ident].
	QuoteIdent(name string) string

	// BoolLiteral renders a boolean as a SQL literal. Postgres uses
	// true/false; MySQL and SQL Server use 1/0.
	BoolLiteral(b bool) string

	// Paginate appends ordering + paging clauses to a base SELECT.
	Paginate(sql, orderBy string, limit, offset int) string

	// ConditionalSum renders a conditional aggregate. Postgres can use
	// SUM(expr) FILTER (WHERE cond); MySQL/SQL Server must fall back to
	// SUM(CASE WHEN cond THEN expr END).
	ConditionalSum(expr, cond string) string

	// Now returns the dialect's current-timestamp function call.
	Now() string
}

// MySQLDialect implements Dialect for MySQL 8.0+.
type MySQLDialect struct{}

// Compile-time assertion that MySQLDialect satisfies the Dialect contract.
var _ Dialect = MySQLDialect{}

// NewMySQLDialect returns the MySQL dialect primitive set.
func NewMySQLDialect() MySQLDialect { return MySQLDialect{} }

// Placeholder returns "?" for every argument: MySQL uses positional
// placeholders, so the numeric index n is intentionally ignored.
func (MySQLDialect) Placeholder(n int) string { return "?" }

// QuoteIdent wraps name in backticks, the MySQL identifier quote character.
// Any embedded backtick is doubled per MySQL's escaping rule.
func (MySQLDialect) QuoteIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

// BoolLiteral renders true as "1" and false as "0" (MySQL has no native
// boolean literal; BOOL is an alias for TINYINT(1)).
func (MySQLDialect) BoolLiteral(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// Paginate appends ORDER BY (when supplied) followed by LIMIT/OFFSET. MySQL
// accepts LIMIT/OFFSET without an ORDER BY, but a stable sort is supplied by
// callers whenever paging is requested. Non-positive limit/offset values are
// omitted so callers can paginate, limit-only, or neither.
func (MySQLDialect) Paginate(sql, orderBy string, limit, offset int) string {
	var b strings.Builder
	b.WriteString(strings.TrimRight(sql, " \n\t"))
	if ob := strings.TrimSpace(orderBy); ob != "" {
		b.WriteString(" ORDER BY ")
		b.WriteString(ob)
	}
	if limit > 0 {
		b.WriteString(" LIMIT ")
		b.WriteString(itoa(limit))
	}
	if offset > 0 {
		b.WriteString(" OFFSET ")
		b.WriteString(itoa(offset))
	}
	return b.String()
}

// ConditionalSum renders SUM(CASE WHEN cond THEN expr END). MySQL has no
// FILTER (WHERE ...) clause, so the postgres-only filtered aggregate is
// translated to the portable CASE form.
func (MySQLDialect) ConditionalSum(expr, cond string) string {
	return "SUM(CASE WHEN " + cond + " THEN " + expr + " END)"
}

// Now returns MySQL's current-timestamp function.
func (MySQLDialect) Now() string { return "NOW()" }

// RewritePlaceholders converts postgres-style positional placeholders
// ($1, $2, ...) into MySQL positional placeholders (?). This is the key
// mechanical translation when reusing postgres-authored SQL: MySQL binds by
// position, so each $N becomes a single "?" in left-to-right order regardless
// of the numeric value. Repeated $N references are each rewritten to their own
// "?" — the caller is responsible for supplying the argument list in the
// correct positional order if a postgres query reused a parameter.
//
// A literal "$" not followed by a digit (e.g. inside a string literal such as
// '$5.00') is left untouched. Placeholder scanning is purely lexical; it does
// not parse SQL string literals, matching the simple translation contract the
// brief specifies.
func RewritePlaceholders(pgSQL string) string {
	var b strings.Builder
	b.Grow(len(pgSQL))

	for i := 0; i < len(pgSQL); i++ {
		c := pgSQL[i]
		if c != '$' {
			b.WriteByte(c)
			continue
		}
		// Look ahead: is this a $N placeholder (at least one digit)?
		j := i + 1
		for j < len(pgSQL) && pgSQL[j] >= '0' && pgSQL[j] <= '9' {
			j++
		}
		if j == i+1 {
			// "$" not followed by a digit — emit verbatim.
			b.WriteByte(c)
			continue
		}
		// Consumed $N — emit a single positional placeholder.
		b.WriteByte('?')
		i = j - 1
	}

	return b.String()
}

// itoa is a tiny non-allocating-ish integer formatter for the small,
// non-negative limit/offset values used in Paginate. Kept local so the
// dialect file has no fmt dependency for hot-path SQL assembly.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
