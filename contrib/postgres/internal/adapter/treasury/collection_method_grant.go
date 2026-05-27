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
	grantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_grant"
)

// Treasury-domain-rebuild Stage 3: the postgres collection_method_grant repository.
// The table (migration 20260527020000) carries only scalar / enum columns — this is
// a CONFIG entity (Q6: NO usage_count / last_used_at / redemption_count / any event
// counter), so the generic protojson round-trip (protojson -> map -> dbOps; result ->
// protojson) carries every field faithfully with no per-column scan. Mirrors
// collection_method_eligibility_rule.go. Grants do not mutate — there is no Update;
// the only state change is ACTIVE → REVOKED via RevokeCollectionMethodGrant.
func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.CollectionMethodGrant, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres collection_method_grant repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresCollectionMethodGrantRepository(dbOps, tableName), nil
	})
}

// PostgresCollectionMethodGrantRepository implements collection_method_grant
// create / read / revoke / list / bulk-grant using PostgreSQL.
type PostgresCollectionMethodGrantRepository struct {
	grantpb.UnimplementedCollectionMethodGrantDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresCollectionMethodGrantRepository creates a new PostgreSQL collection_method_grant repository.
func NewPostgresCollectionMethodGrantRepository(dbOps interfaces.DatabaseOperation, tableName string) grantpb.CollectionMethodGrantDomainServiceServer {
	if tableName == "" {
		tableName = "collection_method_grant"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresCollectionMethodGrantRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func collectionMethodGrantToMap(grant *grantpb.CollectionMethodGrant) (map[string]any, error) {
	jsonData, err := protojson.Marshal(grant)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collection_method_grant protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal collection_method_grant JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")
	return data, nil
}

func mapToCollectionMethodGrant(result map[string]any) (*grantpb.CollectionMethodGrant, error) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collection_method_grant result to JSON: %w", err)
	}
	grant := &grantpb.CollectionMethodGrant{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, grant); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to collection_method_grant protobuf: %w", err)
	}
	return grant, nil
}

// CreateCollectionMethodGrant creates a new collection_method_grant record.
func (r *PostgresCollectionMethodGrantRepository) CreateCollectionMethodGrant(ctx context.Context, req *grantpb.CreateCollectionMethodGrantRequest) (*grantpb.CreateCollectionMethodGrantResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("collection_method_grant data is required")
	}

	data, err := collectionMethodGrantToMap(req.Data)
	if err != nil {
		return nil, err
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection_method_grant: %w", err)
	}

	grant, err := mapToCollectionMethodGrant(result)
	if err != nil {
		return nil, err
	}

	return &grantpb.CreateCollectionMethodGrantResponse{
		Success: true,
		Data:    []*grantpb.CollectionMethodGrant{grant},
	}, nil
}

// ReadCollectionMethodGrant retrieves a collection_method_grant record by ID.
func (r *PostgresCollectionMethodGrantRepository) ReadCollectionMethodGrant(ctx context.Context, req *grantpb.ReadCollectionMethodGrantRequest) (*grantpb.ReadCollectionMethodGrantResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_method_grant ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection_method_grant: %w", err)
	}

	grant, err := mapToCollectionMethodGrant(result)
	if err != nil {
		return nil, err
	}

	return &grantpb.ReadCollectionMethodGrantResponse{
		Success: true,
		Data:    []*grantpb.CollectionMethodGrant{grant},
	}, nil
}

// RevokeCollectionMethodGrant flips an existing grant's status ACTIVE → REVOKED.
// Grants do not mutate; this is the ONLY state change a grant undergoes. The
// repository performs a read-modify-write that updates only the status + revoke
// audit fields, leaving every other field as stored.
func (r *PostgresCollectionMethodGrantRepository) RevokeCollectionMethodGrant(ctx context.Context, req *grantpb.RevokeCollectionMethodGrantRequest) (*grantpb.RevokeCollectionMethodGrantResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_method_grant ID is required")
	}

	existing, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection_method_grant for revoke: %w", err)
	}

	grant, err := mapToCollectionMethodGrant(existing)
	if err != nil {
		return nil, err
	}

	// Only the status + revoke audit fields change; the grant body is immutable.
	grant.Status = grantpb.CollectionMethodGrantStatus_COLLECTION_METHOD_GRANT_STATUS_REVOKED
	if req.Data.RevokedByUserId != nil {
		grant.RevokedByUserId = req.Data.RevokedByUserId
	}
	if req.Data.RevokeReason != nil {
		grant.RevokeReason = req.Data.RevokeReason
	}

	data, err := collectionMethodGrantToMap(grant)
	if err != nil {
		return nil, err
	}

	result, err := r.dbOps.Update(ctx, r.tableName, grant.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke collection_method_grant: %w", err)
	}

	updated, err := mapToCollectionMethodGrant(result)
	if err != nil {
		return nil, err
	}

	return &grantpb.RevokeCollectionMethodGrantResponse{
		Success: true,
		Data:    []*grantpb.CollectionMethodGrant{updated},
	}, nil
}

