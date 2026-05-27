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
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// Treasury-domain-rebuild Stage 1 (Wave 5): the postgres collection_method
// repository. The template-level columns (posting_kind, category,
// audience_mode, tax_effect_kind, lifecycle, source, version fields, GL
// defaults) were added by migration 20260527000000 as TEXT / INTEGER columns.
// Enum fields persist as their string name, so the generic protojson round-trip
// (protojson -> map -> dbOps; result -> protojson) carries them faithfully with
// no per-column scan. message-typed oneof variants (method_details /
// template_details) are not columns and are dropped by the marshal/round-trip.
func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.CollectionMethod, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres collection_method repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresCollectionMethodRepository(dbOps, tableName), nil
	})
}

// PostgresCollectionMethodRepository implements collection_method CRUD using PostgreSQL.
type PostgresCollectionMethodRepository struct {
	collectionmethodpb.UnimplementedCollectionMethodDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresCollectionMethodRepository creates a new PostgreSQL collection_method repository.
func NewPostgresCollectionMethodRepository(dbOps interfaces.DatabaseOperation, tableName string) collectionmethodpb.CollectionMethodDomainServiceServer {
	if tableName == "" {
		tableName = "collection_method"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresCollectionMethodRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func collectionMethodToMap(cm *collectionmethodpb.CollectionMethod) (map[string]any, error) {
	jsonData, err := protojson.Marshal(cm)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collection_method protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal collection_method JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")
	return data, nil
}

func mapToCollectionMethod(result map[string]any) (*collectionmethodpb.CollectionMethod, error) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collection_method result to JSON: %w", err)
	}
	cm := &collectionmethodpb.CollectionMethod{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to collection_method protobuf: %w", err)
	}
	return cm, nil
}

// CreateCollectionMethod creates a new collection_method record.
func (r *PostgresCollectionMethodRepository) CreateCollectionMethod(ctx context.Context, req *collectionmethodpb.CreateCollectionMethodRequest) (*collectionmethodpb.CreateCollectionMethodResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("collection_method data is required")
	}

	data, err := collectionMethodToMap(req.Data)
	if err != nil {
		return nil, err
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection_method: %w", err)
	}

	cm, err := mapToCollectionMethod(result)
	if err != nil {
		return nil, err
	}

	return &collectionmethodpb.CreateCollectionMethodResponse{
		Success: true,
		Data:    []*collectionmethodpb.CollectionMethod{cm},
	}, nil
}

// ReadCollectionMethod retrieves a collection_method record by ID.
func (r *PostgresCollectionMethodRepository) ReadCollectionMethod(ctx context.Context, req *collectionmethodpb.ReadCollectionMethodRequest) (*collectionmethodpb.ReadCollectionMethodResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_method ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection_method: %w", err)
	}

	cm, err := mapToCollectionMethod(result)
	if err != nil {
		return nil, err
	}

	return &collectionmethodpb.ReadCollectionMethodResponse{
		Success: true,
		Data:    []*collectionmethodpb.CollectionMethod{cm},
	}, nil
}

// UpdateCollectionMethod updates a collection_method record.
func (r *PostgresCollectionMethodRepository) UpdateCollectionMethod(ctx context.Context, req *collectionmethodpb.UpdateCollectionMethodRequest) (*collectionmethodpb.UpdateCollectionMethodResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_method ID is required")
	}

	data, err := collectionMethodToMap(req.Data)
	if err != nil {
		return nil, err
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update collection_method: %w", err)
	}

	cm, err := mapToCollectionMethod(result)
	if err != nil {
		return nil, err
	}

	return &collectionmethodpb.UpdateCollectionMethodResponse{
		Success: true,
		Data:    []*collectionmethodpb.CollectionMethod{cm},
	}, nil
}

// DeleteCollectionMethod deletes a collection_method record (soft delete).
func (r *PostgresCollectionMethodRepository) DeleteCollectionMethod(ctx context.Context, req *collectionmethodpb.DeleteCollectionMethodRequest) (*collectionmethodpb.DeleteCollectionMethodResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_method ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete collection_method: %w", err)
	}

	return &collectionmethodpb.DeleteCollectionMethodResponse{
		Success: true,
	}, nil
}

// ListCollectionMethods lists collection_method records with optional filters.
func (r *PostgresCollectionMethodRepository) ListCollectionMethods(ctx context.Context, req *collectionmethodpb.ListCollectionMethodsRequest) (*collectionmethodpb.ListCollectionMethodsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && (req.Filters != nil || req.Pagination != nil) {
		params = &interfaces.ListParams{Filters: req.GetFilters(), Pagination: req.GetPagination()}
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list collection_methods: %w", err)
	}

	methods := make([]*collectionmethodpb.CollectionMethod, 0, len(listResult.Data))
	for _, result := range listResult.Data {
		cm, err := mapToCollectionMethod(result)
		if err != nil {
			log.Printf("WARN: collection_method list row decode: %v", err)
			continue
		}
		methods = append(methods, cm)
	}

	return &collectionmethodpb.ListCollectionMethodsResponse{
		Success: true,
		Data:    methods,
	}, nil
}

// GetCollectionMethodListPageData lists collection_methods with pagination metadata.
func (r *PostgresCollectionMethodRepository) GetCollectionMethodListPageData(ctx context.Context, req *collectionmethodpb.GetCollectionMethodListPageDataRequest) (*collectionmethodpb.GetCollectionMethodListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get collection_method list page data request is required")
	}

	params := &interfaces.ListParams{
		Filters:    req.GetFilters(),
		Pagination: req.GetPagination(),
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection_method list page data: %w", err)
	}

	methods := make([]*collectionmethodpb.CollectionMethod, 0, len(listResult.Data))
	for _, result := range listResult.Data {
		cm, err := mapToCollectionMethod(result)
		if err != nil {
			log.Printf("WARN: collection_method page-data row decode: %v", err)
			continue
		}
		methods = append(methods, cm)
	}

	resp := &collectionmethodpb.GetCollectionMethodListPageDataResponse{
		CollectionMethodList: methods,
		Success:              true,
	}
	if listResult.Pagination != nil {
		resp.Pagination = listResult.Pagination
	} else {
		total := listResult.Total
		resp.Pagination = &commonpb.PaginationResponse{TotalItems: total}
	}
	return resp, nil
}

// GetCollectionMethodItemPageData retrieves a single collection_method by ID.
func (r *PostgresCollectionMethodRepository) GetCollectionMethodItemPageData(ctx context.Context, req *collectionmethodpb.GetCollectionMethodItemPageDataRequest) (*collectionmethodpb.GetCollectionMethodItemPageDataResponse, error) {
	if req == nil || req.CollectionMethodId == "" {
		return nil, fmt.Errorf("collection_method ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.CollectionMethodId)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection_method item page data: %w", err)
	}

	cm, err := mapToCollectionMethod(result)
	if err != nil {
		return nil, err
	}

	return &collectionmethodpb.GetCollectionMethodItemPageDataResponse{
		CollectionMethod: cm,
		Success:          true,
	}, nil
}

// NewCollectionMethodRepository creates a new PostgreSQL collection_method repository (old-style constructor).
func NewCollectionMethodRepository(db *sql.DB, tableName string) collectionmethodpb.CollectionMethodDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresCollectionMethodRepository(dbOps, tableName)
}
