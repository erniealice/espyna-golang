//go:build microsoft && microsoftgraph

package consumer

import (
	// Import Microsoft Graph email adapter to trigger registration via init()
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/email/microsoft"
)
