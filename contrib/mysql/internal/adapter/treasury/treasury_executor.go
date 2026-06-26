//go:build mysql

// Package treasury provides MySQL 8.0+ adapters for all treasury domain
// entities. The SQL in each file is mechanically translated from the postgres
// gold standard via the dialect rules in
// docs/plan/20260527-multi-dialect-adapter-alignment/brief.md:
//
//   - $N → ? (positional, re-sequenced)
//   - "ident" → `ident` (backtick quoting)
//   - ILIKE → LIKE (MySQL ci collation handles case-insensitivity)
//   - FILTER (WHERE c) → SUM(CASE WHEN c THEN expr END)
//   - COUNT(*) OVER () stays (MySQL 8.0+ window functions)
//   - RETURNING → app-side UUID + SELECT after insert (two-step)
//
// Every query enforces workspace_id isolation (multi-tenancy guardrail).
// Monetary values are always centavos — never divide/multiply by 100 in SQL.
package treasury

import (
	"context"
	"fmt"
	"time"

	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
)

// executorProvider is a local interface satisfied by
// mysqlCore.WorkspaceAwareOperations. Entity adapters use it to obtain the
// transaction-aware DBExecutor (either *sql.DB or an active *sql.Tx) for raw
// SQL queries that bypass the generic CRUD layer.
type executorProvider interface {
	GetExecutor(ctx context.Context) sqlexec.DBExecutor
}

// convertMillisToTime converts a millis-epoch value in a JSON map to time.Time.
// Protobuf int64 fields serialize to JSON strings via protojson
// (e.g. "1771886746000"). MySQL datetime columns need time.Time, not raw millis.
// This is identical in behaviour to the postgres version — the conversion is
// dialect-agnostic.
func convertMillisToTime(data map[string]any, jsonKey, _ string) {
	v, ok := data[jsonKey]
	if !ok {
		return
	}
	switch val := v.(type) {
	case string:
		// protojson serializes int64 as string.
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
