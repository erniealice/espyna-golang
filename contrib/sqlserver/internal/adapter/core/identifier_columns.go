//go:build sqlserver

package core

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"sync"
	"time"
)

// Identifier equality can use an ordinary index when the stored identifier is
// normalized to lowercase. Human-entered codes retain case-insensitive matching.
var humanCodeColumns = map[string]struct{}{
	"internal_id": {},
	"tax_id":      {},
	"code":        {},
}

func isIdentifierName(column string) bool {
	if _, human := humanCodeColumns[column]; human {
		return false
	}
	return column == "id" || strings.HasSuffix(column, "_id")
}

type keyColumnSet struct {
	once    sync.Once
	columns map[string]map[string]struct{}
}

var keyColumnsByDB sync.Map // *sql.DB -> *keyColumnSet

const keyColumnsSQL = `
	SELECT DISTINCT t.name, c.name
	FROM sys.tables AS t
	JOIN sys.columns AS c ON c.object_id = t.object_id
	WHERE t.schema_id = SCHEMA_ID()
	  AND (
	    EXISTS (
	      SELECT 1 FROM sys.indexes AS i
	      JOIN sys.index_columns AS ic ON ic.object_id = i.object_id AND ic.index_id = i.index_id
	      WHERE i.object_id = t.object_id AND i.is_primary_key = 1 AND ic.column_id = c.column_id
	    )
	    OR EXISTS (
	      SELECT 1 FROM sys.foreign_key_columns AS fk
	      WHERE fk.parent_object_id = t.object_id AND fk.parent_column_id = c.column_id
	    )
	  )`

func loadKeyColumns(ctx context.Context, db *sql.DB) map[string]map[string]struct{} {
	rows, err := db.QueryContext(ctx, keyColumnsSQL)
	if err != nil {
		log.Printf("sqlserver identifier columns: catalog key load failed, using naming rule only: %v", err)
		return nil
	}
	defer rows.Close()
	columns := make(map[string]map[string]struct{})
	for rows.Next() {
		var table, column string
		if err := rows.Scan(&table, &column); err != nil {
			log.Printf("sqlserver identifier columns: catalog key scan failed, using naming rule only: %v", err)
			return nil
		}
		if columns[table] == nil {
			columns[table] = make(map[string]struct{})
		}
		columns[table][column] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		log.Printf("sqlserver identifier columns: catalog key rows failed, using naming rule only: %v", err)
		return nil
	}
	return columns
}

func (s *SQLServerOperations) isIdentifierColumn(ctx context.Context, table, column string) bool {
	if _, human := humanCodeColumns[column]; human {
		return false
	}
	if isIdentifierName(column) {
		return true
	}
	if s == nil || s.db == nil || table == "" {
		return false
	}
	entry, _ := keyColumnsByDB.LoadOrStore(s.db, &keyColumnSet{})
	set := entry.(*keyColumnSet)
	set.once.Do(func() {
		loadCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		set.columns = loadKeyColumns(loadCtx, s.db)
	})
	_, ok := set.columns[table][column]
	return ok
}
