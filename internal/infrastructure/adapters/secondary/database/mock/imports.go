//go:build mock_db

package mock

import (
	// Repository sub-packages - each registers its factory via init()
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/common"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/event"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/integration"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/payment"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/product"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/subscription"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/workflow"
)
