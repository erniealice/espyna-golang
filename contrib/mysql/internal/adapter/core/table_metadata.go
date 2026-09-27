//go:build mysql

package core

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
)

type tableMetadata struct {
	columns []string
	types   map[string]string
}

type tableMetadataKey struct {
	db    *sql.DB
	table string
}

// A successful catalog read is stable for the lifetime of the pool. A failed
// read is never cached, so a transient metadata failure can recover.
var tableMetadataCache sync.Map // tableMetadataKey -> tableMetadata

func (m *MySQLOperations) getTableMetadata(ctx context.Context, tableName string) (tableMetadata, error) {
	key := tableMetadataKey{m.db, tableName}
	if cached, ok := tableMetadataCache.Load(key); ok {
		return cached.(tableMetadata), nil
	}
	const query = `
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = ?
		ORDER BY ordinal_position`
	rows, err := m.getExecutor(ctx).QueryContext(ctx, query, tableName)
	if err != nil {
		return tableMetadata{}, err
	}
	defer rows.Close()
	metadata := tableMetadata{types: make(map[string]string)}
	for rows.Next() {
		var name, dataType string
		if err := rows.Scan(&name, &dataType); err != nil {
			return tableMetadata{}, err
		}
		metadata.columns = append(metadata.columns, name)
		metadata.types[name] = dataType
	}
	if err := rows.Err(); err != nil {
		return tableMetadata{}, err
	}
	if len(metadata.columns) == 0 {
		return tableMetadata{}, fmt.Errorf("mysql table %q has no columns in current database", tableName)
	}
	tableMetadataCache.Store(key, metadata)
	return metadata, nil
}
