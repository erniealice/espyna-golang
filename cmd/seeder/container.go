package main

import (
	"leapfor.xyz/espyna/internal/composition/core"

	// Import only required adapters (no HTTP adapters needed for seeder)
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/auth/mock"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/id/noop"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/id/uuidv7"
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/storage/mock"
)

// SeederContainer wraps core.Container for seeder use
type SeederContainer = core.Container

// NewSeederContainer creates a container for the seeder from environment
func NewSeederContainer() *SeederContainer {
	return core.NewContainerFromEnv()
}
