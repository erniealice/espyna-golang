//go:build postgresql

package ledger

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
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ChargePolicyVersion, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres charge_policy_version repository requires *sql.DB, got %T", conn)
		}
		// Audited + workspace-aware: the generic write's diff-audit joins the caller's
		// transaction and every by-id/list query carries the trusted workspace predicate
		// (entity registered workspaceScopeDirectRequired — no empty-workspace wildcard).
		dbOps := postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db))
		return NewPostgresChargePolicyVersionRepository(dbOps, tableName), nil
	})
}

// PostgresChargePolicyVersionRepository implements charge_policy_version CRUD via the generic workspace-aware operations.
type PostgresChargePolicyVersionRepository struct {
	versionpb.UnimplementedChargePolicyVersionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresChargePolicyVersionRepository creates a charge_policy_version repository.
func NewPostgresChargePolicyVersionRepository(dbOps interfaces.DatabaseOperation, tableName string) versionpb.ChargePolicyVersionDomainServiceServer {
	if tableName == "" {
		tableName = entityid.ChargePolicyVersion
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresChargePolicyVersionRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresChargePolicyVersionRepository) CreateChargePolicyVersion(ctx context.Context, req *versionpb.CreateChargePolicyVersionRequest) (*versionpb.CreateChargePolicyVersionResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("charge_policy_version data is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create charge_policy_version: %w", err)
	}
	item := &versionpb.ChargePolicyVersion{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &versionpb.CreateChargePolicyVersionResponse{Data: []*versionpb.ChargePolicyVersion{item}, Success: true}, nil
}

func (r *PostgresChargePolicyVersionRepository) ReadChargePolicyVersion(ctx context.Context, req *versionpb.ReadChargePolicyVersionRequest) (*versionpb.ReadChargePolicyVersionResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_policy_version ID is required")
	}
	res, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read charge_policy_version: %w", err)
	}
	item := &versionpb.ChargePolicyVersion{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &versionpb.ReadChargePolicyVersionResponse{Data: []*versionpb.ChargePolicyVersion{item}, Success: true}, nil
}

func (r *PostgresChargePolicyVersionRepository) UpdateChargePolicyVersion(ctx context.Context, req *versionpb.UpdateChargePolicyVersionRequest) (*versionpb.UpdateChargePolicyVersionResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_policy_version ID is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update charge_policy_version: %w", err)
	}
	item := &versionpb.ChargePolicyVersion{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &versionpb.UpdateChargePolicyVersionResponse{Data: []*versionpb.ChargePolicyVersion{item}, Success: true}, nil
}

func (r *PostgresChargePolicyVersionRepository) DeleteChargePolicyVersion(ctx context.Context, req *versionpb.DeleteChargePolicyVersionRequest) (*versionpb.DeleteChargePolicyVersionResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_policy_version ID is required")
	}
	if err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete charge_policy_version: %w", err)
	}
	return &versionpb.DeleteChargePolicyVersionResponse{Success: true}, nil
}

func (r *PostgresChargePolicyVersionRepository) ListChargePolicyVersions(ctx context.Context, req *versionpb.ListChargePolicyVersionsRequest) (*versionpb.ListChargePolicyVersionsResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), andScopeFilter(req.GetFilters(), "charge_policy_id", req.ChargePolicyId), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &versionpb.ListChargePolicyVersionsResponse{Data: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresChargePolicyVersionRepository) GetChargePolicyVersionListPageData(ctx context.Context, req *versionpb.GetChargePolicyVersionListPageDataRequest) (*versionpb.GetChargePolicyVersionListPageDataResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &versionpb.GetChargePolicyVersionListPageDataResponse{ChargePolicyVersionList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresChargePolicyVersionRepository) GetChargePolicyVersionItemPageData(ctx context.Context, req *versionpb.GetChargePolicyVersionItemPageDataRequest) (*versionpb.GetChargePolicyVersionItemPageDataResponse, error) {
	if req == nil || req.ChargePolicyVersionId == "" {
		return nil, fmt.Errorf("charge_policy_version ID is required")
	}
	res, err := r.dbOps.Read(ctx, r.tableName, req.ChargePolicyVersionId)
	if err != nil {
		return nil, fmt.Errorf("failed to read charge_policy_version: %w", err)
	}
	item := &versionpb.ChargePolicyVersion{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &versionpb.GetChargePolicyVersionItemPageDataResponse{ChargePolicyVersion: item, Success: true}, nil
}

func (r *PostgresChargePolicyVersionRepository) listPage(ctx context.Context, params *interfaces.ListParams) ([]*versionpb.ChargePolicyVersion, *commonpb.PaginationResponse, error) {
	lr, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list charge_policy_version: %w", err)
	}
	items := make([]*versionpb.ChargePolicyVersion, 0, len(lr.Data))
	for _, row := range lr.Data {
		item := &versionpb.ChargePolicyVersion{}
		if err := postgresCore.ScopedResultToProto(row, item); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	return items, lr.Pagination, nil
}

// LockChargePolicyVersionForUpdate takes the version row FOR UPDATE inside the ambient
// transaction (domainports.ChargePolicyVersionLocker).
func (r *PostgresChargePolicyVersionRepository) LockChargePolicyVersionForUpdate(ctx context.Context, id string) (*versionpb.ChargePolicyVersion, error) {
	if err := postgresCore.LockScopedRowForUpdate(ctx, r.dbOps, r.tableName, id); err != nil {
		return nil, err
	}
	res, err := r.dbOps.Read(ctx, r.tableName, id)
	if err != nil {
		return nil, fmt.Errorf("charge_policy_version lock: %w", err)
	}
	item := &versionpb.ChargePolicyVersion{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return item, nil
}
