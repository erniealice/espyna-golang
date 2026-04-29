//go:build postgresql

package payroll

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
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	paycyclepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/pay_cycle"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.PayCycle, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres pay_cycle repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresPayCycleRepository(dbOps, tableName), nil
	})
}

// PostgresPayCycleRepository implements pay cycle CRUD operations using PostgreSQL.
type PostgresPayCycleRepository struct {
	paycyclepb.UnimplementedPayCycleDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresPayCycleRepository creates a new PostgreSQL pay cycle repository.
func NewPostgresPayCycleRepository(dbOps interfaces.DatabaseOperation, tableName string) paycyclepb.PayCycleDomainServiceServer {
	if tableName == "" {
		tableName = "pay_cycle"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresPayCycleRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePayCycle creates a new pay cycle record.
func (r *PostgresPayCycleRepository) CreatePayCycle(ctx context.Context, req *paycyclepb.CreatePayCycleRequest) (*paycyclepb.CreatePayCycleResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("pay cycle data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create pay_cycle: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pc := &paycyclepb.PayCycle{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &paycyclepb.CreatePayCycleResponse{Success: true, Data: []*paycyclepb.PayCycle{pc}}, nil
}

// ReadPayCycle retrieves a pay cycle by ID.
func (r *PostgresPayCycleRepository) ReadPayCycle(ctx context.Context, req *paycyclepb.ReadPayCycleRequest) (*paycyclepb.ReadPayCycleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("pay cycle ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read pay_cycle: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pc := &paycyclepb.PayCycle{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &paycyclepb.ReadPayCycleResponse{Success: true, Data: []*paycyclepb.PayCycle{pc}}, nil
}

// UpdatePayCycle updates a pay cycle record.
func (r *PostgresPayCycleRepository) UpdatePayCycle(ctx context.Context, req *paycyclepb.UpdatePayCycleRequest) (*paycyclepb.UpdatePayCycleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("pay cycle ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update pay_cycle: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pc := &paycyclepb.PayCycle{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &paycyclepb.UpdatePayCycleResponse{Success: true, Data: []*paycyclepb.PayCycle{pc}}, nil
}

// DeletePayCycle soft-deletes a pay cycle.
func (r *PostgresPayCycleRepository) DeletePayCycle(ctx context.Context, req *paycyclepb.DeletePayCycleRequest) (*paycyclepb.DeletePayCycleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("pay cycle ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete pay_cycle: %w", err)
	}
	return &paycyclepb.DeletePayCycleResponse{Success: true}, nil
}

// ListPayCycles lists pay cycle records with optional filters.
func (r *PostgresPayCycleRepository) ListPayCycles(ctx context.Context, req *paycyclepb.ListPayCyclesRequest) (*paycyclepb.ListPayCyclesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list pay_cycles: %w", err)
	}
	var items []*paycyclepb.PayCycle
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal pay_cycle row: %v", err)
			continue
		}
		pc := &paycyclepb.PayCycle{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pc); err != nil {
			log.Printf("WARN: protojson unmarshal pay_cycle: %v", err)
			continue
		}
		items = append(items, pc)
	}
	return &paycyclepb.ListPayCyclesResponse{Success: true, Data: items}, nil
}

// GetPayCycleListPageData retrieves pay cycles with pagination, filtering, sorting, and search.
func (r *PostgresPayCycleRepository) GetPayCycleListPageData(
	ctx context.Context,
	req *paycyclepb.GetPayCycleListPageDataRequest,
) (*paycyclepb.GetPayCycleListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get pay cycle list page data request is required")
	}

	var params *interfaces.ListParams
	if req.Filters != nil {
		if params == nil {
			params = &interfaces.ListParams{}
		}
		params.Filters = req.Filters
	}

	limit := int32(50)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
			}
		}
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list pay_cycle list page data: %w", err)
	}

	var items []*paycyclepb.PayCycle
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal pay_cycle row: %v", err)
			continue
		}
		pc := &paycyclepb.PayCycle{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pc); err != nil {
			log.Printf("WARN: protojson unmarshal pay_cycle: %v", err)
			continue
		}
		items = append(items, pc)
	}

	totalCount := int64(len(items))
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &paycyclepb.GetPayCycleListPageDataResponse{
		PayCycleList: items,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
		Success: true,
	}, nil
}

// GetPayCycleItemPageData retrieves a single pay cycle.
func (r *PostgresPayCycleRepository) GetPayCycleItemPageData(
	ctx context.Context,
	req *paycyclepb.GetPayCycleItemPageDataRequest,
) (*paycyclepb.GetPayCycleItemPageDataResponse, error) {
	if req == nil || req.GetPayCycleId() == "" {
		return nil, fmt.Errorf("pay cycle ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetPayCycleId())
	if err != nil {
		return nil, fmt.Errorf("failed to read pay_cycle item: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pc := &paycyclepb.PayCycle{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &paycyclepb.GetPayCycleItemPageDataResponse{
		PayCycle: pc,
		Success:  true,
	}, nil
}
