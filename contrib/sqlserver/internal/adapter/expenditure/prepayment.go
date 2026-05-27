//go:build sqlserver

package expenditure

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
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	prepaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/prepayment"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.Prepayment, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver prepayment repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerPrepaymentRepository(dbOps, tableName), nil
	})
}

// SQLServerPrepaymentRepository implements prepayment CRUD using SQL Server.
type SQLServerPrepaymentRepository struct {
	prepaymentpb.UnimplementedPrepaymentDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerPrepaymentRepository creates a new SQL Server prepayment repository.
func NewSQLServerPrepaymentRepository(dbOps interfaces.DatabaseOperation, tableName string) prepaymentpb.PrepaymentDomainServiceServer {
	if tableName == "" {
		tableName = "prepayment"
	}
	return &SQLServerPrepaymentRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerPrepaymentRepository) CreatePrepayment(ctx context.Context, req *prepaymentpb.CreatePrepaymentRequest) (*prepaymentpb.CreatePrepaymentResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("prepayment data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	convertMillisToTime(data, "startDate")
	convertMillisToTime(data, "endDate")
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create prepayment: %w", err)
	}
	sqlserverCore.ConvertMillisToDateStr(result, "start_date", "end_date")
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	prepayment := &prepaymentpb.Prepayment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, prepayment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &prepaymentpb.CreatePrepaymentResponse{Success: true, Data: []*prepaymentpb.Prepayment{prepayment}}, nil
}

func (r *SQLServerPrepaymentRepository) ReadPrepayment(ctx context.Context, req *prepaymentpb.ReadPrepaymentRequest) (*prepaymentpb.ReadPrepaymentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("prepayment ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read prepayment: %w", err)
	}
	sqlserverCore.ConvertMillisToDateStr(result, "start_date", "end_date")
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	prepayment := &prepaymentpb.Prepayment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, prepayment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &prepaymentpb.ReadPrepaymentResponse{Success: true, Data: []*prepaymentpb.Prepayment{prepayment}}, nil
}

func (r *SQLServerPrepaymentRepository) UpdatePrepayment(ctx context.Context, req *prepaymentpb.UpdatePrepaymentRequest) (*prepaymentpb.UpdatePrepaymentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("prepayment ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	convertMillisToTime(data, "startDate")
	convertMillisToTime(data, "endDate")
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update prepayment: %w", err)
	}
	sqlserverCore.ConvertMillisToDateStr(result, "start_date", "end_date")
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	prepayment := &prepaymentpb.Prepayment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, prepayment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &prepaymentpb.UpdatePrepaymentResponse{Success: true, Data: []*prepaymentpb.Prepayment{prepayment}}, nil
}

func (r *SQLServerPrepaymentRepository) DeletePrepayment(ctx context.Context, req *prepaymentpb.DeletePrepaymentRequest) (*prepaymentpb.DeletePrepaymentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("prepayment ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete prepayment: %w", err)
	}
	return &prepaymentpb.DeletePrepaymentResponse{Success: true}, nil
}

func (r *SQLServerPrepaymentRepository) ListPrepayments(ctx context.Context, req *prepaymentpb.ListPrepaymentsRequest) (*prepaymentpb.ListPrepaymentsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list prepayments: %w", err)
	}
	var prepayments []*prepaymentpb.Prepayment
	for _, result := range listResult.Data {
		sqlserverCore.ConvertMillisToDateStr(result, "start_date", "end_date")
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal prepayment row: %v", err)
			continue
		}
		prepayment := &prepaymentpb.Prepayment{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, prepayment); err != nil {
			log.Printf("WARN: protojson unmarshal prepayment: %v", err)
			continue
		}
		prepayments = append(prepayments, prepayment)
	}
	return &prepaymentpb.ListPrepaymentsResponse{Success: true, Data: prepayments}, nil
}

// GetPrepaymentListPageData — Prepayment entity retired 2026-05-17; returns empty list.
func (r *SQLServerPrepaymentRepository) GetPrepaymentListPageData(ctx context.Context, req *prepaymentpb.GetPrepaymentListPageDataRequest) (*prepaymentpb.GetPrepaymentListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get prepayment list page data request is required")
	}
	_ = ctx
	page := int32(1)
	totalPages := int32(0)
	return &prepaymentpb.GetPrepaymentListPageDataResponse{
		PrepaymentList: nil,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  0,
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     false,
			HasPrev:     false,
		},
		Success: true,
	}, nil
}

// GetPrepaymentItemPageData retrieves a single prepayment.
func (r *SQLServerPrepaymentRepository) GetPrepaymentItemPageData(ctx context.Context, req *prepaymentpb.GetPrepaymentItemPageDataRequest) (*prepaymentpb.GetPrepaymentItemPageDataResponse, error) {
	if req == nil || req.PrepaymentId == "" {
		return nil, fmt.Errorf("prepayment ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.PrepaymentId)
	if err != nil {
		return nil, fmt.Errorf("failed to read prepayment '%s': %w", req.PrepaymentId, err)
	}
	sqlserverCore.ConvertMillisToDateStr(result, "start_date", "end_date")
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	prepayment := &prepaymentpb.Prepayment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, prepayment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &prepaymentpb.GetPrepaymentItemPageDataResponse{Prepayment: prepayment, Success: true}, nil
}
