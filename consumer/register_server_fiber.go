//go:build fiber && !vanilla && !gin && !fiber_v3

package consumer

// Import fiber adapter to trigger registration via init()
import (
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/fiber"
)
