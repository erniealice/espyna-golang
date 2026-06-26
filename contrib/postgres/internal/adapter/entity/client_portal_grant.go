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
	clientportalgrantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_portal_grant"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ClientPortalGrant, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres client_portal_grant repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresClientPortalGrantRepository(dbOps, tableName), nil
	})
}

// PostgresClientPortalGrantRepository implements client portal grant CRUD operations using PostgreSQL.
type PostgresClientPortalGrantRepository struct {
	clientportalgrantpb.UnimplementedClientPortalGrantDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresClientPortalGrantRepository creates a new PostgreSQL client_portal_grant repository.
func NewPostgresClientPortalGrantRepository(dbOps interfaces.DatabaseOperation, tableName string) clientportalgrantpb.ClientPortalGrantDomainServiceServer {
	if tableName == "" {
		tableName = "client_portal_grant"
	}
	return &PostgresClientPortalGrantRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateClientPortalGrant creates a new client portal grant record.
func (r *PostgresClientPortalGrantRepository) CreateClientPortalGrant(ctx context.Context, req *clientportalgrantpb.CreateClientPortalGrantRequest) (*clientportalgrantpb.CreateClientPortalGrantResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("client_portal_grant data is required")
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
		return nil, fmt.Errorf("failed to create client_portal_grant: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	grant := &clientportalgrantpb.ClientPortalGrant{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, grant); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &clientportalgrantpb.CreateClientPortalGrantResponse{Data: []*clientportalgrantpb.ClientPortalGrant{grant}}, nil
}

// ReadClientPortalGrant retrieves a client portal grant by ID.
func (r *PostgresClientPortalGrantRepository) ReadClientPortalGrant(ctx context.Context, req *clientportalgrantpb.ReadClientPortalGrantRequest) (*clientportalgrantpb.ReadClientPortalGrantResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client_portal_grant ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read client_portal_grant: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	grant := &clientportalgrantpb.ClientPortalGrant{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, grant); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &clientportalgrantpb.ReadClientPortalGrantResponse{Data: []*clientportalgrantpb.ClientPortalGrant{grant}}, nil
}

// UpdateClientPortalGrant updates an existing client portal grant record.
func (r *PostgresClientPortalGrantRepository) UpdateClientPortalGrant(ctx context.Context, req *clientportalgrantpb.UpdateClientPortalGrantRequest) (*clientportalgrantpb.UpdateClientPortalGrantResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client_portal_grant ID is required")
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
		return nil, fmt.Errorf("failed to update client_portal_grant: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	grant := &clientportalgrantpb.ClientPortalGrant{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, grant); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &clientportalgrantpb.UpdateClientPortalGrantResponse{Data: []*clientportalgrantpb.ClientPortalGrant{grant}}, nil
}

// DeleteClientPortalGrant soft-deletes a client portal grant.
func (r *PostgresClientPortalGrantRepository) DeleteClientPortalGrant(ctx context.Context, req *clientportalgrantpb.DeleteClientPortalGrantRequest) (*clientportalgrantpb.DeleteClientPortalGrantResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client_portal_grant ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete client_portal_grant: %w", err)
	}
	return &clientportalgrantpb.DeleteClientPortalGrantResponse{Success: true}, nil
}

// ListClientPortalGrants lists client portal grants matching optional filters.
func (r *PostgresClientPortalGrantRepository) ListClientPortalGrants(ctx context.Context, req *clientportalgrantpb.ListClientPortalGrantsRequest) (*clientportalgrantpb.ListClientPortalGrantsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list client_portal_grants: %w", err)
	}
	var grants []*clientportalgrantpb.ClientPortalGrant
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		grant := &clientportalgrantpb.ClientPortalGrant{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, grant); err != nil {
			continue
		}
		grants = append(grants, grant)
	}
	return &clientportalgrantpb.ListClientPortalGrantsResponse{Data: grants}, nil
}
