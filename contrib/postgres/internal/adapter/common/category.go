//go:build postgresql

package common

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/shared/identity"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	categorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Category, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres category repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
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
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, category); err != nil {
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
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, category); err != nil {
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
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, category); err != nil {
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
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var categories []*categorypb.Category
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		category := &categorypb.Category{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, category); err != nil {
			continue
		}
		categories = append(categories, category)
	}

	return &categorypb.ListCategoriesResponse{
		Data: categories,
	}, nil
}

// GetCategoryListPageData returns all categories (active AND inactive) for the settings list page.
// This is intentionally NOT part of the CategoryDomainServiceServer interface — ListCategories
// is for dropdowns and always filters active=true. This method is for management UIs.
func (r *PostgresCategoryRepository) GetCategoryListPageData(ctx context.Context) ([]*categorypb.Category, error) {
	wsID := identity.Must(ctx).WorkspaceID

	var db *sql.DB
	if pgOps, ok := r.dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	if db == nil {
		return nil, fmt.Errorf("direct DB access unavailable for GetCategoryListPageData")
	}

	rows, err := db.QueryContext(ctx, `
		SELECT id, name, description, code, module, parent_id,
		       date_created, date_modified, active, workspace_id
		FROM category
		WHERE workspace_id = $1
		ORDER BY name ASC
	`, wsID)
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	var categories []*categorypb.Category
	for rows.Next() {
		var (
			id          string
			name        string
			description *string
			code        string
			module      string
			parentID    *string
			dateCreated *time.Time
			dateMod     *time.Time
			active      bool
			workspaceID string
		)
		if err := rows.Scan(&id, &name, &description, &code, &module, &parentID,
			&dateCreated, &dateMod, &active, &workspaceID); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		cat := &categorypb.Category{Id: id, Name: name, Code: code, Module: module, Active: active}
		if description != nil {
			cat.Description = *description
		}
		if parentID != nil {
			cat.ParentId = parentID
		}
		if dateCreated != nil {
			ts := dateCreated.UnixMilli()
			cat.DateCreated = &ts
			s := dateCreated.Format(time.RFC3339)
			cat.DateCreatedString = &s
		}
		if dateMod != nil {
			ts := dateMod.UnixMilli()
			cat.DateModified = &ts
			s := dateMod.Format(time.RFC3339)
			cat.DateModifiedString = &s
		}
		categories = append(categories, cat)
	}
	return categories, rows.Err()
}

// NewCategoryRepository creates a new PostgreSQL category repository (old-style constructor)
func NewCategoryRepository(db *sql.DB, tableName string) categorypb.CategoryDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresCategoryRepository(dbOps, tableName)
}
