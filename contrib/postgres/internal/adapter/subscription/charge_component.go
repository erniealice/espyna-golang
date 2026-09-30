//go:build postgresql

package subscription

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
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
)

// ChargeComponent: Slice B known-cost recovery (20260927-usage-and-pass-through-charges, build-spec §6.2).
// Every by-id and list operation goes through the workspace-aware operations, entity registered
// workspaceScopeDirectRequired: a trusted, non-empty workspace predicate, no wildcard.

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ChargeComponent, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres charge_component repository requires *sql.DB, got %T", conn)
		}
		// Audited + workspace-aware: diff-audit joins the caller's transaction.
		dbOps := postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db))
		return NewPostgresChargeComponentRepository(dbOps, tableName), nil
	})
}

// PostgresChargeComponentRepository implements charge_component CRUD via the generic workspace-aware operations.
type PostgresChargeComponentRepository struct {
	chargecomponentpb.UnimplementedChargeComponentDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresChargeComponentRepository creates a charge_component repository.
func NewPostgresChargeComponentRepository(dbOps interfaces.DatabaseOperation, tableName string) chargecomponentpb.ChargeComponentDomainServiceServer {
	if tableName == "" {
		tableName = entityid.ChargeComponent
	}
	return &PostgresChargeComponentRepository{dbOps: dbOps, tableName: tableName}
}

func (r *PostgresChargeComponentRepository) readByID(ctx context.Context, id string) (*chargecomponentpb.ChargeComponent, error) {
	res, err := r.dbOps.Read(ctx, r.tableName, id)
	if err != nil {
		return nil, fmt.Errorf("failed to read charge_component: %w", err)
	}
	item := &chargecomponentpb.ChargeComponent{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *PostgresChargeComponentRepository) CreateChargeComponent(ctx context.Context, req *chargecomponentpb.CreateChargeComponentRequest) (*chargecomponentpb.CreateChargeComponentResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("charge_component data is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create charge_component: %w", err)
	}
	item := &chargecomponentpb.ChargeComponent{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &chargecomponentpb.CreateChargeComponentResponse{Data: []*chargecomponentpb.ChargeComponent{item}, Success: true}, nil
}

func (r *PostgresChargeComponentRepository) ReadChargeComponent(ctx context.Context, req *chargecomponentpb.ReadChargeComponentRequest) (*chargecomponentpb.ReadChargeComponentResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_component ID is required")
	}
	item, err := r.readByID(ctx, req.Data.Id)
	if err != nil {
		return nil, err
	}
	return &chargecomponentpb.ReadChargeComponentResponse{Data: []*chargecomponentpb.ChargeComponent{item}, Success: true}, nil
}

func (r *PostgresChargeComponentRepository) UpdateChargeComponent(ctx context.Context, req *chargecomponentpb.UpdateChargeComponentRequest) (*chargecomponentpb.UpdateChargeComponentResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_component ID is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update charge_component: %w", err)
	}
	item := &chargecomponentpb.ChargeComponent{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &chargecomponentpb.UpdateChargeComponentResponse{Data: []*chargecomponentpb.ChargeComponent{item}, Success: true}, nil
}

// DeleteChargeComponent is refused (C6): a charge component is part of an immutable financial
// row (its billable charge) and is never removed at the port.
func (r *PostgresChargeComponentRepository) DeleteChargeComponent(ctx context.Context, req *chargecomponentpb.DeleteChargeComponentRequest) (*chargecomponentpb.DeleteChargeComponentResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_component ID is required")
	}
	return nil, postgresCore.ErrImmutableRecord("charge_component")
}

func (r *PostgresChargeComponentRepository) ListChargeComponents(ctx context.Context, req *chargecomponentpb.ListChargeComponentsRequest) (*chargecomponentpb.ListChargeComponentsResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, _, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &chargecomponentpb.ListChargeComponentsResponse{Data: items, Success: true}, nil
}

func (r *PostgresChargeComponentRepository) GetChargeComponentListPageData(ctx context.Context, req *chargecomponentpb.GetChargeComponentListPageDataRequest) (*chargecomponentpb.GetChargeComponentListPageDataResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &chargecomponentpb.GetChargeComponentListPageDataResponse{ChargeComponentList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresChargeComponentRepository) GetChargeComponentItemPageData(ctx context.Context, req *chargecomponentpb.GetChargeComponentItemPageDataRequest) (*chargecomponentpb.GetChargeComponentItemPageDataResponse, error) {
	if req == nil || req.ChargeComponentId == "" {
		return nil, fmt.Errorf("charge_component ID is required")
	}
	item, err := r.readByID(ctx, req.ChargeComponentId)
	if err != nil {
		return nil, err
	}
	return &chargecomponentpb.GetChargeComponentItemPageDataResponse{ChargeComponent: item, Success: true}, nil
}

func (r *PostgresChargeComponentRepository) listPage(ctx context.Context, params *interfaces.ListParams) ([]*chargecomponentpb.ChargeComponent, *commonpb.PaginationResponse, error) {
	lr, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list charge_component: %w", err)
	}
	items := make([]*chargecomponentpb.ChargeComponent, 0, len(lr.Data))
	for _, row := range lr.Data {
		item := &chargecomponentpb.ChargeComponent{}
		if err := postgresCore.ScopedResultToProto(row, item); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	return items, lr.Pagination, nil
}
