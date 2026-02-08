//go:build gin

package consumer

// Import gin adapter to trigger registration via init()
import (
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/gin"
)
