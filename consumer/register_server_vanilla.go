//go:build vanilla && !gin && !fiber && !fiber_v3

package consumer

// Import vanilla adapter to trigger registration via init()
import (
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/vanilla"
)
