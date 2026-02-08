//go:build postgres

package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/erniealice/espyna-golang/internal/application/ports"
)

// PostgresMigrationService implements the MigrationService interface for PostgreSQL
type PostgresMigrationService struct {
	migrate        *migrate.Migrate
	db             *sql.DB
	migrationsPath string
}

// NewPostgresMigrationService creates a new PostgreSQL migration service
func NewPostgresMigrationService(db *sql.DB, migrationsPath string) (ports.MigrationService, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	// Validate and normalize migrations path
	if migrationsPath == "" {
		migrationsPath = "migrations"
	}

	// Ensure the path is absolute and use file:// scheme
	if !strings.HasPrefix(migrationsPath, "file://") {
		absPath, err := filepath.Abs(migrationsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve absolute path for migrations: %w", err)
		}
		migrationsPath = "file://" + absPath
	}

	// Create database driver
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres driver: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "postgres", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return &PostgresMigrationService{
		migrate:        m,
		db:             db,
		migrationsPath: migrationsPath,
	}, nil
}

// Up applies all pending migrations to bring the database to the latest version
func (p *PostgresMigrationService) Up(ctx context.Context) error {
	err := p.migrate.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return ports.NewMigrationError(
			ports.MigrationErrCodeMigrationFailed,
			"failed to apply migrations up",
			0,
			err,
		)
	}
	return nil
}

// Down rolls back the database to the previous migration version
func (p *PostgresMigrationService) Down(ctx context.Context) error {
	err := p.migrate.Steps(-1)
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return ports.NewMigrationError(
			ports.MigrationErrCodeMigrationFailed,
			"failed to rollback migration",
			0,
			err,
		)
	}
	return nil
}

// Version returns the current migration version and whether the database is dirty
func (p *PostgresMigrationService) Version(ctx context.Context) (uint, bool, error) {
	version, dirty, err := p.migrate.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, ports.NewMigrationError(
			ports.MigrationErrCodeConnectionFailed,
			"failed to get migration version",
			0,
			err,
		)
	}

	// If no migrations have been applied, return version 0
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}

	return version, dirty, nil
}

// Status returns detailed migration status information
func (p *PostgresMigrationService) Status(ctx context.Context) (*ports.MigrationStatus, error) {
	version, dirty, err := p.Version(ctx)
	if err != nil {
		return nil, err
	}

	status := &ports.MigrationStatus{
		CurrentVersion:    version,
		Dirty:             dirty,
		AppliedMigrations: []ports.AppliedMigration{},
		PendingMigrations: []ports.PendingMigration{},
	}

	// Query applied migrations from schema_migrations table
	appliedMigrations, err := p.getAppliedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}
	status.AppliedMigrations = appliedMigrations
	status.TotalMigrations = uint(len(appliedMigrations))

	// Set last migration time if we have applied migrations
	if len(appliedMigrations) > 0 {
		status.LastMigrationTime = &appliedMigrations[len(appliedMigrations)-1].AppliedAt
	}

	return status, nil
}

// Migrate applies migrations up to a specific version
func (p *PostgresMigrationService) Migrate(ctx context.Context, targetVersion uint) error {
	err := p.migrate.Migrate(targetVersion)
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return ports.NewMigrationError(
			ports.MigrationErrCodeMigrationFailed,
			fmt.Sprintf("failed to migrate to version %d", targetVersion),
			targetVersion,
			err,
		)
	}
	return nil
}

// Force marks a migration version as completed without running it
func (p *PostgresMigrationService) Force(ctx context.Context, version uint) error {
	err := p.migrate.Force(int(version))
	if err != nil {
		return ports.NewMigrationError(
			ports.MigrationErrCodeMigrationFailed,
			fmt.Sprintf("failed to force migration to version %d", version),
			version,
			err,
		)
	}
	return nil
}

// Close closes the migration service and cleans up resources
func (p *PostgresMigrationService) Close() error {
	sourceErr, dbErr := p.migrate.Close()
	if sourceErr != nil {
		return fmt.Errorf("failed to close source: %w", sourceErr)
	}
	if dbErr != nil {
		return fmt.Errorf("failed to close database: %w", dbErr)
	}
	return nil
}

// getAppliedMigrations queries the schema_migrations table for applied migrations
func (p *PostgresMigrationService) getAppliedMigrations(ctx context.Context) ([]ports.AppliedMigration, error) {
	// Check if schema_migrations table exists
	var exists bool
	err := p.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'schema_migrations'
		)`).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check if schema_migrations table exists: %w", err)
	}

	if !exists {
		return []ports.AppliedMigration{}, nil
	}

	// Query applied migrations - golang-migrate uses a simple table with version and dirty fields
	rows, err := p.db.QueryContext(ctx, `
		SELECT version 
		FROM schema_migrations 
		WHERE dirty = false 
		ORDER BY version ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	var migrations []ports.AppliedMigration
	for rows.Next() {
		var version uint
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("failed to scan migration version: %w", err)
		}

		migrations = append(migrations, ports.AppliedMigration{
			Version:     version,
			Description: fmt.Sprintf("Migration %d", version),
			AppliedAt:   time.Now(), // golang-migrate doesn't store timestamp, so use current time
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migration rows: %w", err)
	}

	return migrations, nil
}
