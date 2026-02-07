//go:build vanilla && !gin && !fiber && !fiber_v3

package consumer

// Import vanilla adapter to trigger registration via init()
import (
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/primary/http/vanilla"
)
