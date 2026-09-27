//go:build sqlserver

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

func (s *SQLServerOperations) getTableMetadata(ctx context.Context, tableName string) (tableMetadata, error) {
	key := tableMetadataKey{s.db, tableName}
	if cached, ok := tableMetadataCache.Load(key); ok {
		return cached.(tableMetadata), nil
	}
	const query = `
		SELECT COLUMN_NAME, DATA_TYPE
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = SCHEMA_NAME() AND TABLE_NAME = @p1
		ORDER BY ORDINAL_POSITION`
	rows, err := s.getExecutor(ctx).QueryContext(ctx, query, tableName)
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
		return tableMetadata{}, fmt.Errorf("sqlserver table %q has no columns in current schema", tableName)
	}
	tableMetadataCache.Store(key, metadata)
	return metadata, nil
}
