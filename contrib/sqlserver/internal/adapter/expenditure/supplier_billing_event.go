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
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_billing_event"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.SupplierBillingEvent, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver supplier_billing_event repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerSupplierBillingEventRepository(dbOps, tableName), nil
	})
}

// SQLServerSupplierBillingEventRepository implements supplier_billing_event CRUD using SQL Server.
type SQLServerSupplierBillingEventRepository struct {
	pb.UnimplementedSupplierBillingEventDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerSupplierBillingEventRepository creates a new SQL Server supplier_billing_event repository.
func NewSQLServerSupplierBillingEventRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.SupplierBillingEventDomainServiceServer {
	if tableName == "" {
		tableName = entityid.SupplierBillingEvent
	}
	return &SQLServerSupplierBillingEventRepository{dbOps: dbOps, tableName: tableName}
}

// foldSupplierBillingEventEnumStringsToInt collapses protojson enum-name strings
// on status + trigger to numeric wire values for INTEGER-typed columns.
func foldSupplierBillingEventEnumStringsToInt(data map[string]any) {
	if v, ok := data["status"].(string); ok {
		if num, ok := pb.SupplierBillingEventStatus_value[v]; ok {
			data["status"] = int32(num)
		}
	}
	if v, ok := data["trigger"].(string); ok {
		if num, ok := pb.SupplierBillingEventTrigger_value[v]; ok {
			data["trigger"] = int32(num)
		}
	}
}

func (r *SQLServerSupplierBillingEventRepository) CreateSupplierBillingEvent(ctx context.Context, req *pb.CreateSupplierBillingEventRequest) (*pb.CreateSupplierBillingEventResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("supplier_billing_event data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal supplier_billing_event to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal supplier_billing_event JSON to map: %w", err)
	}
	foldSupplierBillingEventEnumStringsToInt(data)

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create supplier_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal supplier_billing_event result: %w", err)
	}
	ev := &pb.SupplierBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ev); err != nil {
		return nil, fmt.Errorf("failed to unmarshal supplier_billing_event proto: %w", err)
	}
	return &pb.CreateSupplierBillingEventResponse{Success: true, Data: []*pb.SupplierBillingEvent{ev}}, nil
}

func (r *SQLServerSupplierBillingEventRepository) ReadSupplierBillingEvent(ctx context.Context, req *pb.ReadSupplierBillingEventRequest) (*pb.ReadSupplierBillingEventResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier_billing_event ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal supplier_billing_event result: %w", err)
	}
	ev := &pb.SupplierBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ev); err != nil {
		return nil, fmt.Errorf("failed to unmarshal supplier_billing_event proto: %w", err)
	}
	return &pb.ReadSupplierBillingEventResponse{Success: true, Data: []*pb.SupplierBillingEvent{ev}}, nil
}

func (r *SQLServerSupplierBillingEventRepository) UpdateSupplierBillingEvent(ctx context.Context, req *pb.UpdateSupplierBillingEventRequest) (*pb.UpdateSupplierBillingEventResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier_billing_event ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal supplier_billing_event to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal supplier_billing_event JSON to map: %w", err)
	}
	foldSupplierBillingEventEnumStringsToInt(data)

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update supplier_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal supplier_billing_event result: %w", err)
	}
	ev := &pb.SupplierBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ev); err != nil {
		return nil, fmt.Errorf("failed to unmarshal supplier_billing_event proto: %w", err)
	}
	return &pb.UpdateSupplierBillingEventResponse{Success: true, Data: []*pb.SupplierBillingEvent{ev}}, nil
}

func (r *SQLServerSupplierBillingEventRepository) DeleteSupplierBillingEvent(ctx context.Context, req *pb.DeleteSupplierBillingEventRequest) (*pb.DeleteSupplierBillingEventResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier_billing_event ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete supplier_billing_event: %w", err)
	}
	return &pb.DeleteSupplierBillingEventResponse{Success: true}, nil
}

func (r *SQLServerSupplierBillingEventRepository) ListSupplierBillingEvents(ctx context.Context, req *pb.ListSupplierBillingEventsRequest) (*pb.ListSupplierBillingEventsResponse, error) {
	var params *interfaces.ListParams
	if req != nil {
		params = &interfaces.ListParams{
			Search:     req.Search,
			Filters:    req.Filters,
			Sort:       req.Sort,
			Pagination: req.Pagination,
		}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list supplier_billing_events: %w", err)
	}
	events := make([]*pb.SupplierBillingEvent, 0, len(listResult.Data))
	for _, raw := range listResult.Data {
		rawJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(raw))
		if err != nil {
			log.Printf("WARN: marshal supplier_billing_event row: %v", err)
			continue
		}
		ev := &pb.SupplierBillingEvent{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawJSON, ev); err != nil {
			log.Printf("WARN: unmarshal supplier_billing_event row: %v", err)
			continue
		}
		events = append(events, ev)
	}
	return &pb.ListSupplierBillingEventsResponse{Success: true, Data: events}, nil
}

// GetSupplierBillingEventListPageData returns a paginated list view.
func (r *SQLServerSupplierBillingEventRepository) GetSupplierBillingEventListPageData(ctx context.Context, req *pb.GetSupplierBillingEventListPageDataRequest) (*pb.GetSupplierBillingEventListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get supplier_billing_event list page data request is required")
	}
	listResp, err := r.ListSupplierBillingEvents(ctx, &pb.ListSupplierBillingEventsRequest{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, err
	}
	var page *commonpb.PaginationResponse
	if req.Pagination != nil && req.Pagination.GetOffset() != nil {
		currentPage := req.Pagination.GetOffset().Page
		totalItems := int32(len(listResp.Data))
		page = &commonpb.PaginationResponse{
			TotalItems:  totalItems,
			CurrentPage: &currentPage,
		}
	}
	return &pb.GetSupplierBillingEventListPageDataResponse{
		SupplierBillingEventList: listResp.Data,
		Pagination:               page,
		Success:                  true,
	}, nil
}

// GetSupplierBillingEventItemPageData retrieves a single supplier_billing_event.
func (r *SQLServerSupplierBillingEventRepository) GetSupplierBillingEventItemPageData(ctx context.Context, req *pb.GetSupplierBillingEventItemPageDataRequest) (*pb.GetSupplierBillingEventItemPageDataResponse, error) {
	if req == nil || req.SupplierBillingEventId == "" {
		return nil, fmt.Errorf("supplier_billing_event ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.SupplierBillingEventId)
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal supplier_billing_event row: %w", err)
	}
	ev := &pb.SupplierBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ev); err != nil {
		return nil, fmt.Errorf("failed to unmarshal supplier_billing_event: %w", err)
	}
	return &pb.GetSupplierBillingEventItemPageDataResponse{SupplierBillingEvent: ev, Success: true}, nil
}
