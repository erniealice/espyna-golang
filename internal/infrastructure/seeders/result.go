package seeders

// SeedResult captures the outcome of a seeder run
type SeedResult struct {
	SeederName string
	Created    int
	Skipped    int
	Deleted    int
	Errors     []string
	Details    []SeedDetail
}

// SeedDetail provides information about individual seed operations
type SeedDetail struct {
	ID     string
	Name   string
	Action string // "created", "skipped", "deleted", "error", "would_create", "would_skip"
	Reason string
}
