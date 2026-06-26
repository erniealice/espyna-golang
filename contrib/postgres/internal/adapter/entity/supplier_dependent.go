//go:build postgresql

package entity

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
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	supplierdependentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_dependent"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SupplierDependent, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres supplier_dependent repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSupplierDependentRepository(dbOps, tableName), nil
	})
}

// PostgresSupplierDependentRepository implements supplier dependent CRUD operations using PostgreSQL.
type PostgresSupplierDependentRepository struct {
	supplierdependentpb.UnimplementedSupplierDependentDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresSupplierDependentRepository creates a new PostgreSQL supplier dependent repository.
func NewPostgresSupplierDependentRepository(dbOps interfaces.DatabaseOperation, tableName string) supplierdependentpb.SupplierDependentDomainServiceServer {
	if tableName == "" {
		tableName = "supplier_dependent"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresSupplierDependentRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateSupplierDependent creates a new supplier dependent record.
func (r *PostgresSupplierDependentRepository) CreateSupplierDependent(ctx context.Context, req *supplierdependentpb.CreateSupplierDependentRequest) (*supplierdependentpb.CreateSupplierDependentResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("supplier dependent data is required")
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
		return nil, fmt.Errorf("failed to create supplier_dependent: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	sd := &supplierdependentpb.SupplierDependent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &supplierdependentpb.CreateSupplierDependentResponse{Success: true, Data: []*supplierdependentpb.SupplierDependent{sd}}, nil
}

// ReadSupplierDependent retrieves a supplier dependent by ID.
func (r *PostgresSupplierDependentRepository) ReadSupplierDependent(ctx context.Context, req *supplierdependentpb.ReadSupplierDependentRequest) (*supplierdependentpb.ReadSupplierDependentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier dependent ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_dependent: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	sd := &supplierdependentpb.SupplierDependent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &supplierdependentpb.ReadSupplierDependentResponse{Success: true, Data: []*supplierdependentpb.SupplierDependent{sd}}, nil
}

// UpdateSupplierDependent updates a supplier dependent record.
func (r *PostgresSupplierDependentRepository) UpdateSupplierDependent(ctx context.Context, req *supplierdependentpb.UpdateSupplierDependentRequest) (*supplierdependentpb.UpdateSupplierDependentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier dependent ID is required")
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
		return nil, fmt.Errorf("failed to update supplier_dependent: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	sd := &supplierdependentpb.SupplierDependent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &supplierdependentpb.UpdateSupplierDependentResponse{Success: true, Data: []*supplierdependentpb.SupplierDependent{sd}}, nil
}

// DeleteSupplierDependent soft-deletes a supplier dependent.
func (r *PostgresSupplierDependentRepository) DeleteSupplierDependent(ctx context.Context, req *supplierdependentpb.DeleteSupplierDependentRequest) (*supplierdependentpb.DeleteSupplierDependentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier dependent ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete supplier_dependent: %w", err)
	}
	return &supplierdependentpb.DeleteSupplierDependentResponse{Success: true}, nil
}

// ListSupplierDependents lists supplier dependent records with optional filters.
func (r *PostgresSupplierDependentRepository) ListSupplierDependents(ctx context.Context, req *supplierdependentpb.ListSupplierDependentsRequest) (*supplierdependentpb.ListSupplierDependentsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list supplier_dependents: %w", err)
	}
	var items []*supplierdependentpb.SupplierDependent
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal supplier_dependent row: %v", err)
			continue
		}
		sd := &supplierdependentpb.SupplierDependent{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sd); err != nil {
			log.Printf("WARN: protojson unmarshal supplier_dependent: %v", err)
			continue
		}
		items = append(items, sd)
	}
	return &supplierdependentpb.ListSupplierDependentsResponse{Success: true, Data: items}, nil
}

// GetSupplierDependentListPageData retrieves supplier dependents with pagination, filtering, sorting, and search.
func (r *PostgresSupplierDependentRepository) GetSupplierDependentListPageData(
	ctx context.Context,
	req *supplierdependentpb.GetSupplierDependentListPageDataRequest,
) (*supplierdependentpb.GetSupplierDependentListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get supplier dependent list page data request is required")
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
		return nil, fmt.Errorf("failed to list supplier_dependent list page data: %w", err)
	}

	var items []*supplierdependentpb.SupplierDependent
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal supplier_dependent row: %v", err)
			continue
		}
		sd := &supplierdependentpb.SupplierDependent{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sd); err != nil {
			log.Printf("WARN: protojson unmarshal supplier_dependent: %v", err)
			continue
		}
		items = append(items, sd)
	}

	totalCount := int64(len(items))
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &supplierdependentpb.GetSupplierDependentListPageDataResponse{
		SupplierDependentList: items,
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

// GetSupplierDependentItemPageData retrieves a single supplier dependent.
func (r *PostgresSupplierDependentRepository) GetSupplierDependentItemPageData(
	ctx context.Context,
	req *supplierdependentpb.GetSupplierDependentItemPageDataRequest,
) (*supplierdependentpb.GetSupplierDependentItemPageDataResponse, error) {
	if req == nil || req.GetSupplierDependentId() == "" {
		return nil, fmt.Errorf("supplier dependent ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetSupplierDependentId())
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_dependent item: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	sd := &supplierdependentpb.SupplierDependent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &supplierdependentpb.GetSupplierDependentItemPageDataResponse{
		SupplierDependent: sd,
		Success:           true,
	}, nil
}
