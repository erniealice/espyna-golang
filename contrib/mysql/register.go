//go:build mysql

// Package mysql registers the MySQL database adapter with espyna's registry.
// Import this package with a blank identifier to enable MySQL support:
//
//	import _ "github.com/erniealice/espyna-golang/contrib/mysql"
package mysql

import (
	// Import triggers adapter's init() which self-registers with the registry.
	_ "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter"

	// Register the MySQL SQL driver (database/sql "mysql" driver name).
	_ "github.com/go-sql-driver/mysql"
)
