//go:build !postgresql

package core

import "github.com/erniealice/espyna-golang/reference"

// RefChecker fallback for non-postgres builds. Returns nil — callers that
// require actual checks are excluded from non-postgres deployments.
func (c *Container) RefChecker() reference.Checker { return nil }
