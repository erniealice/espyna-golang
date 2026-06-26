//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	supplierportalgrantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_portal_grant"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SupplierPortalGrant, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres supplier_portal_grant repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSupplierPortalGrantRepository(dbOps, tableName), nil
	})
}

// PostgresSupplierPortalGrantRepository implements supplier portal grant CRUD operations using PostgreSQL.
type PostgresSupplierPortalGrantRepository struct {
	supplierportalgrantpb.UnimplementedSupplierPortalGrantDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresSupplierPortalGrantRepository creates a new PostgreSQL supplier_portal_grant repository.
func NewPostgresSupplierPortalGrantRepository(dbOps interfaces.DatabaseOperation, tableName string) supplierportalgrantpb.SupplierPortalGrantDomainServiceServer {
	if tableName == "" {
		tableName = "supplier_portal_grant"
	}
	return &PostgresSupplierPortalGrantRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateSupplierPortalGrant creates a new supplier portal grant record.
func (r *PostgresSupplierPortalGrantRepository) CreateSupplierPortalGrant(ctx context.Context, req *supplierportalgrantpb.CreateSupplierPortalGrantRequest) (*supplierportalgrantpb.CreateSupplierPortalGrantResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("supplier_portal_grant data is required")
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
		return nil, fmt.Errorf("failed to create supplier_portal_grant: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	grant := &supplierportalgrantpb.SupplierPortalGrant{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, grant); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &supplierportalgrantpb.CreateSupplierPortalGrantResponse{Data: []*supplierportalgrantpb.SupplierPortalGrant{grant}}, nil
}

// ReadSupplierPortalGrant retrieves a supplier portal grant by ID.
func (r *PostgresSupplierPortalGrantRepository) ReadSupplierPortalGrant(ctx context.Context, req *supplierportalgrantpb.ReadSupplierPortalGrantRequest) (*supplierportalgrantpb.ReadSupplierPortalGrantResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier_portal_grant ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_portal_grant: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	grant := &supplierportalgrantpb.SupplierPortalGrant{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, grant); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &supplierportalgrantpb.ReadSupplierPortalGrantResponse{Data: []*supplierportalgrantpb.SupplierPortalGrant{grant}}, nil
}

// UpdateSupplierPortalGrant updates an existing supplier portal grant record.
func (r *PostgresSupplierPortalGrantRepository) UpdateSupplierPortalGrant(ctx context.Context, req *supplierportalgrantpb.UpdateSupplierPortalGrantRequest) (*supplierportalgrantpb.UpdateSupplierPortalGrantResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier_portal_grant ID is required")
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
		return nil, fmt.Errorf("failed to update supplier_portal_grant: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	grant := &supplierportalgrantpb.SupplierPortalGrant{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, grant); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &supplierportalgrantpb.UpdateSupplierPortalGrantResponse{Data: []*supplierportalgrantpb.SupplierPortalGrant{grant}}, nil
}

// DeleteSupplierPortalGrant soft-deletes a supplier portal grant.
func (r *PostgresSupplierPortalGrantRepository) DeleteSupplierPortalGrant(ctx context.Context, req *supplierportalgrantpb.DeleteSupplierPortalGrantRequest) (*supplierportalgrantpb.DeleteSupplierPortalGrantResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier_portal_grant ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete supplier_portal_grant: %w", err)
	}
	return &supplierportalgrantpb.DeleteSupplierPortalGrantResponse{Success: true}, nil
}

// ListSupplierPortalGrants lists supplier portal grants matching optional filters.
func (r *PostgresSupplierPortalGrantRepository) ListSupplierPortalGrants(ctx context.Context, req *supplierportalgrantpb.ListSupplierPortalGrantsRequest) (*supplierportalgrantpb.ListSupplierPortalGrantsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list supplier_portal_grants: %w", err)
	}
	var grants []*supplierportalgrantpb.SupplierPortalGrant
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		grant := &supplierportalgrantpb.SupplierPortalGrant{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, grant); err != nil {
			continue
		}
		grants = append(grants, grant)
	}
	return &supplierportalgrantpb.ListSupplierPortalGrantsResponse{Data: grants}, nil
}
