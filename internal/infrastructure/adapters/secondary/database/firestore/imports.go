//go:build firestore

package firestore

import (
	// Repository sub-packages - each registers its factory via init()
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/common"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/entity"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/event"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/integration"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/payment"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/product"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/subscription"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/workflow"
)
