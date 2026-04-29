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
	ratetablepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/rate_table"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.RateTable, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres rate_table repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresRateTableRepository(dbOps, tableName), nil
	})
}

// PostgresRateTableRepository implements rate table CRUD operations using PostgreSQL.
type PostgresRateTableRepository struct {
	ratetablepb.UnimplementedRateTableDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresRateTableRepository creates a new PostgreSQL rate table repository.
func NewPostgresRateTableRepository(dbOps interfaces.DatabaseOperation, tableName string) ratetablepb.RateTableDomainServiceServer {
	if tableName == "" {
		tableName = "rate_table"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresRateTableRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateRateTable creates a new rate table record.
func (r *PostgresRateTableRepository) CreateRateTable(ctx context.Context, req *ratetablepb.CreateRateTableRequest) (*ratetablepb.CreateRateTableResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("rate table data is required")
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
		return nil, fmt.Errorf("failed to create rate_table: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	rt := &ratetablepb.RateTable{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &ratetablepb.CreateRateTableResponse{Success: true, Data: []*ratetablepb.RateTable{rt}}, nil
}

// ReadRateTable retrieves a rate table by ID.
func (r *PostgresRateTableRepository) ReadRateTable(ctx context.Context, req *ratetablepb.ReadRateTableRequest) (*ratetablepb.ReadRateTableResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rate table ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read rate_table: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	rt := &ratetablepb.RateTable{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &ratetablepb.ReadRateTableResponse{Success: true, Data: []*ratetablepb.RateTable{rt}}, nil
}

// UpdateRateTable updates a rate table record.
func (r *PostgresRateTableRepository) UpdateRateTable(ctx context.Context, req *ratetablepb.UpdateRateTableRequest) (*ratetablepb.UpdateRateTableResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rate table ID is required")
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
		return nil, fmt.Errorf("failed to update rate_table: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	rt := &ratetablepb.RateTable{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &ratetablepb.UpdateRateTableResponse{Success: true, Data: []*ratetablepb.RateTable{rt}}, nil
}

// DeleteRateTable soft-deletes a rate table.
func (r *PostgresRateTableRepository) DeleteRateTable(ctx context.Context, req *ratetablepb.DeleteRateTableRequest) (*ratetablepb.DeleteRateTableResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rate table ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete rate_table: %w", err)
	}
	return &ratetablepb.DeleteRateTableResponse{Success: true}, nil
}

// ListRateTables lists rate table records with optional filters.
func (r *PostgresRateTableRepository) ListRateTables(ctx context.Context, req *ratetablepb.ListRateTablesRequest) (*ratetablepb.ListRateTablesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list rate_tables: %w", err)
	}
	var items []*ratetablepb.RateTable
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal rate_table row: %v", err)
			continue
		}
		rt := &ratetablepb.RateTable{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rt); err != nil {
			log.Printf("WARN: protojson unmarshal rate_table: %v", err)
			continue
		}
		items = append(items, rt)
	}
	return &ratetablepb.ListRateTablesResponse{Success: true, Data: items}, nil
}

// GetRateTableListPageData retrieves rate tables with pagination, filtering, sorting, and search.
func (r *PostgresRateTableRepository) GetRateTableListPageData(
	ctx context.Context,
	req *ratetablepb.GetRateTableListPageDataRequest,
) (*ratetablepb.GetRateTableListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get rate table list page data request is required")
	}

	var params *interfaces.ListParams
	if req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
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
		return nil, fmt.Errorf("failed to list rate_table list page data: %w", err)
	}

	var items []*ratetablepb.RateTable
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal rate_table row: %v", err)
			continue
		}
		rt := &ratetablepb.RateTable{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rt); err != nil {
			log.Printf("WARN: protojson unmarshal rate_table: %v", err)
			continue
		}
		items = append(items, rt)
	}

	totalCount := int64(len(items))
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &ratetablepb.GetRateTableListPageDataResponse{
		RateTableList: items,
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

// GetRateTableItemPageData retrieves a single rate table.
func (r *PostgresRateTableRepository) GetRateTableItemPageData(
	ctx context.Context,
	req *ratetablepb.GetRateTableItemPageDataRequest,
) (*ratetablepb.GetRateTableItemPageDataResponse, error) {
	if req == nil || req.GetRateTableId() == "" {
		return nil, fmt.Errorf("rate table ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetRateTableId())
	if err != nil {
		return nil, fmt.Errorf("failed to read rate_table item: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	rt := &ratetablepb.RateTable{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &ratetablepb.GetRateTableItemPageDataResponse{
		RateTable: rt,
		Success:   true,
	}, nil
}
