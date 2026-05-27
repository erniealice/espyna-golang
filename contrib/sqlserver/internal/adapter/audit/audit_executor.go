//go:build sqlserver

package audit

import (
	"context"
	"database/sql"

	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

func init() {
	registry.RegisterAuditServiceFactory(func(db any) any {
		sqlDB, ok := db.(*sql.DB)
		if !ok {
			return nil
		}
		return New(sqlDB)
	})
}

// executorProvider provides a transaction-aware database executor.
// Used by audit methods that need raw SQL access beyond the standard
// infraports.AuditService interface.
type executorProvider interface {
	GetExecutor(ctx context.Context) interface {
		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
		QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
		QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	}
}
