// Package helpers re-exports internal storage helper functions for use by contrib sub-modules.
package helpers

import (
	internal "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/storage/common"
)

var (
	GenerateObjectID  = internal.GenerateObjectID
	DetectContentType = internal.DetectContentType
)
