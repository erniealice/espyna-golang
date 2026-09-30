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
	componentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_component"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ChargePolicyComponent, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres charge_policy_component repository requires *sql.DB, got %T", conn)
		}
		// Audited + workspace-aware: the generic write's diff-audit joins the caller's
		// transaction and every by-id/list query carries the trusted workspace predicate
		// (entity registered workspaceScopeDirectRequired — no empty-workspace wildcard).
		dbOps := postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db))
		return NewPostgresChargePolicyComponentRepository(dbOps, tableName), nil
	})
}

// PostgresChargePolicyComponentRepository implements charge_policy_component CRUD via the generic workspace-aware operations.
type PostgresChargePolicyComponentRepository struct {
	componentpb.UnimplementedChargePolicyComponentDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresChargePolicyComponentRepository creates a charge_policy_component repository.
func NewPostgresChargePolicyComponentRepository(dbOps interfaces.DatabaseOperation, tableName string) componentpb.ChargePolicyComponentDomainServiceServer {
	if tableName == "" {
		tableName = entityid.ChargePolicyComponent
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresChargePolicyComponentRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresChargePolicyComponentRepository) CreateChargePolicyComponent(ctx context.Context, req *componentpb.CreateChargePolicyComponentRequest) (*componentpb.CreateChargePolicyComponentResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("charge_policy_component data is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create charge_policy_component: %w", err)
	}
	item := &componentpb.ChargePolicyComponent{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &componentpb.CreateChargePolicyComponentResponse{Data: []*componentpb.ChargePolicyComponent{item}, Success: true}, nil
}

func (r *PostgresChargePolicyComponentRepository) ReadChargePolicyComponent(ctx context.Context, req *componentpb.ReadChargePolicyComponentRequest) (*componentpb.ReadChargePolicyComponentResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_policy_component ID is required")
	}
	res, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read charge_policy_component: %w", err)
	}
	item := &componentpb.ChargePolicyComponent{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &componentpb.ReadChargePolicyComponentResponse{Data: []*componentpb.ChargePolicyComponent{item}, Success: true}, nil
}

func (r *PostgresChargePolicyComponentRepository) UpdateChargePolicyComponent(ctx context.Context, req *componentpb.UpdateChargePolicyComponentRequest) (*componentpb.UpdateChargePolicyComponentResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_policy_component ID is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update charge_policy_component: %w", err)
	}
	item := &componentpb.ChargePolicyComponent{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &componentpb.UpdateChargePolicyComponentResponse{Data: []*componentpb.ChargePolicyComponent{item}, Success: true}, nil
}

func (r *PostgresChargePolicyComponentRepository) DeleteChargePolicyComponent(ctx context.Context, req *componentpb.DeleteChargePolicyComponentRequest) (*componentpb.DeleteChargePolicyComponentResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_policy_component ID is required")
	}
	if err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete charge_policy_component: %w", err)
	}
	return &componentpb.DeleteChargePolicyComponentResponse{Success: true}, nil
}

func (r *PostgresChargePolicyComponentRepository) ListChargePolicyComponents(ctx context.Context, req *componentpb.ListChargePolicyComponentsRequest) (*componentpb.ListChargePolicyComponentsResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), andScopeFilter(req.GetFilters(), "charge_policy_version_id", req.ChargePolicyVersionId), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &componentpb.ListChargePolicyComponentsResponse{Data: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresChargePolicyComponentRepository) GetChargePolicyComponentListPageData(ctx context.Context, req *componentpb.GetChargePolicyComponentListPageDataRequest) (*componentpb.GetChargePolicyComponentListPageDataResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &componentpb.GetChargePolicyComponentListPageDataResponse{ChargePolicyComponentList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresChargePolicyComponentRepository) GetChargePolicyComponentItemPageData(ctx context.Context, req *componentpb.GetChargePolicyComponentItemPageDataRequest) (*componentpb.GetChargePolicyComponentItemPageDataResponse, error) {
	if req == nil || req.ChargePolicyComponentId == "" {
		return nil, fmt.Errorf("charge_policy_component ID is required")
	}
	res, err := r.dbOps.Read(ctx, r.tableName, req.ChargePolicyComponentId)
	if err != nil {
		return nil, fmt.Errorf("failed to read charge_policy_component: %w", err)
	}
	item := &componentpb.ChargePolicyComponent{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &componentpb.GetChargePolicyComponentItemPageDataResponse{ChargePolicyComponent: item, Success: true}, nil
}

func (r *PostgresChargePolicyComponentRepository) listPage(ctx context.Context, params *interfaces.ListParams) ([]*componentpb.ChargePolicyComponent, *commonpb.PaginationResponse, error) {
	lr, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list charge_policy_component: %w", err)
	}
	items := make([]*componentpb.ChargePolicyComponent, 0, len(lr.Data))
	for _, row := range lr.Data {
		item := &componentpb.ChargePolicyComponent{}
		if err := postgresCore.ScopedResultToProto(row, item); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	return items, lr.Pagination, nil
}
