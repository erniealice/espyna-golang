//go:build postgres

package postgres

import (
	// Repository sub-packages - each registers its factory via init()
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres/entity"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres/event"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres/integrations"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres/payment"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres/product"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres/subscription"
)
