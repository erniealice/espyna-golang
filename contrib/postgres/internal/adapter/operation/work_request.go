//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.WorkRequest, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres work_request repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresWorkRequestRepository(dbOps, tableName), nil
	})
}

// PostgresWorkRequestRepository implements work_request CRUD operations using PostgreSQL.
type PostgresWorkRequestRepository struct {
	pb.UnimplementedWorkRequestDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresWorkRequestRepository creates a new PostgreSQL work_request repository.
func NewPostgresWorkRequestRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.WorkRequestDomainServiceServer {
	if tableName == "" {
		tableName = "work_request"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresWorkRequestRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateWorkRequest creates a new work_request record.
func (r *PostgresWorkRequestRepository) CreateWorkRequest(ctx context.Context, req *pb.CreateWorkRequestRequest) (*pb.CreateWorkRequestResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("work_request data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create work_request: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workRequest := &pb.WorkRequest{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workRequest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateWorkRequestResponse{
		Success: true,
		Data:    []*pb.WorkRequest{workRequest},
	}, nil
}

// ReadWorkRequest retrieves a work_request record by ID.
func (r *PostgresWorkRequestRepository) ReadWorkRequest(ctx context.Context, req *pb.ReadWorkRequestRequest) (*pb.ReadWorkRequestResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("work_request ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read work_request: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workRequest := &pb.WorkRequest{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workRequest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadWorkRequestResponse{
		Success: true,
		Data:    []*pb.WorkRequest{workRequest},
	}, nil
}

// UpdateWorkRequest updates a work_request record.
func (r *PostgresWorkRequestRepository) UpdateWorkRequest(ctx context.Context, req *pb.UpdateWorkRequestRequest) (*pb.UpdateWorkRequestResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("work_request ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update work_request: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workRequest := &pb.WorkRequest{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workRequest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateWorkRequestResponse{
		Success: true,
		Data:    []*pb.WorkRequest{workRequest},
	}, nil
}

// DeleteWorkRequest deletes a work_request record (soft delete).
func (r *PostgresWorkRequestRepository) DeleteWorkRequest(ctx context.Context, req *pb.DeleteWorkRequestRequest) (*pb.DeleteWorkRequestResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("work_request ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete work_request: %w", err)
	}

	return &pb.DeleteWorkRequestResponse{
		Success: true,
	}, nil
}

// ListWorkRequests lists work_request records with optional filters.
func (r *PostgresWorkRequestRepository) ListWorkRequests(ctx context.Context, req *pb.ListWorkRequestsRequest) (*pb.ListWorkRequestsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list work_requests: %w", err)
	}

	var workRequests []*pb.WorkRequest
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal work_request row: %v", err)
			continue
		}

		workRequest := &pb.WorkRequest{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workRequest); err != nil {
			log.Printf("WARN: protojson unmarshal work_request: %v", err)
			continue
		}
		workRequests = append(workRequests, workRequest)
	}

	return &pb.ListWorkRequestsResponse{
		Success: true,
		Data:    workRequests,
	}, nil
}
