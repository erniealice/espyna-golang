//go:build gin

package consumer

// Import gin adapter to trigger registration via init()
import (
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/primary/http/gin"
)
