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
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// collection_method is a lookup/reference table (id, name, provider_name,
// active) with NO workspace_id — see migrations/postgres baseline
// public.collection_method. It therefore mirrors loan_payment.go (no
// multi-tenant filter, self-contained CTE) rather than collection.go (which is
// workspace-scoped with advance_* schedule columns). The oneof method_details
// (card / bank_account) is not persisted in this table; the protojson round-trip
// CRUD path is column-filtered by PostgresOperations so the absent nested
// columns are dropped automatically.

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

// PostgresCollectionMethodRepository implements collection_method CRUD operations using PostgreSQL
type PostgresCollectionMethodRepository struct {
	collectionmethodpb.UnimplementedCollectionMethodDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresCollectionMethodRepository creates a new PostgreSQL collection_method repository
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

// CreateCollectionMethod creates a new collection_method record
func (r *PostgresCollectionMethodRepository) CreateCollectionMethod(ctx context.Context, req *collectionmethodpb.CreateCollectionMethodRequest) (*collectionmethodpb.CreateCollectionMethodResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("collection_method data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection_method: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collectionMethod := &collectionmethodpb.CollectionMethod{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, collectionMethod); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionmethodpb.CreateCollectionMethodResponse{
		Success: true,
		Data:    []*collectionmethodpb.CollectionMethod{collectionMethod},
	}, nil
}

// ReadCollectionMethod retrieves a collection_method record by ID
func (r *PostgresCollectionMethodRepository) ReadCollectionMethod(ctx context.Context, req *collectionmethodpb.ReadCollectionMethodRequest) (*collectionmethodpb.ReadCollectionMethodResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_method ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection_method: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collectionMethod := &collectionmethodpb.CollectionMethod{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, collectionMethod); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionmethodpb.ReadCollectionMethodResponse{
		Success: true,
		Data:    []*collectionmethodpb.CollectionMethod{collectionMethod},
	}, nil
}

// UpdateCollectionMethod updates a collection_method record
func (r *PostgresCollectionMethodRepository) UpdateCollectionMethod(ctx context.Context, req *collectionmethodpb.UpdateCollectionMethodRequest) (*collectionmethodpb.UpdateCollectionMethodResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_method ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update collection_method: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collectionMethod := &collectionmethodpb.CollectionMethod{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, collectionMethod); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionmethodpb.UpdateCollectionMethodResponse{
		Success: true,
		Data:    []*collectionmethodpb.CollectionMethod{collectionMethod},
	}, nil
}

// DeleteCollectionMethod deletes a collection_method record (soft delete)
func (r *PostgresCollectionMethodRepository) DeleteCollectionMethod(ctx context.Context, req *collectionmethodpb.DeleteCollectionMethodRequest) (*collectionmethodpb.DeleteCollectionMethodResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_method ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete collection_method: %w", err)
	}

	return &collectionmethodpb.DeleteCollectionMethodResponse{
		Success: true,
	}, nil
}

// ListCollectionMethods lists collection_method records with optional filters
func (r *PostgresCollectionMethodRepository) ListCollectionMethods(ctx context.Context, req *collectionmethodpb.ListCollectionMethodsRequest) (*collectionmethodpb.ListCollectionMethodsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list collection_methods: %w", err)
	}

	var collectionMethods []*collectionmethodpb.CollectionMethod
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal collection_method row: %v", err)
			continue
		}

		collectionMethod := &collectionmethodpb.CollectionMethod{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, collectionMethod); err != nil {
			log.Printf("WARN: protojson unmarshal collection_method: %v", err)
			continue
		}
		collectionMethods = append(collectionMethods, collectionMethod)
	}

	return &collectionmethodpb.ListCollectionMethodsResponse{
		Success: true,
		Data:    collectionMethods,
	}, nil
}

