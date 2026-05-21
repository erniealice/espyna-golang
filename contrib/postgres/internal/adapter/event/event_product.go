//go:build postgresql

package event

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	eventproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_product"
	"google.golang.org/protobuf/encoding/protojson"
)

// PostgresEventProductRepository implements event product CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_event_product_active ON event_product(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_event_product_event_id ON event_product(event_id) - Filter by event
//   - CREATE INDEX idx_event_product_product_id ON event_product(product_id) - Filter by product
//   - CREATE INDEX idx_event_product_date_created ON event_product(date_created DESC) - Default sorting
type PostgresEventProductRepository struct {
	eventproductpb.UnimplementedEventProductDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.EventProduct, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres event_product repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresEventProductRepository(dbOps, tableName), nil
	})
}

// NewPostgresEventProductRepository creates a new PostgreSQL event product repository
func NewPostgresEventProductRepository(dbOps interfaces.DatabaseOperation, tableName string) eventproductpb.EventProductDomainServiceServer {
	if tableName == "" {
		tableName = "event_product" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresEventProductRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateEventProduct creates a new event product using common PostgreSQL operations
func (r *PostgresEventProductRepository) CreateEventProduct(ctx context.Context, req *eventproductpb.CreateEventProductRequest) (*eventproductpb.CreateEventProductResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event product data is required")
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
		return nil, fmt.Errorf("failed to create event product: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventProduct := &eventproductpb.EventProduct{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventProduct); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventproductpb.CreateEventProductResponse{
		Data: []*eventproductpb.EventProduct{eventProduct},
	}, nil
}

// ReadEventProduct retrieves an event product using common PostgreSQL operations
func (r *PostgresEventProductRepository) ReadEventProduct(ctx context.Context, req *eventproductpb.ReadEventProductRequest) (*eventproductpb.ReadEventProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event product ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event product: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventProduct := &eventproductpb.EventProduct{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventProduct); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventproductpb.ReadEventProductResponse{
		Data: []*eventproductpb.EventProduct{eventProduct},
	}, nil
}

// UpdateEventProduct updates an event product using common PostgreSQL operations
func (r *PostgresEventProductRepository) UpdateEventProduct(ctx context.Context, req *eventproductpb.UpdateEventProductRequest) (*eventproductpb.UpdateEventProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event product ID is required")
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
		return nil, fmt.Errorf("failed to update event product: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventProduct := &eventproductpb.EventProduct{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventProduct); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventproductpb.UpdateEventProductResponse{
		Data: []*eventproductpb.EventProduct{eventProduct},
	}, nil
}

// DeleteEventProduct deletes an event product using common PostgreSQL operations
func (r *PostgresEventProductRepository) DeleteEventProduct(ctx context.Context, req *eventproductpb.DeleteEventProductRequest) (*eventproductpb.DeleteEventProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event product ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete event product: %w", err)
	}

	return &eventproductpb.DeleteEventProductResponse{
		Success: true,
	}, nil
}

// ListEventProducts lists event products using common PostgreSQL operations
func (r *PostgresEventProductRepository) ListEventProducts(ctx context.Context, req *eventproductpb.ListEventProductsRequest) (*eventproductpb.ListEventProductsResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list event products: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var eventProducts []*eventproductpb.EventProduct
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		eventProduct := &eventproductpb.EventProduct{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventProduct); err != nil {
			// Log error and continue with next item
			continue
		}
		eventProducts = append(eventProducts, eventProduct)
	}

	return &eventproductpb.ListEventProductsResponse{
		Data: eventProducts,
	}, nil
}

// NewEventProductRepository creates a new PostgreSQL event_product repository (old-style constructor)
func NewEventProductRepository(db *sql.DB, tableName string) eventproductpb.EventProductDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresEventProductRepository(dbOps, tableName)
}
