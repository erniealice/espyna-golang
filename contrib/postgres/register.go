// Package postgres registers the PostgreSQL database adapter with espyna's registry.
// Import this package with a blank identifier to enable PostgreSQL support:
//
//	import _ "github.com/erniealice/espyna-golang/contrib/postgres"
package postgres

import (
	// Import triggers adapter's init() which self-registers with the registry.
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter"
)
