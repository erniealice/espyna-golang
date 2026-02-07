package seeders

import "leapfor.xyz/vya"

// Options configures seeder behavior
type Options struct {
	// WorkspaceID is the target workspace for seeding
	WorkspaceID string

	// BusinessType filters templates to a specific business type
	BusinessType vya.BusinessType

	// Reset deletes existing templates before seeding
	Reset bool

	// DryRun previews changes without persisting
	DryRun bool

	// TemplateID seeds a specific template only
	TemplateID string

	// Verbose enables detailed output
	Verbose bool
}