// collectionMethodSortableSQLCols is the fail-closed sort whitelist for
// GetCollectionMethodListPageData. Only columns projected by the CTE SELECT are
// included so ORDER BY can never reference an unprojected/injected identifier.
// The collection_method table is a minimal lookup table: id, name,
// provider_name, active (no date_* columns).
var collectionMethodSortableSQLCols = []string{
	"id", "name", "provider_name", "active",
}

// GetCollectionMethodListPageData retrieves collection_methods with pagination,
// filtering, sorting, and search using CTE. The collection_method table has no
// workspace_id column (it is a reference/lookup table), so unlike
// GetCollectionListPageData there is no multi-tenant filter.
func (r *PostgresCollectionMethodRepository) GetCollectionMethodListPageData(
	ctx context.Context,
	req *collectionmethodpb.GetCollectionMethodListPageDataRequest,
) (*collectionmethodpb.GetCollectionMethodListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get collection_method list page data request is required")
	}

	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	// Sort — fail-closed against the per-entity whitelist. An unknown sort column
	// errors instead of being interpolated verbatim into ORDER BY.
	orderByClause, err := postgresCore.BuildOrderBy(collectionMethodSortableSQLCols, req.GetSort(), "name ASC")
	if err != nil {
		return nil, err
	}

	query := `
		WITH enriched AS (
			SELECT
				cm.id,
				cm.name,
				cm.provider_name,
				cm.active
			FROM collection_method cm
			WHERE cm.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       cm.name ILIKE $1 OR
			       cm.provider_name ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		` + orderByClause + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection_method list page data: %w", err)
	}
	defer rows.Close()

	var collectionMethods []*collectionmethodpb.CollectionMethod
	var totalCount int64

	for rows.Next() {
		var (
			id           string
			name         *string
			providerName *string
			active       bool
			total        int64
		)

		if err := rows.Scan(&id, &name, &providerName, &active, &total); err != nil {
			return nil, fmt.Errorf("failed to scan collection_method row: %w", err)
		}

		totalCount = total

		collectionMethod := &collectionmethodpb.CollectionMethod{
			Id:     id,
			Active: active,
		}
		if name != nil {
			collectionMethod.Name = *name
		}
		if providerName != nil {
			collectionMethod.ProviderName = providerName
		}

		collectionMethods = append(collectionMethods, collectionMethod)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating collection_method rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &collectionmethodpb.GetCollectionMethodListPageDataResponse{
		CollectionMethodList: collectionMethods,
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

// GetCollectionMethodItemPageData retrieves a single collection_method with enriched data using CTE
func (r *PostgresCollectionMethodRepository) GetCollectionMethodItemPageData(
	ctx context.Context,
	req *collectionmethodpb.GetCollectionMethodItemPageDataRequest,
) (*collectionmethodpb.GetCollectionMethodItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get collection_method item page data request is required")
	}
	if req.CollectionMethodId == "" {
		return nil, fmt.Errorf("collection_method ID is required")
	}

	query := `
		WITH enriched AS (
			SELECT
				cm.id,
				cm.name,
				cm.provider_name,
				cm.active
			FROM collection_method cm
			WHERE cm.id = $1
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.CollectionMethodId)

	var (
		id           string
		name         *string
		providerName *string
		active       bool
	)

	err := row.Scan(&id, &name, &providerName, &active)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("collection_method with ID '%s' not found", req.CollectionMethodId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query collection_method item page data: %w", err)
	}

	collectionMethod := &collectionmethodpb.CollectionMethod{
		Id:     id,
		Active: active,
	}
	if name != nil {
		collectionMethod.Name = *name
	}
	if providerName != nil {
		collectionMethod.ProviderName = providerName
	}

	return &collectionmethodpb.GetCollectionMethodItemPageDataResponse{
		CollectionMethod: collectionMethod,
		Success:          true,
	}, nil
}

// NewCollectionMethodRepository creates a new PostgreSQL collection_method repository (old-style constructor)
func NewCollectionMethodRepository(db *sql.DB, tableName string) collectionmethodpb.CollectionMethodDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresCollectionMethodRepository(dbOps, tableName)
}