// ListCollectionMethodGrants lists collection_method_grant records with optional filters.
func (r *PostgresCollectionMethodGrantRepository) ListCollectionMethodGrants(ctx context.Context, req *grantpb.ListCollectionMethodGrantsRequest) (*grantpb.ListCollectionMethodGrantsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && (req.Filters != nil || req.Pagination != nil) {
		params = &interfaces.ListParams{Filters: req.GetFilters(), Pagination: req.GetPagination()}
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list collection_method_grants: %w", err)
	}

	grants := make([]*grantpb.CollectionMethodGrant, 0, len(listResult.Data))
	for _, result := range listResult.Data {
		grant, err := mapToCollectionMethodGrant(result)
		if err != nil {
			log.Printf("WARN: collection_method_grant list row decode: %v", err)
			continue
		}
		grants = append(grants, grant)
	}

	return &grantpb.ListCollectionMethodGrantsResponse{
		Success: true,
		Data:    grants,
	}, nil
}

// BulkGrantCollectionMethodGrants creates many grants in one call (one CM template,
// many clients). Each row is inserted; the per-row results are aggregated. The
// audience-mode guardrail is evaluated at the use-case layer, not here.
func (r *PostgresCollectionMethodGrantRepository) BulkGrantCollectionMethodGrants(ctx context.Context, req *grantpb.BulkGrantCollectionMethodGrantsRequest) (*grantpb.BulkGrantCollectionMethodGrantsResponse, error) {
	if req == nil || len(req.Data) == 0 {
		return nil, fmt.Errorf("collection_method_grant bulk data is required")
	}

	created := make([]*grantpb.CollectionMethodGrant, 0, len(req.Data))
	for _, grant := range req.Data {
		if grant == nil {
			continue
		}
		data, err := collectionMethodGrantToMap(grant)
		if err != nil {
			return nil, err
		}
		result, err := r.dbOps.Create(ctx, r.tableName, data)
		if err != nil {
			return nil, fmt.Errorf("failed to bulk-create collection_method_grant: %w", err)
		}
		row, err := mapToCollectionMethodGrant(result)
		if err != nil {
			return nil, err
		}
		created = append(created, row)
	}

	return &grantpb.BulkGrantCollectionMethodGrantsResponse{
		Success: true,
		Data:    created,
	}, nil
}

// GetCollectionMethodGrantListPageData lists collection_method_grants with pagination metadata.
func (r *PostgresCollectionMethodGrantRepository) GetCollectionMethodGrantListPageData(ctx context.Context, req *grantpb.GetCollectionMethodGrantListPageDataRequest) (*grantpb.GetCollectionMethodGrantListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get collection_method_grant list page data request is required")
	}

	params := &interfaces.ListParams{
		Filters:    req.GetFilters(),
		Pagination: req.GetPagination(),
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection_method_grant list page data: %w", err)
	}

	grants := make([]*grantpb.CollectionMethodGrant, 0, len(listResult.Data))
	for _, result := range listResult.Data {
		grant, err := mapToCollectionMethodGrant(result)
		if err != nil {
			log.Printf("WARN: collection_method_grant page-data row decode: %v", err)
			continue
		}
		grants = append(grants, grant)
	}

	resp := &grantpb.GetCollectionMethodGrantListPageDataResponse{
		CollectionMethodGrantList: grants,
		Success:                   true,
	}
	if listResult.Pagination != nil {
		resp.Pagination = listResult.Pagination
	} else {
		total := listResult.Total
		resp.Pagination = &commonpb.PaginationResponse{TotalItems: total}
	}
	return resp, nil
}

// GetCollectionMethodGrantItemPageData retrieves a single collection_method_grant by ID.
func (r *PostgresCollectionMethodGrantRepository) GetCollectionMethodGrantItemPageData(ctx context.Context, req *grantpb.GetCollectionMethodGrantItemPageDataRequest) (*grantpb.GetCollectionMethodGrantItemPageDataResponse, error) {
	if req == nil || req.CollectionMethodGrantId == "" {
		return nil, fmt.Errorf("collection_method_grant ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.CollectionMethodGrantId)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection_method_grant item page data: %w", err)
	}

	grant, err := mapToCollectionMethodGrant(result)
	if err != nil {
		return nil, err
	}

	return &grantpb.GetCollectionMethodGrantItemPageDataResponse{
		CollectionMethodGrant: grant,
		Success:               true,
	}, nil
}

// NewCollectionMethodGrantRepository creates a new PostgreSQL collection_method_grant repository (old-style constructor).
func NewCollectionMethodGrantRepository(db *sql.DB, tableName string) grantpb.CollectionMethodGrantDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresCollectionMethodGrantRepository(dbOps, tableName)
}
