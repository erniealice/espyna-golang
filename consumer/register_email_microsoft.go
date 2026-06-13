//go:build microsoft_email

package consumer

import (
	// Import Microsoft Graph email adapter to trigger registration via init().
	// The adapter is guarded by //go:build microsoft_email — the sole canonical tag.
	_ "github.com/erniealice/espyna-golang/contrib/microsoft"
)
