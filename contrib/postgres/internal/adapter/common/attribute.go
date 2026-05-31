//go:build postgresql

package common

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Attribute, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresAttributeRepository(dbOps, tableName), nil
	})
}

// PostgresAttributeRepository implements attribute CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_attribute_active ON attribute(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_attribute_code ON attribute(code) - Search on code field
//   - CREATE INDEX idx_attribute_module ON attribute(module) - Filter by module
//   - CREATE INDEX idx_attribute_date_created ON attribute(date_created DESC) - Default sorting
type PostgresAttributeRepository struct {
	commonpb.UnimplementedAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresAttributeRepository creates a new PostgreSQL attribute repository
func NewPostgresAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) commonpb.AttributeDomainServiceServer {
	if tableName == "" {
		tableName = "attribute" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresAttributeRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateAttribute creates a new attribute using common PostgreSQL operations
func (r *PostgresAttributeRepository) CreateAttribute(ctx context.Context, req *commonpb.CreateAttributeRequest) (*commonpb.CreateAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("attribute data is required")
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
		return nil, fmt.Errorf("failed to create attribute: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	attribute := &commonpb.Attribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &commonpb.CreateAttributeResponse{
		Data:    []*commonpb.Attribute{attribute},
		Success: true,
	}, nil
}

// ReadAttribute retrieves an attribute using common PostgreSQL operations
func (r *PostgresAttributeRepository) ReadAttribute(ctx context.Context, req *commonpb.ReadAttributeRequest) (*commonpb.ReadAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("attribute ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read attribute: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	attribute := &commonpb.Attribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &commonpb.ReadAttributeResponse{
		Data:    []*commonpb.Attribute{attribute},
		Success: true,
	}, nil
}

// UpdateAttribute updates an attribute using common PostgreSQL operations
func (r *PostgresAttributeRepository) UpdateAttribute(ctx context.Context, req *commonpb.UpdateAttributeRequest) (*commonpb.UpdateAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("attribute ID is required")
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
		return nil, fmt.Errorf("failed to update attribute: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	attribute := &commonpb.Attribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &commonpb.UpdateAttributeResponse{
		Data:    []*commonpb.Attribute{attribute},
		Success: true,
	}, nil
}

// DeleteAttribute deletes an attribute using common PostgreSQL operations (soft delete)
func (r *PostgresAttributeRepository) DeleteAttribute(ctx context.Context, req *commonpb.DeleteAttributeRequest) (*commonpb.DeleteAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("attribute ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete attribute: %w", err)
	}

	return &commonpb.DeleteAttributeResponse{
		Success: true,
	}, nil
}

// ListAttributes lists attributes using common PostgreSQL operations
func (r *PostgresAttributeRepository) ListAttributes(ctx context.Context, req *commonpb.ListAttributesRequest) (*commonpb.ListAttributesResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list attributes: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var attributes []*commonpb.Attribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		attribute := &commonpb.Attribute{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attribute); err != nil {
			// Log error and continue with next item
			continue
		}
		attributes = append(attributes, attribute)
	}

	if attributes == nil {
		attributes = make([]*commonpb.Attribute, 0)
	}

	return &commonpb.ListAttributesResponse{
		Data:    attributes,
		Success: true,
	}, nil
}

// NewAttributeRepository creates a new PostgreSQL attribute repository (old-style constructor)
func NewAttributeRepository(db *sql.DB, tableName string) commonpb.AttributeDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresAttributeRepository(dbOps, tableName)
}
