//go:build postgresql

package core

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"sync"
)

// Identifier columns compare exactly (plan 20260927-db-query-performance, Q1).
//
// A case-insensitive StringFilter used to emit LOWER(col) = lower($n) for every
// column. On an identifier column that wraps the indexed column in a function,
// so an id lookup became a sequential scan (audit DB-04: 2,150 buffers vs 10).
// Stored identifiers are lowercase (UUIDv7 / slug ids; verified by
// docs/plan/20260927-db-query-performance/sql/probe-identifier-case.sql), so
// col = lower($n) returns exactly the rows LOWER(col) = lower($n) did, and can
// use the index.
//
// An identifier column is a primary/foreign key column per the catalog, or a
// column named id / *_id — except humanCodeColumns, which hold operator-entered
// codes whose case is not normalized and keep case-insensitive matching.

// humanCodeColumns are id-shaped or key columns holding human-entered codes.
// client.internal_id (a student number) is the one column observed with
// uppercase values; tax_id and code are the same kind of value.
var humanCodeColumns = map[string]struct{}{
	"internal_id": {},
	"tax_id":      {},
	"code":        {},
}

// isIdentifierName applies the naming half of the rule; it needs no table.
func isIdentifierName(column string) bool {
	if _, human := humanCodeColumns[column]; human {
		return false
	}
	return column == "id" || strings.HasSuffix(column, "_id")
}

// keyColumnSet is the catalog's primary/foreign key columns, loaded once per
// connection pool. A load failure leaves it empty: matching falls back to the
// naming rule, and an unlisted key column simply keeps the old LOWER() form.
type keyColumnSet struct {
	once    sync.Once
	columns map[string]map[string]struct{} // table -> column
}

var keyColumnsByDB sync.Map // *sql.DB -> *keyColumnSet

const keyColumnsSQL = `
	SELECT DISTINCT cl.relname, a.attname
	FROM pg_constraint k
	JOIN pg_class cl ON cl.oid = k.conrelid
	JOIN pg_namespace n ON n.oid = cl.relnamespace AND n.nspname = current_schema()
	CROSS JOIN LATERAL unnest(k.conkey) AS u(attnum)
	JOIN pg_attribute a ON a.attrelid = k.conrelid AND a.attnum = u.attnum
	WHERE k.contype IN ('p', 'f')`

func loadKeyColumns(ctx context.Context, db *sql.DB) map[string]map[string]struct{} {
	rows, err := db.QueryContext(ctx, keyColumnsSQL)
	if err != nil {
		log.Printf("identifier columns: catalog key load failed, using naming rule only: %v", err)
		return nil
	}
	defer rows.Close()
	columns := make(map[string]map[string]struct{})
	for rows.Next() {
		var table, column string
		if err := rows.Scan(&table, &column); err != nil {
			log.Printf("identifier columns: catalog key scan failed, using naming rule only: %v", err)
			return nil
		}
		if columns[table] == nil {
			columns[table] = make(map[string]struct{})
		}
		columns[table][column] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		log.Printf("identifier columns: catalog key rows failed, using naming rule only: %v", err)
		return nil
	}
	return columns
}

// isIdentifierColumn reports whether an equality filter on table.column should
// compare exactly. The catalog is read from the pool, never the request
// transaction, so it cannot interleave with an open row stream.
func (p *PostgresOperations) isIdentifierColumn(ctx context.Context, table, column string) bool {
	if _, human := humanCodeColumns[column]; human {
		return false
	}
	if isIdentifierName(column) {
		return true
	}
	if p == nil || p.db == nil || table == "" {
		return false
	}
	entry, _ := keyColumnsByDB.LoadOrStore(p.db, &keyColumnSet{})
	set := entry.(*keyColumnSet)
	set.once.Do(func() { set.columns = loadKeyColumns(context.WithoutCancel(ctx), p.db) })
	_, ok := set.columns[table][column]
	return ok
}
