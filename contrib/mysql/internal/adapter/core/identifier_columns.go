//go:build mysql

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
	SELECT DISTINCT TABLE_NAME, COLUMN_NAME
	FROM information_schema.KEY_COLUMN_USAGE
	WHERE TABLE_SCHEMA = DATABASE()
	  AND (CONSTRAINT_NAME = 'PRIMARY' OR REFERENCED_TABLE_NAME IS NOT NULL)`

func loadKeyColumns(ctx context.Context, db *sql.DB) map[string]map[string]struct{} {
	rows, err := db.QueryContext(ctx, keyColumnsSQL)
	if err != nil {
		log.Printf("mysql identifier columns: catalog key load failed, using naming rule only: %v", err)
		return nil
	}
	defer rows.Close()
	columns := make(map[string]map[string]struct{})
	for rows.Next() {
		var table, column string
		if err := rows.Scan(&table, &column); err != nil {
			log.Printf("mysql identifier columns: catalog key scan failed, using naming rule only: %v", err)
			return nil
		}
		if columns[table] == nil {
			columns[table] = make(map[string]struct{})
		}
		columns[table][column] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		log.Printf("mysql identifier columns: catalog key rows failed, using naming rule only: %v", err)
		return nil
	}
	return columns
}

func (m *MySQLOperations) isIdentifierColumn(ctx context.Context, table, column string) bool {
	if _, human := humanCodeColumns[column]; human {
		return false
	}
	if isIdentifierName(column) {
		return true
	}
	if m == nil || m.db == nil || table == "" {
		return false
	}
	entry, _ := keyColumnsByDB.LoadOrStore(m.db, &keyColumnSet{})
	set := entry.(*keyColumnSet)
	set.once.Do(func() {
		loadCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		set.columns = loadKeyColumns(loadCtx, m.db)
	})
	_, ok := set.columns[table][column]
	return ok
}
