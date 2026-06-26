//go:build postgresql

package core

import (
	"database/sql"

	// Blank-import triggers contrib/postgres/register.go init() which
	// self-registers the postgres database provider. Importing
	// contrib/postgres/reference alone is NOT sufficient — Go imports
	// are per-package, and the reference subpackage doesn't pull in
	// the parent package's init().
	_ "github.com/erniealice/espyna-golang/contrib/postgres"
	pgref "github.com/erniealice/espyna-golang/contrib/postgres/reference"
	"github.com/erniealice/espyna-golang/ports"
)

// RefChecker returns the postgres-backed reference checker resolved from the
// container's database provider. Compiled only when the postgresql build tag
// is active so non-postgres builds don't drag contrib/postgres into the binary.
//
// Returns nil when the database provider is not available or does not expose a
// *sql.DB connection (e.g. firestore).
func (c *Container) RefChecker() ports.Checker {
	dbProvider := c.GetDatabaseProvider()
	if dbProvider == nil {
		return nil
	}
	connHolder, ok := dbProvider.(interface{ GetConnection() any })
	if !ok {
		return nil
	}
	conn := connHolder.GetConnection()
	if conn == nil {
		return nil
	}
	sqlDB, ok := conn.(*sql.DB)
	if !ok || sqlDB == nil {
		return nil
	}
	return pgref.NewChecker(sqlDB)
}
