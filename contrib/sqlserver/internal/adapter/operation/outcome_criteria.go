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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.OutcomeCriteria, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver outcome_criteria repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerOutcomeCriteriaRepository(dbOps, tableName), nil
	})
}

// SQLServerOutcomeCriteriaRepository implements outcome_criteria CRUD operations using SQL Server.
type SQLServerOutcomeCriteriaRepository struct {
	pb.UnimplementedOutcomeCriteriaDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerOutcomeCriteriaRepository creates a new SQL Server outcome_criteria repository.
func NewSQLServerOutcomeCriteriaRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.OutcomeCriteriaDomainServiceServer {
	if tableName == "" {
		tableName = "outcome_criteria"
	}
	return &SQLServerOutcomeCriteriaRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerOutcomeCriteriaRepository) CreateOutcomeCriteria(ctx context.Context, req *pb.CreateOutcomeCriteriaRequest) (*pb.CreateOutcomeCriteriaResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("outcome_criteria data is required")
	}
	jsonData, _ := protojson.Marshal(req.Data)
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create outcome_criteria: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	oc := &pb.OutcomeCriteria{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, oc) //nolint:errcheck
	return &pb.CreateOutcomeCriteriaResponse{Success: true, Data: []*pb.OutcomeCriteria{oc}}, nil
}

func (r *SQLServerOutcomeCriteriaRepository) ReadOutcomeCriteria(ctx context.Context, req *pb.ReadOutcomeCriteriaRequest) (*pb.ReadOutcomeCriteriaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("outcome_criteria ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read outcome_criteria: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	oc := &pb.OutcomeCriteria{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, oc) //nolint:errcheck
	return &pb.ReadOutcomeCriteriaResponse{Success: true, Data: []*pb.OutcomeCriteria{oc}}, nil
}

func (r *SQLServerOutcomeCriteriaRepository) UpdateOutcomeCriteria(ctx context.Context, req *pb.UpdateOutcomeCriteriaRequest) (*pb.UpdateOutcomeCriteriaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("outcome_criteria ID is required")
	}
	jsonData, _ := protojson.Marshal(req.Data)
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update outcome_criteria: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	oc := &pb.OutcomeCriteria{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, oc) //nolint:errcheck
	return &pb.UpdateOutcomeCriteriaResponse{Success: true, Data: []*pb.OutcomeCriteria{oc}}, nil
}

func (r *SQLServerOutcomeCriteriaRepository) DeleteOutcomeCriteria(ctx context.Context, req *pb.DeleteOutcomeCriteriaRequest) (*pb.DeleteOutcomeCriteriaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("outcome_criteria ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete outcome_criteria: %w", err)
	}
	return &pb.DeleteOutcomeCriteriaResponse{Success: true}, nil
}

func (r *SQLServerOutcomeCriteriaRepository) ListOutcomeCriterias(ctx context.Context, req *pb.ListOutcomeCriteriasRequest) (*pb.ListOutcomeCriteriasResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list outcome_criterias: %w", err)
	}
	var items []*pb.OutcomeCriteria
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal outcome_criteria row: %v", err)
			continue
		}
		oc := &pb.OutcomeCriteria{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, oc); err != nil {
			log.Printf("WARN: protojson unmarshal outcome_criteria: %v", err)
			continue
		}
		items = append(items, oc)
	}
	return &pb.ListOutcomeCriteriasResponse{Success: true, Data: items}, nil
}

func (r *SQLServerOutcomeCriteriaRepository) GetOutcomeCriteriaListPageData(ctx context.Context, req *pb.GetOutcomeCriteriaListPageDataRequest) (*pb.GetOutcomeCriteriaListPageDataResponse, error) {

	return nil, fmt.Errorf("GetOutcomeCriteriaListPageData not yet implemented")
}

func (r *SQLServerOutcomeCriteriaRepository) GetOutcomeCriteriaItemPageData(ctx context.Context, req *pb.GetOutcomeCriteriaItemPageDataRequest) (*pb.GetOutcomeCriteriaItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetOutcomeCriteriaItemPageData not yet implemented")
}
