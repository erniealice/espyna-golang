//go:build postgres

package postgres

import (
	// Repository sub-packages - each registers its factory via init()
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/entity"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/event"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/integrations"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/payment"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/product"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/subscription"
)
