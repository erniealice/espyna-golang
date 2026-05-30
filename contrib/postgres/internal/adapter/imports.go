//go:build postgresql

package postgres

import (
	// Repository sub-packages - each registers its factory via init()
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/asset"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/common"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/document"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/entity"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/event"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/expenditure"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/finance"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/fulfillment"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/funding"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/integration"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/inventory"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/ledger"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/operation"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/payroll"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/procurement"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/product"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/rbac"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/revenue"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/subscription"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/tax"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/tenancy"
	_ "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/treasury"
)
