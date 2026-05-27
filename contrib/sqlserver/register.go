//go:build sqlserver

// Package sqlserver registers the SQL Server database adapter with espyna's
// registry. Import this package with a blank identifier to enable SQL Server
// support:
//
//	import _ "github.com/erniealice/espyna-golang/contrib/sqlserver"
//
// This is the greenfield foundation (MS-1): it wires the SQL Server driver and
// the dialect-primitive core package. Domain adapters (MS-2/3/4) self-register
// via their own init() functions under internal/adapter/ and will be added to
// the blank-import set as they land.
package sqlserver

import (
	// Driver registration: importing this package registers the "sqlserver"
	// (and legacy "mssql") driver names with database/sql.
	_ "github.com/microsoft/go-mssqldb"

	// Import triggers the core dialect package's init() and pulls the
	// dialect-primitive layer into the binary.
	_ "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"

	// Domain adapters: each sub-package self-registers its factory via init().
	_ "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/entity"
	_ "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/event"
	_ "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/fulfillment"
	_ "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/inventory"
	_ "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/operation"
)
