//go:build fiber_v3

package consumer

// Import fiberv3 adapter to trigger registration via init()
import (
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/primary/http/fiberv3"
)
