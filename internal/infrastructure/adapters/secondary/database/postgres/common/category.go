//go:build postgres

package common

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	categorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "category", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres category repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresCategoryRepository(dbOps, tableName), nil
	})
}

// PostgresCategoryRepository implements category CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_category_active ON category(active) WHERE active = true - Filter active categories
//   - CREATE INDEX idx_category_name ON category(name) - Search on name field
//   - CREATE INDEX idx_category_name_trgm ON category USING gin(name gin_trgm_ops) - Fuzzy search support
//   - CREATE INDEX idx_category_code ON category(code) - Lookup by code
//   - CREATE INDEX idx_category_module ON category(module) - Filter by module
//   - CREATE INDEX idx_category_parent_id ON category(parent_id) - Hierarchical queries
//   - CREATE INDEX idx_category_date_created ON category(date_created DESC) - Default sorting
type PostgresCategoryRepository struct {
	categorypb.UnimplementedCategoryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresCategoryRepository creates a new PostgreSQL category repository
func NewPostgresCategoryRepository(dbOps interfaces.DatabaseOperation, tableName string) categorypb.CategoryDomainServiceServer {
	if tableName == "" {
		tableName = "category" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresCategoryRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateCategory creates a new category using common PostgreSQL operations
func (r *PostgresCategoryRepository) CreateCategory(ctx context.Context, req *categorypb.CreateCategoryRequest) (*categorypb.CreateCategoryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("category data is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	category := &categorypb.Category{}
	if err := protojson.Unmarshal(resultJSON, category); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &categorypb.CreateCategoryResponse{
		Data: []*categorypb.Category{category},
	}, nil
}

// ReadCategory retrieves a category using common PostgreSQL operations
func (r *PostgresCategoryRepository) ReadCategory(ctx context.Context, req *categorypb.ReadCategoryRequest) (*categorypb.ReadCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("category ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read category: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	category := &categorypb.Category{}
	if err := protojson.Unmarshal(resultJSON, category); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &categorypb.ReadCategoryResponse{
		Data: []*categorypb.Category{category},
	}, nil
}

// UpdateCategory updates a category using common PostgreSQL operations
func (r *PostgresCategoryRepository) UpdateCategory(ctx context.Context, req *categorypb.UpdateCategoryRequest) (*categorypb.UpdateCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("category ID is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	category := &categorypb.Category{}
	if err := protojson.Unmarshal(resultJSON, category); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &categorypb.UpdateCategoryResponse{
		Data: []*categorypb.Category{category},
	}, nil
}

// DeleteCategory deletes a category using common PostgreSQL operations
func (r *PostgresCategoryRepository) DeleteCategory(ctx context.Context, req *categorypb.DeleteCategoryRequest) (*categorypb.DeleteCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("category ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete category: %w", err)
	}

	return &categorypb.DeleteCategoryResponse{
		Success: true,
	}, nil
}

// ListCategories lists categories using common PostgreSQL operations
func (r *PostgresCategoryRepository) ListCategories(ctx context.Context, req *categorypb.ListCategoriesRequest) (*categorypb.ListCategoriesResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var categories []*categorypb.Category
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		category := &categorypb.Category{}
		if err := protojson.Unmarshal(resultJSON, category); err != nil {
			// Log error and continue with next item
			continue
		}
		categories = append(categories, category)
	}

	return &categorypb.ListCategoriesResponse{
		Data: categories,
	}, nil
}

// NewCategoryRepository creates a new PostgreSQL category repository (old-style constructor)
func NewCategoryRepository(db *sql.DB, tableName string) categorypb.CategoryDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresCategoryRepository(dbOps, tableName)
}
