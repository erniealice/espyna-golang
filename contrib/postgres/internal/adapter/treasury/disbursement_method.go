//go:build postgresql

package treasury

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
	disbursementmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_method"
)

// Treasury-domain-rebuild Stage 1 (Wave 5): the postgres disbursement_method
// repository — symmetric to collection_method minus audience_mode (D-4.9
// buying-side asymmetry: no audience-grant model in v1). Template-level columns
// added by migration 20260527000000. Generic protojson round-trip carries all
// scalar + enum (TEXT) fields; message-typed oneof variants are not columns.
func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.DisbursementMethod, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres disbursement_method repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresDisbursementMethodRepository(dbOps, tableName), nil
	})
}

// PostgresDisbursementMethodRepository implements disbursement_method CRUD using PostgreSQL.
type PostgresDisbursementMethodRepository struct {
	disbursementmethodpb.UnimplementedDisbursementMethodDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresDisbursementMethodRepository creates a new PostgreSQL disbursement_method repository.
func NewPostgresDisbursementMethodRepository(dbOps interfaces.DatabaseOperation, tableName string) disbursementmethodpb.DisbursementMethodDomainServiceServer {
	if tableName == "" {
		tableName = "disbursement_method"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresDisbursementMethodRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func disbursementMethodToMap(dm *disbursementmethodpb.DisbursementMethod) (map[string]any, error) {
	jsonData, err := protojson.Marshal(dm)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal disbursement_method protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal disbursement_method JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")
	return data, nil
}

func mapToDisbursementMethod(result map[string]any) (*disbursementmethodpb.DisbursementMethod, error) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal disbursement_method result to JSON: %w", err)
	}
	dm := &disbursementmethodpb.DisbursementMethod{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, dm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to disbursement_method protobuf: %w", err)
	}
	return dm, nil
}

// CreateDisbursementMethod creates a new disbursement_method record.
func (r *PostgresDisbursementMethodRepository) CreateDisbursementMethod(ctx context.Context, req *disbursementmethodpb.CreateDisbursementMethodRequest) (*disbursementmethodpb.CreateDisbursementMethodResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("disbursement_method data is required")
	}

	data, err := disbursementMethodToMap(req.Data)
	if err != nil {
		return nil, err
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create disbursement_method: %w", err)
	}

	dm, err := mapToDisbursementMethod(result)
	if err != nil {
		return nil, err
	}

	return &disbursementmethodpb.CreateDisbursementMethodResponse{
		Success: true,
		Data:    []*disbursementmethodpb.DisbursementMethod{dm},
	}, nil
}

// ReadDisbursementMethod retrieves a disbursement_method record by ID.
func (r *PostgresDisbursementMethodRepository) ReadDisbursementMethod(ctx context.Context, req *disbursementmethodpb.ReadDisbursementMethodRequest) (*disbursementmethodpb.ReadDisbursementMethodResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("disbursement_method ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read disbursement_method: %w", err)
	}

	dm, err := mapToDisbursementMethod(result)
	if err != nil {
		return nil, err
	}

	return &disbursementmethodpb.ReadDisbursementMethodResponse{
		Success: true,
		Data:    []*disbursementmethodpb.DisbursementMethod{dm},
	}, nil
}

// UpdateDisbursementMethod updates a disbursement_method record.
func (r *PostgresDisbursementMethodRepository) UpdateDisbursementMethod(ctx context.Context, req *disbursementmethodpb.UpdateDisbursementMethodRequest) (*disbursementmethodpb.UpdateDisbursementMethodResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("disbursement_method ID is required")
	}

	data, err := disbursementMethodToMap(req.Data)
	if err != nil {
		return nil, err
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update disbursement_method: %w", err)
	}

	dm, err := mapToDisbursementMethod(result)
	if err != nil {
		return nil, err
	}

	return &disbursementmethodpb.UpdateDisbursementMethodResponse{
		Success: true,
		Data:    []*disbursementmethodpb.DisbursementMethod{dm},
	}, nil
}

