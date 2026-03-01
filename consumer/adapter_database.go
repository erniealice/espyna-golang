package consumer

import (
	"context"
	"fmt"

	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
)

/*
 ESPYNA CONSUMER APP - Technology-Agnostic Database Adapter

Provides direct CRUD access to database collections without requiring
the full use cases/provider/repository initialization chain.

This adapter works with ANY database backend (Firestore, Postgres, Mock)
based on your CONFIG_DATABASE_PROVIDER environment variable.

Usage:

	// Option 1: Get from container (recommended)
	container := consumer.NewContainerFromEnv()
	adapter := consumer.NewDatabaseAdapterFromContainer(container)
	defer adapter.Close()

	// Option 2: Create standalone (legacy compatibility)
	adapter, err := consumer.NewDatabaseAdapterFromEnv(ctx)
	if err != nil {
	    log.Fatal(err)
	}
	defer adapter.Close()

	// Generic CRUD on any collection
	doc, err := adapter.Read(ctx, "payments", "doc-id")
	docs, err := adapter.List(ctx, "licenses", nil)
	result, err := adapter.Create(ctx, "logs", map[string]any{"message": "hello"})
*/

// DatabaseAdapter provides technology-agnostic CRUD access to database collections.
// It wraps the DatabaseOperation interface and works with Firestore, Postgres, or Mock.
type DatabaseAdapter struct {
	ops       interfaces.DatabaseOperation
	container *Container
}

// NewDatabaseAdapterFromContainer creates a DatabaseAdapter from an existing container.
// This is the recommended way to create the adapter as it reuses the container's connection.
func NewDatabaseAdapterFromContainer(container *Container) *DatabaseAdapter {
	if container == nil {
		return nil
	}

	ops := container.GetDatabaseOperations()
	if ops == nil {
		return nil
	}

	dbOps, ok := ops.(interfaces.DatabaseOperation)
	if !ok {
		return nil
	}

	return &DatabaseAdapter{
		ops:       dbOps,
		container: container,
	}
}

// NewDatabaseAdapterFromEnv creates a standalone DatabaseAdapter from environment variables.
// This creates its own container internally - use NewDatabaseAdapterFromContainer if you
// already have a container to avoid creating duplicate connections.
func NewDatabaseAdapterFromEnv(ctx context.Context) (*DatabaseAdapter, error) {
	container, err := NewContainerFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to create container from environment: %w", err)
	}

	adapter := NewDatabaseAdapterFromContainer(container)
	if adapter == nil {
		container.Close()
		return nil, fmt.Errorf("failed to create database adapter from container")
	}

	return adapter, nil
}

// Close closes the database adapter.
// Note: If created from container, this does NOT close the container.
// The caller is responsible for closing the container separately.
func (a *DatabaseAdapter) Close() error {
	// Don't close the container here - let the caller manage it
	return nil
}

// GetOperations returns the underlying DatabaseOperation interface for advanced usage.
func (a *DatabaseAdapter) GetOperations() interfaces.DatabaseOperation {
	return a.ops
}

// --- Generic CRUD Operations ---

// Create creates a new document in the specified collection.
// Automatically sets: id (if not provided), active=true, date_created, date_modified
func (a *DatabaseAdapter) Create(ctx context.Context, collection string, data map[string]any) (map[string]any, error) {
	if a.ops == nil {
		return nil, fmt.Errorf("database operations not initialized")
	}
	return a.ops.Create(ctx, collection, data)
}

// Read retrieves a document by ID from the specified collection.
func (a *DatabaseAdapter) Read(ctx context.Context, collection string, id string) (map[string]any, error) {
	if a.ops == nil {
		return nil, fmt.Errorf("database operations not initialized")
	}
	return a.ops.Read(ctx, collection, id)
}

// Update updates an existing document in the specified collection.
// Automatically updates date_modified and preserves date_created.
func (a *DatabaseAdapter) Update(ctx context.Context, collection string, id string, data map[string]any) (map[string]any, error) {
	if a.ops == nil {
		return nil, fmt.Errorf("database operations not initialized")
	}
	return a.ops.Update(ctx, collection, id, data)
}

// Delete performs a soft delete (sets active=false) on a document.
func (a *DatabaseAdapter) Delete(ctx context.Context, collection string, id string) error {
	if a.ops == nil {
		return fmt.Errorf("database operations not initialized")
	}
	return a.ops.Delete(ctx, collection, id)
}

// HardDelete permanently removes a document from the collection.
func (a *DatabaseAdapter) HardDelete(ctx context.Context, collection string, id string) error {
	if a.ops == nil {
		return fmt.Errorf("database operations not initialized")
	}
	return a.ops.HardDelete(ctx, collection, id)
}

// List retrieves documents from the specified collection with optional parameters.
// Supports filtering, sorting, and pagination via ListParams.
// Automatically filters by active=true.
func (a *DatabaseAdapter) List(ctx context.Context, collection string, params *interfaces.ListParams) (*interfaces.ListResult, error) {
	if a.ops == nil {
		return nil, fmt.Errorf("database operations not initialized")
	}
	return a.ops.List(ctx, collection, params)
}

// ListSimple retrieves all active documents from a collection without parameters.
// This is a convenience method for simple listing without filters/pagination.
func (a *DatabaseAdapter) ListSimple(ctx context.Context, collection string) ([]map[string]any, error) {
	result, err := a.List(ctx, collection, nil)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

// Query executes a structured query against the collection.
func (a *DatabaseAdapter) Query(ctx context.Context, collection string, queryBuilder interfaces.QueryBuilder) ([]map[string]any, error) {
	if a.ops == nil {
		return nil, fmt.Errorf("database operations not initialized")
	}
	return a.ops.Query(ctx, collection, queryBuilder)
}

// QueryOne executes a structured query and returns the first result.
func (a *DatabaseAdapter) QueryOne(ctx context.Context, collection string, queryBuilder interfaces.QueryBuilder) (map[string]any, error) {
	if a.ops == nil {
		return nil, fmt.Errorf("database operations not initialized")
	}
	return a.ops.QueryOne(ctx, collection, queryBuilder)
}

// --- Re-export types for consumer convenience ---

// ListParams re-exports the ListParams type for consumer convenience
type ListParams = interfaces.ListParams

// ListResult re-exports the ListResult type for consumer convenience
type ListResult = interfaces.ListResult

// QueryBuilder re-exports the QueryBuilder interface for consumer convenience
type QueryBuilder = interfaces.QueryBuilder
