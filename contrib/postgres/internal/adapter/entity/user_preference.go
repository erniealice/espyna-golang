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
	userpreferencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user_preference"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.UserPreference, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres user_preference repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresUserPreferenceRepository(dbOps, tableName), nil
	})
}

// PostgresUserPreferenceRepository implements user preference CRUD operations using PostgreSQL.
// One row per (user_id, workspace_id) pair; stores appearance, locale, and notification preferences.
type PostgresUserPreferenceRepository struct {
	userpreferencepb.UnimplementedUserPreferenceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresUserPreferenceRepository creates a new PostgreSQL user_preference repository.
func NewPostgresUserPreferenceRepository(dbOps interfaces.DatabaseOperation, tableName string) userpreferencepb.UserPreferenceDomainServiceServer {
	if tableName == "" {
		tableName = "user_preference"
	}
	return &PostgresUserPreferenceRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateUserPreference creates a new user preference record.
func (r *PostgresUserPreferenceRepository) CreateUserPreference(ctx context.Context, req *userpreferencepb.CreateUserPreferenceRequest) (*userpreferencepb.CreateUserPreferenceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("user_preference data is required")
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
		return nil, fmt.Errorf("failed to create user_preference: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pref := &userpreferencepb.UserPreference{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pref); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &userpreferencepb.CreateUserPreferenceResponse{Data: []*userpreferencepb.UserPreference{pref}}, nil
}

// ReadUserPreference retrieves a user preference by ID.
func (r *PostgresUserPreferenceRepository) ReadUserPreference(ctx context.Context, req *userpreferencepb.ReadUserPreferenceRequest) (*userpreferencepb.ReadUserPreferenceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("user_preference ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read user_preference: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pref := &userpreferencepb.UserPreference{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pref); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &userpreferencepb.ReadUserPreferenceResponse{Data: []*userpreferencepb.UserPreference{pref}}, nil
}

// UpdateUserPreference updates an existing user preference record.
func (r *PostgresUserPreferenceRepository) UpdateUserPreference(ctx context.Context, req *userpreferencepb.UpdateUserPreferenceRequest) (*userpreferencepb.UpdateUserPreferenceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("user_preference ID is required")
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
		return nil, fmt.Errorf("failed to update user_preference: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pref := &userpreferencepb.UserPreference{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pref); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &userpreferencepb.UpdateUserPreferenceResponse{Data: []*userpreferencepb.UserPreference{pref}}, nil
}

// DeleteUserPreference soft-deletes a user preference record.
func (r *PostgresUserPreferenceRepository) DeleteUserPreference(ctx context.Context, req *userpreferencepb.DeleteUserPreferenceRequest) (*userpreferencepb.DeleteUserPreferenceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("user_preference ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete user_preference: %w", err)
	}
	return &userpreferencepb.DeleteUserPreferenceResponse{Success: true}, nil
}

// ListUserPreferences lists user preferences matching optional filters.
func (r *PostgresUserPreferenceRepository) ListUserPreferences(ctx context.Context, req *userpreferencepb.ListUserPreferencesRequest) (*userpreferencepb.ListUserPreferencesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list user_preferences: %w", err)
	}
	var prefs []*userpreferencepb.UserPreference
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		pref := &userpreferencepb.UserPreference{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pref); err != nil {
			continue
		}
		prefs = append(prefs, pref)
	}
	return &userpreferencepb.ListUserPreferencesResponse{Data: prefs}, nil
}
