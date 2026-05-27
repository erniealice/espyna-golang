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
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_relation"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.JobTemplateRelation, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver job_template_relation repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerJobTemplateRelationRepository(dbOps, tableName), nil
	})
}

// SQLServerJobTemplateRelationRepository implements job_template_relation CRUD operations using SQL Server.
type SQLServerJobTemplateRelationRepository struct {
	pb.UnimplementedJobTemplateRelationDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerJobTemplateRelationRepository creates a new SQL Server job_template_relation repository.
func NewSQLServerJobTemplateRelationRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobTemplateRelationDomainServiceServer {
	if tableName == "" {
		tableName = "job_template_relation"
	}
	return &SQLServerJobTemplateRelationRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerJobTemplateRelationRepository) CreateJobTemplateRelation(ctx context.Context, req *pb.CreateJobTemplateRelationRequest) (*pb.CreateJobTemplateRelationResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job_template_relation data is required")
	}
	jsonData, _ := protojson.Marshal(req.Data)
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create job_template_relation: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	rel := &pb.JobTemplateRelation{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rel) //nolint:errcheck
	return &pb.CreateJobTemplateRelationResponse{Success: true, Data: []*pb.JobTemplateRelation{rel}}, nil
}

func (r *SQLServerJobTemplateRelationRepository) ReadJobTemplateRelation(ctx context.Context, req *pb.ReadJobTemplateRelationRequest) (*pb.ReadJobTemplateRelationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job_template_relation ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job_template_relation: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	rel := &pb.JobTemplateRelation{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rel) //nolint:errcheck
	return &pb.ReadJobTemplateRelationResponse{Success: true, Data: []*pb.JobTemplateRelation{rel}}, nil
}

func (r *SQLServerJobTemplateRelationRepository) UpdateJobTemplateRelation(ctx context.Context, req *pb.UpdateJobTemplateRelationRequest) (*pb.UpdateJobTemplateRelationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job_template_relation ID is required")
	}
	jsonData, _ := protojson.Marshal(req.Data)
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update job_template_relation: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	rel := &pb.JobTemplateRelation{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rel) //nolint:errcheck
	return &pb.UpdateJobTemplateRelationResponse{Success: true, Data: []*pb.JobTemplateRelation{rel}}, nil
}

func (r *SQLServerJobTemplateRelationRepository) DeleteJobTemplateRelation(ctx context.Context, req *pb.DeleteJobTemplateRelationRequest) (*pb.DeleteJobTemplateRelationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job_template_relation ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete job_template_relation: %w", err)
	}
	return &pb.DeleteJobTemplateRelationResponse{Success: true}, nil
}

func (r *SQLServerJobTemplateRelationRepository) ListJobTemplateRelations(ctx context.Context, req *pb.ListJobTemplateRelationsRequest) (*pb.ListJobTemplateRelationsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job_template_relations: %w", err)
	}
	var items []*pb.JobTemplateRelation
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal job_template_relation row: %v", err)
			continue
		}
		rel := &pb.JobTemplateRelation{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rel); err != nil {
			log.Printf("WARN: protojson unmarshal job_template_relation: %v", err)
			continue
		}
		items = append(items, rel)
	}
	return &pb.ListJobTemplateRelationsResponse{Success: true, Data: items}, nil
}

func (r *SQLServerJobTemplateRelationRepository) GetJobTemplateRelationListPageData(ctx context.Context, req *pb.GetJobTemplateRelationListPageDataRequest) (*pb.GetJobTemplateRelationListPageDataResponse, error) {

	return nil, fmt.Errorf("GetJobTemplateRelationListPageData not yet implemented")
}

func (r *SQLServerJobTemplateRelationRepository) GetJobTemplateRelationItemPageData(ctx context.Context, req *pb.GetJobTemplateRelationItemPageDataRequest) (*pb.GetJobTemplateRelationItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetJobTemplateRelationItemPageData not yet implemented")
}
