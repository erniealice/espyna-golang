//go:build google && googlesheets

package consumer

// Import Google Sheets adapter to trigger registration via init()
// This file is only compiled when both 'google' and 'googlesheets' build tags are present
import (
	_ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/tabular/googlesheets"
)
