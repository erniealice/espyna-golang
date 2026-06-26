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
	workflowtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow_template"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.WorkflowTemplate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres workflow_template repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresWorkflowTemplateRepository(dbOps, tableName), nil
	})
}

// PostgresWorkflowTemplateRepository implements workflow_template CRUD operations using PostgreSQL.
type PostgresWorkflowTemplateRepository struct {
	workflowtemplatepb.UnimplementedWorkflowTemplateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresWorkflowTemplateRepository creates a new PostgreSQL workflow_template repository
func NewPostgresWorkflowTemplateRepository(dbOps interfaces.DatabaseOperation, tableName string) workflowtemplatepb.WorkflowTemplateDomainServiceServer {
	if tableName == "" {
		tableName = "workflow_template" // default fallback
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresWorkflowTemplateRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateWorkflowTemplate creates a new workflow_template using common PostgreSQL operations
func (r *PostgresWorkflowTemplateRepository) CreateWorkflowTemplate(ctx context.Context, req *workflowtemplatepb.CreateWorkflowTemplateRequest) (*workflowtemplatepb.CreateWorkflowTemplateResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("workflow_template data is required")
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
		return nil, fmt.Errorf("failed to create workflow_template: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workflowTemplate := &workflowtemplatepb.WorkflowTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workflowTemplate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workflowtemplatepb.CreateWorkflowTemplateResponse{
		Data: []*workflowtemplatepb.WorkflowTemplate{workflowTemplate},
	}, nil
}

// ReadWorkflowTemplate retrieves a workflow_template using common PostgreSQL operations
func (r *PostgresWorkflowTemplateRepository) ReadWorkflowTemplate(ctx context.Context, req *workflowtemplatepb.ReadWorkflowTemplateRequest) (*workflowtemplatepb.ReadWorkflowTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workflow_template ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow_template: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workflowTemplate := &workflowtemplatepb.WorkflowTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workflowTemplate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workflowtemplatepb.ReadWorkflowTemplateResponse{
		Data: []*workflowtemplatepb.WorkflowTemplate{workflowTemplate},
	}, nil
}

// UpdateWorkflowTemplate updates a workflow_template using common PostgreSQL operations
func (r *PostgresWorkflowTemplateRepository) UpdateWorkflowTemplate(ctx context.Context, req *workflowtemplatepb.UpdateWorkflowTemplateRequest) (*workflowtemplatepb.UpdateWorkflowTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workflow_template ID is required")
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
		return nil, fmt.Errorf("failed to update workflow_template: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workflowTemplate := &workflowtemplatepb.WorkflowTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workflowTemplate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workflowtemplatepb.UpdateWorkflowTemplateResponse{
		Data: []*workflowtemplatepb.WorkflowTemplate{workflowTemplate},
	}, nil
}

// DeleteWorkflowTemplate deletes a workflow_template using common PostgreSQL operations (soft delete)
func (r *PostgresWorkflowTemplateRepository) DeleteWorkflowTemplate(ctx context.Context, req *workflowtemplatepb.DeleteWorkflowTemplateRequest) (*workflowtemplatepb.DeleteWorkflowTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workflow_template ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete workflow_template: %w", err)
	}

	return &workflowtemplatepb.DeleteWorkflowTemplateResponse{
		Success: true,
	}, nil
}

// ListWorkflowTemplates lists workflow_templates using common PostgreSQL operations
func (r *PostgresWorkflowTemplateRepository) ListWorkflowTemplates(ctx context.Context, req *workflowtemplatepb.ListWorkflowTemplatesRequest) (*workflowtemplatepb.ListWorkflowTemplatesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow_templates: %w", err)
	}

	var workflowTemplates []*workflowtemplatepb.WorkflowTemplate
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		workflowTemplate := &workflowtemplatepb.WorkflowTemplate{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workflowTemplate); err != nil {
			continue
		}
		workflowTemplates = append(workflowTemplates, workflowTemplate)
	}

	if workflowTemplates == nil {
		workflowTemplates = make([]*workflowtemplatepb.WorkflowTemplate, 0)
	}

	return &workflowtemplatepb.ListWorkflowTemplatesResponse{
		Data: workflowTemplates,
	}, nil
}

// GetWorkflowTemplateListPageData retrieves workflow_templates with basic pagination via List.
func (r *PostgresWorkflowTemplateRepository) GetWorkflowTemplateListPageData(ctx context.Context, req *workflowtemplatepb.GetWorkflowTemplateListPageDataRequest) (*workflowtemplatepb.GetWorkflowTemplateListPageDataResponse, error) {
	listReq := &workflowtemplatepb.ListWorkflowTemplatesRequest{}
	listResp, err := r.ListWorkflowTemplates(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow_template list page data: %w", err)
	}

	return &workflowtemplatepb.GetWorkflowTemplateListPageDataResponse{
		WorkflowTemplateList: listResp.Data,
		Success:              true,
	}, nil
}

// GetWorkflowTemplateItemPageData retrieves a single workflow_template via Read.
func (r *PostgresWorkflowTemplateRepository) GetWorkflowTemplateItemPageData(ctx context.Context, req *workflowtemplatepb.GetWorkflowTemplateItemPageDataRequest) (*workflowtemplatepb.GetWorkflowTemplateItemPageDataResponse, error) {
	if req.WorkflowTemplateId == "" {
		return nil, fmt.Errorf("workflow_template ID is required")
	}

	readReq := &workflowtemplatepb.ReadWorkflowTemplateRequest{
		Data: &workflowtemplatepb.WorkflowTemplate{Id: req.WorkflowTemplateId},
	}
	readResp, err := r.ReadWorkflowTemplate(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow_template item page data: %w", err)
	}

	if len(readResp.Data) == 0 {
		return nil, fmt.Errorf("workflow_template not found")
	}

	return &workflowtemplatepb.GetWorkflowTemplateItemPageDataResponse{
		WorkflowTemplate: readResp.Data[0],
		Success:          true,
	}, nil
}

// NewWorkflowTemplateRepository creates a new PostgreSQL workflow_template repository (old-style constructor)
func NewWorkflowTemplateRepository(db *sql.DB, tableName string) workflowtemplatepb.WorkflowTemplateDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresWorkflowTemplateRepository(dbOps, tableName)
}
