//go:build fiber && !vanilla && !gin && !fiber_v3

package consumer

// Import fiber adapter to trigger registration via init()
import (
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/primary/http/fiber"
)
