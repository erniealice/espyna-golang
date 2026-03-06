
package firestore

import (
	// Repository sub-packages - each registers its factory via init()
	_ "github.com/erniealice/espyna-golang/contrib/google/internal/database/firestore/common"
	_ "github.com/erniealice/espyna-golang/contrib/google/internal/database/firestore/entity"
	_ "github.com/erniealice/espyna-golang/contrib/google/internal/database/firestore/event"
	_ "github.com/erniealice/espyna-golang/contrib/google/internal/database/firestore/integration"
	_ "github.com/erniealice/espyna-golang/contrib/google/internal/database/firestore/product"
	_ "github.com/erniealice/espyna-golang/contrib/google/internal/database/firestore/subscription"
	_ "github.com/erniealice/espyna-golang/contrib/google/internal/database/firestore/workflow"
)
