//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	delegatesupplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_supplier"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.DelegateSupplier, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres delegate_supplier repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresDelegateSupplierRepository(dbOps, tableName), nil
	})
}

// PostgresDelegateSupplierRepository implements delegate supplier CRUD operations using PostgreSQL.
// This entity mirrors DelegateClient: a Delegate user acts ON BEHALF OF a specific Supplier.
type PostgresDelegateSupplierRepository struct {
	delegatesupplierpb.UnimplementedDelegateSupplierDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresDelegateSupplierRepository creates a new PostgreSQL delegate_supplier repository.
func NewPostgresDelegateSupplierRepository(dbOps interfaces.DatabaseOperation, tableName string) delegatesupplierpb.DelegateSupplierDomainServiceServer {
	if tableName == "" {
		tableName = "delegate_supplier"
	}
	return &PostgresDelegateSupplierRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateDelegateSupplier creates a new delegate supplier record.
func (r *PostgresDelegateSupplierRepository) CreateDelegateSupplier(ctx context.Context, req *delegatesupplierpb.CreateDelegateSupplierRequest) (*delegatesupplierpb.CreateDelegateSupplierResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("delegate_supplier data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create delegate_supplier: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ds := &delegatesupplierpb.DelegateSupplier{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &delegatesupplierpb.CreateDelegateSupplierResponse{Data: []*delegatesupplierpb.DelegateSupplier{ds}}, nil
}

// ReadDelegateSupplier retrieves a delegate supplier by ID.
func (r *PostgresDelegateSupplierRepository) ReadDelegateSupplier(ctx context.Context, req *delegatesupplierpb.ReadDelegateSupplierRequest) (*delegatesupplierpb.ReadDelegateSupplierResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate_supplier ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read delegate_supplier: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ds := &delegatesupplierpb.DelegateSupplier{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &delegatesupplierpb.ReadDelegateSupplierResponse{Data: []*delegatesupplierpb.DelegateSupplier{ds}}, nil
}

// UpdateDelegateSupplier updates an existing delegate supplier record.
func (r *PostgresDelegateSupplierRepository) UpdateDelegateSupplier(ctx context.Context, req *delegatesupplierpb.UpdateDelegateSupplierRequest) (*delegatesupplierpb.UpdateDelegateSupplierResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate_supplier ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update delegate_supplier: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ds := &delegatesupplierpb.DelegateSupplier{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &delegatesupplierpb.UpdateDelegateSupplierResponse{Data: []*delegatesupplierpb.DelegateSupplier{ds}}, nil
}

// DeleteDelegateSupplier soft-deletes a delegate supplier.
func (r *PostgresDelegateSupplierRepository) DeleteDelegateSupplier(ctx context.Context, req *delegatesupplierpb.DeleteDelegateSupplierRequest) (*delegatesupplierpb.DeleteDelegateSupplierResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate_supplier ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete delegate_supplier: %w", err)
	}
	return &delegatesupplierpb.DeleteDelegateSupplierResponse{Success: true}, nil
}

// ListDelegateSuppliers lists delegate suppliers matching optional filters.
func (r *PostgresDelegateSupplierRepository) ListDelegateSuppliers(ctx context.Context, req *delegatesupplierpb.ListDelegateSuppliersRequest) (*delegatesupplierpb.ListDelegateSuppliersResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list delegate_suppliers: %w", err)
	}
	var items []*delegatesupplierpb.DelegateSupplier
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		ds := &delegatesupplierpb.DelegateSupplier{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ds); err != nil {
			continue
		}
		items = append(items, ds)
	}
	return &delegatesupplierpb.ListDelegateSuppliersResponse{Data: items}, nil
}
