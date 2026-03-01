//go:build firestore

package firestore

import (
	// Repository sub-packages - each registers its factory via init()
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/common"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/entity"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/event"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/integration"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/product"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/subscription"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/workflow"
)
