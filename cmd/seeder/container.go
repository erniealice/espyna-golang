package main

import (
	"github.com/erniealice/espyna-golang/internal/composition/core"

	// Import only required adapters (no HTTP adapters needed for seeder)
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/auth/mock"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/id/noop"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/id/uuidv7"
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/storage/mock"
)

// SeederContainer wraps core.Container for seeder use
type SeederContainer = core.Container

// NewSeederContainer creates a container for the seeder from environment
func NewSeederContainer() (*SeederContainer, error) {
	return core.NewContainerFromEnv()
}