// DeleteDisbursementMethod deletes a disbursement_method record (soft delete).
func (r *PostgresDisbursementMethodRepository) DeleteDisbursementMethod(ctx context.Context, req *disbursementmethodpb.DeleteDisbursementMethodRequest) (*disbursementmethodpb.DeleteDisbursementMethodResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("disbursement_method ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete disbursement_method: %w", err)
	}

	return &disbursementmethodpb.DeleteDisbursementMethodResponse{
		Success: true,
	}, nil
}

// ListDisbursementMethods lists disbursement_method records with optional filters.
func (r *PostgresDisbursementMethodRepository) ListDisbursementMethods(ctx context.Context, req *disbursementmethodpb.ListDisbursementMethodsRequest) (*disbursementmethodpb.ListDisbursementMethodsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && (req.Filters != nil || req.Pagination != nil) {
		params = &interfaces.ListParams{Filters: req.GetFilters(), Pagination: req.GetPagination()}
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list disbursement_methods: %w", err)
	}

	methods := make([]*disbursementmethodpb.DisbursementMethod, 0, len(listResult.Data))
	for _, result := range listResult.Data {
		dm, err := mapToDisbursementMethod(result)
		if err != nil {
			log.Printf("WARN: disbursement_method list row decode: %v", err)
			continue
		}
		methods = append(methods, dm)
	}

	return &disbursementmethodpb.ListDisbursementMethodsResponse{
		Success: true,
		Data:    methods,
	}, nil
}

// GetDisbursementMethodListPageData lists disbursement_methods with pagination metadata.
func (r *PostgresDisbursementMethodRepository) GetDisbursementMethodListPageData(ctx context.Context, req *disbursementmethodpb.GetDisbursementMethodListPageDataRequest) (*disbursementmethodpb.GetDisbursementMethodListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get disbursement_method list page data request is required")
	}

	params := &interfaces.ListParams{
		Filters:    req.GetFilters(),
		Pagination: req.GetPagination(),
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to query disbursement_method list page data: %w", err)
	}

	methods := make([]*disbursementmethodpb.DisbursementMethod, 0, len(listResult.Data))
	for _, result := range listResult.Data {
		dm, err := mapToDisbursementMethod(result)
		if err != nil {
			log.Printf("WARN: disbursement_method page-data row decode: %v", err)
			continue
		}
		methods = append(methods, dm)
	}

	resp := &disbursementmethodpb.GetDisbursementMethodListPageDataResponse{
		DisbursementMethodList: methods,
		Success:                true,
	}
	if listResult.Pagination != nil {
		resp.Pagination = listResult.Pagination
	} else {
		total := listResult.Total
		resp.Pagination = &commonpb.PaginationResponse{TotalItems: total}
	}
	return resp, nil
}

// GetDisbursementMethodItemPageData retrieves a single disbursement_method by ID.
func (r *PostgresDisbursementMethodRepository) GetDisbursementMethodItemPageData(ctx context.Context, req *disbursementmethodpb.GetDisbursementMethodItemPageDataRequest) (*disbursementmethodpb.GetDisbursementMethodItemPageDataResponse, error) {
	if req == nil || req.DisbursementMethodId == "" {
		return nil, fmt.Errorf("disbursement_method ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.DisbursementMethodId)
	if err != nil {
		return nil, fmt.Errorf("failed to read disbursement_method item page data: %w", err)
	}

	dm, err := mapToDisbursementMethod(result)
	if err != nil {
		return nil, err
	}

	return &disbursementmethodpb.GetDisbursementMethodItemPageDataResponse{
		DisbursementMethod: dm,
		Success:            true,
	}, nil
}

// NewDisbursementMethodRepository creates a new PostgreSQL disbursement_method repository (old-style constructor).
func NewDisbursementMethodRepository(db *sql.DB, tableName string) disbursementmethodpb.DisbursementMethodDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresDisbursementMethodRepository(dbOps, tableName)
}
