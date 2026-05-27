//go:build mysql

// Package payroll provides MySQL 8.0+ adapters for the payroll domain entities.
// Dialect translation from postgres gold standard:
//   - $N → ? (positional, re-sequenced)
//   - "ident" → `ident` (backtick quoting)
//   - ILIKE → LIKE (MySQL ci collation)
//   - FILTER (WHERE c) → SUM(CASE WHEN c THEN expr END)
//   - COUNT(*) OVER () stays (MySQL 8.0+ window functions)
//   - generate_series() → recursive CTE or application-side month generation
//   - row_to_json() → explicit column SELECTs + scan
//   - $N::text IS NULL → ? IS NULL (no cast needed in MySQL)
//   - RETURNING → app-side UUID + SELECT after insert (two-step)
//   - COUNT(*)::bigint → CAST(COUNT(*) AS SIGNED)
//
// Monetary values are always centavos — never divide/multiply by 100 in SQL.
package payroll

import (
	"context"
	"fmt"
	"time"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
)

// executorProvider is a local interface satisfied by
// mysqlCore.WorkspaceAwareOperations. Entity adapters use it to obtain the
// transaction-aware DBExecutor (either *sql.DB or an active *sql.Tx) for raw
// SQL queries that bypass the generic CRUD layer.
type executorProvider interface {
	GetExecutor(ctx context.Context) interfaces.DBExecutor
}

// convertMillisToTime converts a millis-epoch value in a JSON map to time.Time.
// Protobuf int64 fields serialize to JSON strings via protojson
// (e.g. "1771886746000"). MySQL datetime columns need time.Time, not raw millis.
func convertMillisToTime(data map[string]any, jsonKey, _ string) {
	v, ok := data[jsonKey]
	if !ok {
		return
	}
	switch val := v.(type) {
	case string:
		var millis int64
		if _, err := fmt.Sscanf(val, "%d", &millis); err == nil && millis > 1e12 {
			data[jsonKey] = time.UnixMilli(millis)
		}
	case float64:
		if val > 1e12 {
			data[jsonKey] = time.UnixMilli(int64(val))
		}
	}
}
