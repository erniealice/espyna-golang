//go:build mysql

// Package mysql is the MySQL adapter's self-registration entry point.
//
// init() registers the MySQL database operations factory with the espyna
// registry, so callers that blank-import this package (via
// "github.com/erniealice/espyna-golang/contrib/mysql") automatically have
// MySQL-backed operations available.
//
// Entity adapters in the entity/ subdirectory each have their own init()
// that registers a repository factory keyed by entityid.
package mysql

import (
	"database/sql"
	"fmt"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"

	// Blank imports trigger domain adapter init() registrations.
	_ "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/asset"
	_ "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/document"
	_ "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/entity"
	_ "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/event"
	_ "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/fulfillment"
	_ "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/integration"
	_ "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/inventory"
	_ "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/operation"
	_ "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/product"
	_ "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/subscription"
)

func init() {
	registry.RegisterDatabaseOperationsFactory("mysql", func(conn any) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql: expected *sql.DB, got %T", conn)
		}
		return mysqlCore.NewWorkspaceAwareOperations(db), nil
	})
}
