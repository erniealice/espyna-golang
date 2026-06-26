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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_settlement"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.JobSettlement, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver job_settlement repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerJobSettlementRepository(dbOps, tableName), nil
	})
}

type SQLServerJobSettlementRepository struct {
	pb.UnimplementedJobSettlementDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func NewSQLServerJobSettlementRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobSettlementDomainServiceServer {
	if tableName == "" {
		tableName = "job_settlement"
	}
	return &SQLServerJobSettlementRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerJobSettlementRepository) CreateJobSettlement(ctx context.Context, req *pb.CreateJobSettlementRequest) (*pb.CreateJobSettlementResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job settlement data is required")
	}
	jsonData, _ := protojson.Marshal(req.Data)
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create job settlement: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	s := &pb.JobSettlement{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, s) //nolint:errcheck
	return &pb.CreateJobSettlementResponse{Success: true, Data: []*pb.JobSettlement{s}}, nil
}

func (r *SQLServerJobSettlementRepository) ReadJobSettlement(ctx context.Context, req *pb.ReadJobSettlementRequest) (*pb.ReadJobSettlementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job settlement ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job settlement: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	s := &pb.JobSettlement{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, s) //nolint:errcheck
	return &pb.ReadJobSettlementResponse{Success: true, Data: []*pb.JobSettlement{s}}, nil
}

func (r *SQLServerJobSettlementRepository) UpdateJobSettlement(ctx context.Context, req *pb.UpdateJobSettlementRequest) (*pb.UpdateJobSettlementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job settlement ID is required")
	}
	jsonData, _ := protojson.Marshal(req.Data)
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update job settlement: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	s := &pb.JobSettlement{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, s) //nolint:errcheck
	return &pb.UpdateJobSettlementResponse{Success: true, Data: []*pb.JobSettlement{s}}, nil
}

func (r *SQLServerJobSettlementRepository) DeleteJobSettlement(ctx context.Context, req *pb.DeleteJobSettlementRequest) (*pb.DeleteJobSettlementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job settlement ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete job settlement: %w", err)
	}
	return &pb.DeleteJobSettlementResponse{Success: true}, nil
}

func (r *SQLServerJobSettlementRepository) ListJobSettlements(ctx context.Context, req *pb.ListJobSettlementsRequest) (*pb.ListJobSettlementsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job settlements: %w", err)
	}
	var items []*pb.JobSettlement
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal job_settlement: %v", err)
			continue
		}
		s := &pb.JobSettlement{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, s); err != nil {
			log.Printf("WARN: protojson unmarshal job_settlement: %v", err)
			continue
		}
		items = append(items, s)
	}
	return &pb.ListJobSettlementsResponse{Success: true, Data: items}, nil
}

func (r *SQLServerJobSettlementRepository) GetJobSettlementListPageData(ctx context.Context, req *pb.GetJobSettlementListPageDataRequest) (*pb.GetJobSettlementListPageDataResponse, error) {
	return nil, fmt.Errorf("GetJobSettlementListPageData not yet implemented")
}

func (r *SQLServerJobSettlementRepository) GetJobSettlementItemPageData(ctx context.Context, req *pb.GetJobSettlementItemPageDataRequest) (*pb.GetJobSettlementItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetJobSettlementItemPageData not yet implemented")
}
