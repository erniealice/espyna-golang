//go:build mock_db

package mock

import (
	// Repository sub-packages - each registers its factory via init()
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/common"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/event"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/integration"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/payment"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/product"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/subscription"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/workflow"
)
