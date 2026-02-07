package infrastructure

import (
	"context"
	"time"
)

// MigrationService defines the contract for database migration management
// This interface abstracts database schema management across different providers
type MigrationService interface {
	// Up applies all pending migrations to bring the database to the latest version
	Up(ctx context.Context) error

	// Down rolls back the database to the previous migration version
	Down(ctx context.Context) error

	// Version returns the current migration version and whether the database is dirty
	// Returns: (version, dirty, error)
	// - version: current migration version number (0 if no migrations applied)
	// - dirty: true if a migration failed and left the database in an inconsistent state
	// - error: any error encountered while checking version
	Version(ctx context.Context) (uint, bool, error)

	// Status returns detailed migration status information
	Status(ctx context.Context) (*MigrationStatus, error)

	// Migrate applies migrations up to a specific version
	Migrate(ctx context.Context, targetVersion uint) error

	// Force marks a migration version as completed without running it
	// This is useful for fixing dirty database states
	Force(ctx context.Context, version uint) error

	// Close closes the migration service and cleans up resources
	Close() error
}

// MigrationStatus contains detailed information about the current migration state
type MigrationStatus struct {
	// Current version of the database schema
	CurrentVersion uint `json:"current_version"`

	// Whether the database is in a dirty state (incomplete migration)
	Dirty bool `json:"dirty"`

	// Total number of available migrations
	TotalMigrations uint `json:"total_migrations"`

	// List of applied migrations
	AppliedMigrations []AppliedMigration `json:"applied_migrations"`

	// List of pending migrations
	PendingMigrations []PendingMigration `json:"pending_migrations"`

	// Last migration timestamp
	LastMigrationTime *time.Time `json:"last_migration_time,omitempty"`
}

// AppliedMigration represents a migration that has been successfully applied
type AppliedMigration struct {
	Version     uint      `json:"version"`
	Description string    `json:"description"`
	AppliedAt   time.Time `json:"applied_at"`
}

// PendingMigration represents a migration that is available but not yet applied
type PendingMigration struct {
	Version     uint   `json:"version"`
	Description string `json:"description"`
	FilePath    string `json:"file_path"`
}

// MigrationError represents migration-related errors
type MigrationError struct {
	Code    string
	Message string
	Version uint
	Err     error
}

func (e *MigrationError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Migration error codes
const (
	MigrationErrCodeDirtyDatabase    = "DIRTY_DATABASE"
	MigrationErrCodeVersionNotFound  = "VERSION_NOT_FOUND"
	MigrationErrCodeMigrationFailed  = "MIGRATION_FAILED"
	MigrationErrCodeInvalidVersion   = "INVALID_VERSION"
	MigrationErrCodeConnectionFailed = "CONNECTION_FAILED"
)

// NewMigrationError creates a new migration error
func NewMigrationError(code, message string, version uint, err error) *MigrationError {
	return &MigrationError{
		Code:    code,
		Message: message,
		Version: version,
		Err:     err,
	}
}
