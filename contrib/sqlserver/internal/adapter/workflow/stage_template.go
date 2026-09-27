//go:build sqlserver

package workflow

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	stagetemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage_template"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.StageTemplate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver stage_template repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerStageTemplateRepository(dbOps, tableName), nil
	})
}

// SQLServerStageTemplateRepository implements stage_template CRUD operations using SQLServer.
type SQLServerStageTemplateRepository struct {
	stagetemplatepb.UnimplementedStageTemplateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerStageTemplateRepository creates a new SQLServer stage_template repository
func NewSQLServerStageTemplateRepository(dbOps interfaces.DatabaseOperation, tableName string) stagetemplatepb.StageTemplateDomainServiceServer {
	if tableName == "" {
		tableName = "stage_template" // default fallback
	}

	return &SQLServerStageTemplateRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateStageTemplate creates a new stage_template using common SQLServer operations
func (r *SQLServerStageTemplateRepository) CreateStageTemplate(ctx context.Context, req *stagetemplatepb.CreateStageTemplateRequest) (*stagetemplatepb.CreateStageTemplateResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("stage_template data is required")
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
		return nil, fmt.Errorf("failed to create stage_template: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	stageTemplate := &stagetemplatepb.StageTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, stageTemplate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &stagetemplatepb.CreateStageTemplateResponse{
		Data:    []*stagetemplatepb.StageTemplate{stageTemplate},
		Success: true,
	}, nil
}

// ReadStageTemplate retrieves a stage_template using common SQLServer operations
func (r *SQLServerStageTemplateRepository) ReadStageTemplate(ctx context.Context, req *stagetemplatepb.ReadStageTemplateRequest) (*stagetemplatepb.ReadStageTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("stage_template ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read stage_template: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	stageTemplate := &stagetemplatepb.StageTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, stageTemplate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &stagetemplatepb.ReadStageTemplateResponse{
		Data:    []*stagetemplatepb.StageTemplate{stageTemplate},
		Success: true,
	}, nil
}

// UpdateStageTemplate updates a stage_template using common SQLServer operations
func (r *SQLServerStageTemplateRepository) UpdateStageTemplate(ctx context.Context, req *stagetemplatepb.UpdateStageTemplateRequest) (*stagetemplatepb.UpdateStageTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("stage_template ID is required")
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
		return nil, fmt.Errorf("failed to update stage_template: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	stageTemplate := &stagetemplatepb.StageTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, stageTemplate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &stagetemplatepb.UpdateStageTemplateResponse{
		Data:    []*stagetemplatepb.StageTemplate{stageTemplate},
		Success: true,
	}, nil
}

// DeleteStageTemplate deletes a stage_template using common SQLServer operations (soft delete)
func (r *SQLServerStageTemplateRepository) DeleteStageTemplate(ctx context.Context, req *stagetemplatepb.DeleteStageTemplateRequest) (*stagetemplatepb.DeleteStageTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("stage_template ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete stage_template: %w", err)
	}

	return &stagetemplatepb.DeleteStageTemplateResponse{
		Success: true,
	}, nil
}

// ListStageTemplates lists stage_templates using common SQLServer operations
func (r *SQLServerStageTemplateRepository) ListStageTemplates(ctx context.Context, req *stagetemplatepb.ListStageTemplatesRequest) (*stagetemplatepb.ListStageTemplatesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list stage_templates: %w", err)
	}

	var stageTemplates []*stagetemplatepb.StageTemplate
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		stageTemplate := &stagetemplatepb.StageTemplate{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, stageTemplate); err != nil {
			continue
		}
		stageTemplates = append(stageTemplates, stageTemplate)
	}

	if stageTemplates == nil {
		stageTemplates = make([]*stagetemplatepb.StageTemplate, 0)
	}

	return &stagetemplatepb.ListStageTemplatesResponse{
		Data:    stageTemplates,
		Success: true,
	}, nil
}

// GetStageTemplateListPageData retrieves stage_templates with basic pagination via List.
func (r *SQLServerStageTemplateRepository) GetStageTemplateListPageData(ctx context.Context, req *stagetemplatepb.GetStageTemplateListPageDataRequest) (*stagetemplatepb.GetStageTemplateListPageDataResponse, error) {
	listReq := &stagetemplatepb.ListStageTemplatesRequest{}
	listResp, err := r.ListStageTemplates(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get stage_template list page data: %w", err)
	}

	return &stagetemplatepb.GetStageTemplateListPageDataResponse{
		StageTemplateList: listResp.Data,
		Success:           true,
	}, nil
}

// GetStageTemplateItemPageData retrieves a single stage_template via Read.
func (r *SQLServerStageTemplateRepository) GetStageTemplateItemPageData(ctx context.Context, req *stagetemplatepb.GetStageTemplateItemPageDataRequest) (*stagetemplatepb.GetStageTemplateItemPageDataResponse, error) {
	if req.StageTemplateId == "" {
		return nil, fmt.Errorf("stage_template ID is required")
	}

	readReq := &stagetemplatepb.ReadStageTemplateRequest{
		Data: &stagetemplatepb.StageTemplate{Id: req.StageTemplateId},
	}
	readResp, err := r.ReadStageTemplate(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get stage_template item page data: %w", err)
	}

	if len(readResp.Data) == 0 {
		return nil, fmt.Errorf("stage_template not found")
	}

	return &stagetemplatepb.GetStageTemplateItemPageDataResponse{
		StageTemplate: readResp.Data[0],
		Success:       true,
	}, nil
}

// NewStageTemplateRepository creates a new SQLServer stage_template repository (old-style constructor)
func NewStageTemplateRepository(db *sql.DB, tableName string) stagetemplatepb.StageTemplateDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerStageTemplateRepository(dbOps, tableName)
}
