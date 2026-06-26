// Package sqlexec re-exports the DBExecutor interface for use by contrib sub-modules.
//
// SQL-specific adapters (postgres, mysql, sqlserver) import this package.
// Dialect-neutral code imports database/interfaces instead — which does NOT
// depend on database/sql.
package sqlexec

import (
	internal "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface/sqlexec"
)

// DBExecutor abstracts *sql.DB and *sql.Tx for uniform query execution.
type DBExecutor = internal.DBExecutor
