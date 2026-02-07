//go:build postgres

package postgres

import (
	"github.com/lib/pq"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/model"
)

// mapPostgresError maps a PostgreSQL-specific error to a common DatabaseError
func mapPostgresError(err error) *model.DatabaseError {
	if pqErr, ok := err.(*pq.Error); ok {
		switch pqErr.Code.Name() {
		case "unique_violation":
			return model.NewDatabaseError(pqErr.Message, "UNIQUE_VIOLATION", 409)
		case "foreign_key_violation":
			return model.NewDatabaseError(pqErr.Message, "FOREIGN_KEY_VIOLATION", 400)
		case "not_null_violation":
			return model.NewDatabaseError(pqErr.Message, "NOT_NULL_VIOLATION", 400)
		case "check_violation":
			return model.NewDatabaseError(pqErr.Message, "CHECK_VIOLATION", 400)
		default:
			return model.NewDatabaseError(pqErr.Message, "POSTGRES_ERROR", 500)
		}
	}
	return model.NewDatabaseError(err.Error(), "UNKNOWN_DATABASE_ERROR", 500)
}
