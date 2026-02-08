//go:build microsoft && microsoftgraph

package consumer

import (
	// Import Microsoft Graph email adapter to trigger registration via init()
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/email/microsoft"
)
