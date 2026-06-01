//go:build microsoft_email

package consumer

import (
	// Import Microsoft Graph email adapter to trigger registration via init().
	// NOTE: The microsoft email adapter (email/microsoft/adapter.go) is guarded by
	// //go:build microsoft && microsoftgraph. Until that file's tag is updated to also
	// accept microsoft_email, builds must supply all three tags:
	//   -tags "microsoft_email,microsoft,microsoftgraph"
	// This tag-alignment fix is tracked separately (out of this agent's scope).
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/email/microsoft"
)
