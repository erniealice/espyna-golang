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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.PhaseOutcomeSummary, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver phase_outcome_summary repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerPhaseOutcomeSummaryRepository(dbOps, tableName), nil
	})
}

// SQLServerPhaseOutcomeSummaryRepository implements phase_outcome_summary CRUD operations using SQL Server.
type SQLServerPhaseOutcomeSummaryRepository struct {
	pb.UnimplementedPhaseOutcomeSummaryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerPhaseOutcomeSummaryRepository creates a new SQL Server phase_outcome_summary repository.
func NewSQLServerPhaseOutcomeSummaryRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.PhaseOutcomeSummaryDomainServiceServer {
	if tableName == "" {
		tableName = "phase_outcome_summary"
	}
	return &SQLServerPhaseOutcomeSummaryRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerPhaseOutcomeSummaryRepository) CreatePhaseOutcomeSummary(ctx context.Context, req *pb.CreatePhaseOutcomeSummaryRequest) (*pb.CreatePhaseOutcomeSummaryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("phase_outcome_summary data is required")
	}
	jsonData, _ := protojson.Marshal(req.Data)
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create phase_outcome_summary: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	pos := &pb.PhaseOutcomeSummary{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pos) //nolint:errcheck
	return &pb.CreatePhaseOutcomeSummaryResponse{Success: true, Data: []*pb.PhaseOutcomeSummary{pos}}, nil
}

func (r *SQLServerPhaseOutcomeSummaryRepository) ReadPhaseOutcomeSummary(ctx context.Context, req *pb.ReadPhaseOutcomeSummaryRequest) (*pb.ReadPhaseOutcomeSummaryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("phase_outcome_summary ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read phase_outcome_summary: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	pos := &pb.PhaseOutcomeSummary{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pos) //nolint:errcheck
	return &pb.ReadPhaseOutcomeSummaryResponse{Success: true, Data: []*pb.PhaseOutcomeSummary{pos}}, nil
}

func (r *SQLServerPhaseOutcomeSummaryRepository) UpdatePhaseOutcomeSummary(ctx context.Context, req *pb.UpdatePhaseOutcomeSummaryRequest) (*pb.UpdatePhaseOutcomeSummaryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("phase_outcome_summary ID is required")
	}
	jsonData, _ := protojson.Marshal(req.Data)
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update phase_outcome_summary: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	pos := &pb.PhaseOutcomeSummary{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pos) //nolint:errcheck
	return &pb.UpdatePhaseOutcomeSummaryResponse{Success: true, Data: []*pb.PhaseOutcomeSummary{pos}}, nil
}

func (r *SQLServerPhaseOutcomeSummaryRepository) DeletePhaseOutcomeSummary(ctx context.Context, req *pb.DeletePhaseOutcomeSummaryRequest) (*pb.DeletePhaseOutcomeSummaryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("phase_outcome_summary ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete phase_outcome_summary: %w", err)
	}
	return &pb.DeletePhaseOutcomeSummaryResponse{Success: true}, nil
}

func (r *SQLServerPhaseOutcomeSummaryRepository) ListPhaseOutcomeSummarys(ctx context.Context, req *pb.ListPhaseOutcomeSummarysRequest) (*pb.ListPhaseOutcomeSummarysResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list phase_outcome_summarys: %w", err)
	}
	var items []*pb.PhaseOutcomeSummary
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal phase_outcome_summary row: %v", err)
			continue
		}
		pos := &pb.PhaseOutcomeSummary{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pos); err != nil {
			log.Printf("WARN: protojson unmarshal phase_outcome_summary: %v", err)
			continue
		}
		items = append(items, pos)
	}
	return &pb.ListPhaseOutcomeSummarysResponse{Success: true, Data: items}, nil
}

func (r *SQLServerPhaseOutcomeSummaryRepository) GetPhaseOutcomeSummaryListPageData(ctx context.Context, req *pb.GetPhaseOutcomeSummaryListPageDataRequest) (*pb.GetPhaseOutcomeSummaryListPageDataResponse, error) {
	// TODO: Implement CTE-based paginated query.
	return nil, fmt.Errorf("GetPhaseOutcomeSummaryListPageData not yet implemented")
}

func (r *SQLServerPhaseOutcomeSummaryRepository) GetPhaseOutcomeSummaryItemPageData(ctx context.Context, req *pb.GetPhaseOutcomeSummaryItemPageDataRequest) (*pb.GetPhaseOutcomeSummaryItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetPhaseOutcomeSummaryItemPageData not yet implemented")
}
