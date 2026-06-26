//go:build postgresql

package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	activitytemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity_template"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ActivityTemplate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres activity_template repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresActivityTemplateRepository(dbOps, tableName), nil
	})
}

// PostgresActivityTemplateRepository implements activity_template CRUD operations using PostgreSQL.
type PostgresActivityTemplateRepository struct {
	activitytemplatepb.UnimplementedActivityTemplateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresActivityTemplateRepository creates a new PostgreSQL activity_template repository
func NewPostgresActivityTemplateRepository(dbOps interfaces.DatabaseOperation, tableName string) activitytemplatepb.ActivityTemplateDomainServiceServer {
	if tableName == "" {
		tableName = "activity_template" // default fallback
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresActivityTemplateRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateActivityTemplate creates a new activity_template using common PostgreSQL operations
func (r *PostgresActivityTemplateRepository) CreateActivityTemplate(ctx context.Context, req *activitytemplatepb.CreateActivityTemplateRequest) (*activitytemplatepb.CreateActivityTemplateResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("activity_template data is required")
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
		return nil, fmt.Errorf("failed to create activity_template: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	activityTemplate := &activitytemplatepb.ActivityTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, activityTemplate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &activitytemplatepb.CreateActivityTemplateResponse{
		Data:    []*activitytemplatepb.ActivityTemplate{activityTemplate},
		Success: true,
	}, nil
}

// ReadActivityTemplate retrieves an activity_template using common PostgreSQL operations
func (r *PostgresActivityTemplateRepository) ReadActivityTemplate(ctx context.Context, req *activitytemplatepb.ReadActivityTemplateRequest) (*activitytemplatepb.ReadActivityTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("activity_template ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read activity_template: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	activityTemplate := &activitytemplatepb.ActivityTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, activityTemplate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &activitytemplatepb.ReadActivityTemplateResponse{
		Data:    []*activitytemplatepb.ActivityTemplate{activityTemplate},
		Success: true,
	}, nil
}

// UpdateActivityTemplate updates an activity_template using common PostgreSQL operations
func (r *PostgresActivityTemplateRepository) UpdateActivityTemplate(ctx context.Context, req *activitytemplatepb.UpdateActivityTemplateRequest) (*activitytemplatepb.UpdateActivityTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("activity_template ID is required")
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
		return nil, fmt.Errorf("failed to update activity_template: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	activityTemplate := &activitytemplatepb.ActivityTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, activityTemplate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &activitytemplatepb.UpdateActivityTemplateResponse{
		Data:    []*activitytemplatepb.ActivityTemplate{activityTemplate},
		Success: true,
	}, nil
}

// DeleteActivityTemplate deletes an activity_template using common PostgreSQL operations (soft delete)
func (r *PostgresActivityTemplateRepository) DeleteActivityTemplate(ctx context.Context, req *activitytemplatepb.DeleteActivityTemplateRequest) (*activitytemplatepb.DeleteActivityTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("activity_template ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete activity_template: %w", err)
	}

	return &activitytemplatepb.DeleteActivityTemplateResponse{
		Success: true,
	}, nil
}

// ListActivityTemplates lists activity_templates using common PostgreSQL operations
func (r *PostgresActivityTemplateRepository) ListActivityTemplates(ctx context.Context, req *activitytemplatepb.ListActivityTemplatesRequest) (*activitytemplatepb.ListActivityTemplatesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list activity_templates: %w", err)
	}

	var activityTemplates []*activitytemplatepb.ActivityTemplate
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		activityTemplate := &activitytemplatepb.ActivityTemplate{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, activityTemplate); err != nil {
			continue
		}
		activityTemplates = append(activityTemplates, activityTemplate)
	}

	if activityTemplates == nil {
		activityTemplates = make([]*activitytemplatepb.ActivityTemplate, 0)
	}

	return &activitytemplatepb.ListActivityTemplatesResponse{
		Data:    activityTemplates,
		Success: true,
	}, nil
}

// GetActivityTemplateListPageData retrieves activity_templates with basic pagination via List.
func (r *PostgresActivityTemplateRepository) GetActivityTemplateListPageData(ctx context.Context, req *activitytemplatepb.GetActivityTemplateListPageDataRequest) (*activitytemplatepb.GetActivityTemplateListPageDataResponse, error) {
	listReq := &activitytemplatepb.ListActivityTemplatesRequest{}
	listResp, err := r.ListActivityTemplates(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity_template list page data: %w", err)
	}

	return &activitytemplatepb.GetActivityTemplateListPageDataResponse{
		ActivityTemplateList: listResp.Data,
		Success:              true,
	}, nil
}

// GetActivityTemplateItemPageData retrieves a single activity_template via Read.
func (r *PostgresActivityTemplateRepository) GetActivityTemplateItemPageData(ctx context.Context, req *activitytemplatepb.GetActivityTemplateItemPageDataRequest) (*activitytemplatepb.GetActivityTemplateItemPageDataResponse, error) {
	if req.ActivityTemplateId == "" {
		return nil, fmt.Errorf("activity_template ID is required")
	}

	readReq := &activitytemplatepb.ReadActivityTemplateRequest{
		Data: &activitytemplatepb.ActivityTemplate{Id: req.ActivityTemplateId},
	}
	readResp, err := r.ReadActivityTemplate(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity_template item page data: %w", err)
	}

	if len(readResp.Data) == 0 {
		return nil, fmt.Errorf("activity_template not found")
	}

	return &activitytemplatepb.GetActivityTemplateItemPageDataResponse{
		ActivityTemplate: readResp.Data[0],
		Success:          true,
	}, nil
}

// NewActivityTemplateRepository creates a new PostgreSQL activity_template repository (old-style constructor)
func NewActivityTemplateRepository(db *sql.DB, tableName string) activitytemplatepb.ActivityTemplateDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresActivityTemplateRepository(dbOps, tableName)
}
