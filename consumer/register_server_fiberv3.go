//go:build fiber_v3

package consumer

// Import fiberv3 adapter to trigger registration via init()
import (
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/fiberv3"
)
