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
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request_type"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.WorkRequestType, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres work_request_type repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresWorkRequestTypeRepository(dbOps, tableName), nil
	})
}

// PostgresWorkRequestTypeRepository implements work_request_type CRUD operations using PostgreSQL.
type PostgresWorkRequestTypeRepository struct {
	pb.UnimplementedWorkRequestTypeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresWorkRequestTypeRepository creates a new PostgreSQL work_request_type repository.
func NewPostgresWorkRequestTypeRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.WorkRequestTypeDomainServiceServer {
	if tableName == "" {
		tableName = "work_request_type"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresWorkRequestTypeRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateWorkRequestType creates a new work_request_type record.
func (r *PostgresWorkRequestTypeRepository) CreateWorkRequestType(ctx context.Context, req *pb.CreateWorkRequestTypeRequest) (*pb.CreateWorkRequestTypeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("work_request_type data is required")
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
		return nil, fmt.Errorf("failed to create work_request_type: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workRequestType := &pb.WorkRequestType{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workRequestType); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateWorkRequestTypeResponse{
		Success: true,
		Data:    []*pb.WorkRequestType{workRequestType},
	}, nil
}

// ReadWorkRequestType retrieves a work_request_type record by ID.
func (r *PostgresWorkRequestTypeRepository) ReadWorkRequestType(ctx context.Context, req *pb.ReadWorkRequestTypeRequest) (*pb.ReadWorkRequestTypeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("work_request_type ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read work_request_type: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workRequestType := &pb.WorkRequestType{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workRequestType); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadWorkRequestTypeResponse{
		Success: true,
		Data:    []*pb.WorkRequestType{workRequestType},
	}, nil
}

// UpdateWorkRequestType updates a work_request_type record.
func (r *PostgresWorkRequestTypeRepository) UpdateWorkRequestType(ctx context.Context, req *pb.UpdateWorkRequestTypeRequest) (*pb.UpdateWorkRequestTypeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("work_request_type ID is required")
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
		return nil, fmt.Errorf("failed to update work_request_type: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workRequestType := &pb.WorkRequestType{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workRequestType); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateWorkRequestTypeResponse{
		Success: true,
		Data:    []*pb.WorkRequestType{workRequestType},
	}, nil
}

// DeleteWorkRequestType deletes a work_request_type record (soft delete).
func (r *PostgresWorkRequestTypeRepository) DeleteWorkRequestType(ctx context.Context, req *pb.DeleteWorkRequestTypeRequest) (*pb.DeleteWorkRequestTypeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("work_request_type ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete work_request_type: %w", err)
	}

	return &pb.DeleteWorkRequestTypeResponse{
		Success: true,
	}, nil
}

// ListWorkRequestTypes lists work_request_type records with optional filters.
func (r *PostgresWorkRequestTypeRepository) ListWorkRequestTypes(ctx context.Context, req *pb.ListWorkRequestTypesRequest) (*pb.ListWorkRequestTypesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list work_request_types: %w", err)
	}

	var workRequestTypes []*pb.WorkRequestType
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal work_request_type row: %v", err)
			continue
		}

		workRequestType := &pb.WorkRequestType{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workRequestType); err != nil {
			log.Printf("WARN: protojson unmarshal work_request_type: %v", err)
			continue
		}
		workRequestTypes = append(workRequestTypes, workRequestType)
	}

	return &pb.ListWorkRequestTypesResponse{
		Success: true,
		Data:    workRequestTypes,
	}, nil
}
