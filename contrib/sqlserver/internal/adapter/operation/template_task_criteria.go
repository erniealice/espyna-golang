//go:build sqlserver

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.TemplateTaskCriteria, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver template_task_criteria repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerTemplateTaskCriteriaRepository(dbOps, tableName), nil
	})
}

// SQLServerTemplateTaskCriteriaRepository implements template_task_criteria CRUD operations using SQL Server.
type SQLServerTemplateTaskCriteriaRepository struct {
	pb.UnimplementedTemplateTaskCriteriaDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerTemplateTaskCriteriaRepository creates a new SQL Server template_task_criteria repository.
func NewSQLServerTemplateTaskCriteriaRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.TemplateTaskCriteriaDomainServiceServer {
	if tableName == "" {
		tableName = "template_task_criteria"
	}
	return &SQLServerTemplateTaskCriteriaRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerTemplateTaskCriteriaRepository) CreateTemplateTaskCriteria(ctx context.Context, req *pb.CreateTemplateTaskCriteriaRequest) (*pb.CreateTemplateTaskCriteriaResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("template_task_criteria data is required")
	}
	jsonData, _ := protojson.Marshal(req.Data)
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create template_task_criteria: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	ttc := &pb.TemplateTaskCriteria{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ttc) //nolint:errcheck
	return &pb.CreateTemplateTaskCriteriaResponse{Success: true, Data: []*pb.TemplateTaskCriteria{ttc}}, nil
}

func (r *SQLServerTemplateTaskCriteriaRepository) ReadTemplateTaskCriteria(ctx context.Context, req *pb.ReadTemplateTaskCriteriaRequest) (*pb.ReadTemplateTaskCriteriaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("template_task_criteria ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read template_task_criteria: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	ttc := &pb.TemplateTaskCriteria{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ttc) //nolint:errcheck
	return &pb.ReadTemplateTaskCriteriaResponse{Success: true, Data: []*pb.TemplateTaskCriteria{ttc}}, nil
}

func (r *SQLServerTemplateTaskCriteriaRepository) UpdateTemplateTaskCriteria(ctx context.Context, req *pb.UpdateTemplateTaskCriteriaRequest) (*pb.UpdateTemplateTaskCriteriaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("template_task_criteria ID is required")
	}
	jsonData, _ := protojson.Marshal(req.Data)
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update template_task_criteria: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	ttc := &pb.TemplateTaskCriteria{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ttc) //nolint:errcheck
	return &pb.UpdateTemplateTaskCriteriaResponse{Success: true, Data: []*pb.TemplateTaskCriteria{ttc}}, nil
}

func (r *SQLServerTemplateTaskCriteriaRepository) DeleteTemplateTaskCriteria(ctx context.Context, req *pb.DeleteTemplateTaskCriteriaRequest) (*pb.DeleteTemplateTaskCriteriaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("template_task_criteria ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete template_task_criteria: %w", err)
	}
	return &pb.DeleteTemplateTaskCriteriaResponse{Success: true}, nil
}

func (r *SQLServerTemplateTaskCriteriaRepository) ListTemplateTaskCriterias(ctx context.Context, req *pb.ListTemplateTaskCriteriasRequest) (*pb.ListTemplateTaskCriteriasResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list template_task_criterias: %w", err)
	}
	var items []*pb.TemplateTaskCriteria
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal template_task_criteria row: %v", err)
			continue
		}
		ttc := &pb.TemplateTaskCriteria{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ttc); err != nil {
			log.Printf("WARN: protojson unmarshal template_task_criteria: %v", err)
			continue
		}
		items = append(items, ttc)
	}
	return &pb.ListTemplateTaskCriteriasResponse{Success: true, Data: items}, nil
}

func (r *SQLServerTemplateTaskCriteriaRepository) GetTemplateTaskCriteriaListPageData(ctx context.Context, req *pb.GetTemplateTaskCriteriaListPageDataRequest) (*pb.GetTemplateTaskCriteriaListPageDataResponse, error) {

	return nil, fmt.Errorf("GetTemplateTaskCriteriaListPageData not yet implemented")
}

func (r *SQLServerTemplateTaskCriteriaRepository) GetTemplateTaskCriteriaItemPageData(ctx context.Context, req *pb.GetTemplateTaskCriteriaItemPageDataRequest) (*pb.GetTemplateTaskCriteriaItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetTemplateTaskCriteriaItemPageData not yet implemented")
}
