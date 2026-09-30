//go:build postgresql

package treasury

import (
	"context"
	"database/sql"
	"fmt"

	auditadapter "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/audit"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

// CollectionApplication: Slice B known-cost recovery (20260927-usage-and-pass-through-charges, build-spec §6.2).
// Every by-id and list operation goes through the workspace-aware operations, entity registered
// workspaceScopeDirectRequired: a trusted, non-empty workspace predicate, no wildcard.

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.CollectionApplication, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres collection_application repository requires *sql.DB, got %T", conn)
		}
		// Audited + workspace-aware: diff-audit joins the caller's transaction.
		dbOps := postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db))
		return NewPostgresCollectionApplicationRepository(dbOps, tableName), nil
	})
}

// PostgresCollectionApplicationRepository implements collection_application CRUD via the generic workspace-aware operations.
type PostgresCollectionApplicationRepository struct {
	collectionapplicationpb.UnimplementedCollectionApplicationDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresCollectionApplicationRepository creates a collection_application repository.
func NewPostgresCollectionApplicationRepository(dbOps interfaces.DatabaseOperation, tableName string) collectionapplicationpb.CollectionApplicationDomainServiceServer {
	if tableName == "" {
		tableName = entityid.CollectionApplication
	}
	return &PostgresCollectionApplicationRepository{dbOps: dbOps, tableName: tableName}
}

func (r *PostgresCollectionApplicationRepository) readByID(ctx context.Context, id string) (*collectionapplicationpb.CollectionApplication, error) {
	res, err := r.dbOps.Read(ctx, r.tableName, id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection_application: %w", err)
	}
	item := &collectionapplicationpb.CollectionApplication{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *PostgresCollectionApplicationRepository) CreateCollectionApplication(ctx context.Context, req *collectionapplicationpb.CreateCollectionApplicationRequest) (*collectionapplicationpb.CreateCollectionApplicationResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("collection_application data is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection_application: %w", err)
	}
	item := &collectionapplicationpb.CollectionApplication{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &collectionapplicationpb.CreateCollectionApplicationResponse{Data: []*collectionapplicationpb.CollectionApplication{item}, Success: true}, nil
}

func (r *PostgresCollectionApplicationRepository) ReadCollectionApplication(ctx context.Context, req *collectionapplicationpb.ReadCollectionApplicationRequest) (*collectionapplicationpb.ReadCollectionApplicationResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_application ID is required")
	}
	item, err := r.readByID(ctx, req.Data.Id)
	if err != nil {
		return nil, err
	}
	return &collectionapplicationpb.ReadCollectionApplicationResponse{Data: []*collectionapplicationpb.CollectionApplication{item}, Success: true}, nil
}

func (r *PostgresCollectionApplicationRepository) UpdateCollectionApplication(ctx context.Context, req *collectionapplicationpb.UpdateCollectionApplicationRequest) (*collectionapplicationpb.UpdateCollectionApplicationResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_application ID is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update collection_application: %w", err)
	}
	item := &collectionapplicationpb.CollectionApplication{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &collectionapplicationpb.UpdateCollectionApplicationResponse{Data: []*collectionapplicationpb.CollectionApplication{item}, Success: true}, nil
}

// DeleteCollectionApplication is refused: collection_application rows are immutable financial records, corrected by a new
// row and never removed at the port (build-spec §7 C6).
func (r *PostgresCollectionApplicationRepository) DeleteCollectionApplication(ctx context.Context, req *collectionapplicationpb.DeleteCollectionApplicationRequest) (*collectionapplicationpb.DeleteCollectionApplicationResponse, error) {
	return nil, postgresCore.ErrImmutableRecord(r.tableName)
}

func (r *PostgresCollectionApplicationRepository) ListCollectionApplications(ctx context.Context, req *collectionapplicationpb.ListCollectionApplicationsRequest) (*collectionapplicationpb.ListCollectionApplicationsResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, _, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &collectionapplicationpb.ListCollectionApplicationsResponse{Data: items, Success: true}, nil
}

func (r *PostgresCollectionApplicationRepository) GetCollectionApplicationListPageData(ctx context.Context, req *collectionapplicationpb.GetCollectionApplicationListPageDataRequest) (*collectionapplicationpb.GetCollectionApplicationListPageDataResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &collectionapplicationpb.GetCollectionApplicationListPageDataResponse{CollectionApplicationList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresCollectionApplicationRepository) GetCollectionApplicationItemPageData(ctx context.Context, req *collectionapplicationpb.GetCollectionApplicationItemPageDataRequest) (*collectionapplicationpb.GetCollectionApplicationItemPageDataResponse, error) {
	if req == nil || req.CollectionApplicationId == "" {
		return nil, fmt.Errorf("collection_application ID is required")
	}
	item, err := r.readByID(ctx, req.CollectionApplicationId)
	if err != nil {
		return nil, err
	}
	return &collectionapplicationpb.GetCollectionApplicationItemPageDataResponse{CollectionApplication: item, Success: true}, nil
}

func (r *PostgresCollectionApplicationRepository) listPage(ctx context.Context, params *interfaces.ListParams) ([]*collectionapplicationpb.CollectionApplication, *commonpb.PaginationResponse, error) {
	lr, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list collection_application: %w", err)
	}
	items := make([]*collectionapplicationpb.CollectionApplication, 0, len(lr.Data))
	for _, row := range lr.Data {
		item := &collectionapplicationpb.CollectionApplication{}
		if err := postgresCore.ScopedResultToProto(row, item); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	return items, lr.Pagination, nil
}

// LockCollectionApplicationForUpdate takes one application row FOR UPDATE inside the ambient
// transaction (domainports.CollectionApplicationLocker); workspace-scoped, fail closed without a
// transaction, foreign or missing row = not found.
func (r *PostgresCollectionApplicationRepository) LockCollectionApplicationForUpdate(ctx context.Context, id string) (*collectionapplicationpb.CollectionApplication, error) {
	if err := postgresCore.LockScopedRowForUpdate(ctx, r.dbOps, r.tableName, id); err != nil {
		return nil, err
	}
	return r.readByID(ctx, id)
}
