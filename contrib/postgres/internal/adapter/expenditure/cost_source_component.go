//go:build postgresql

package expenditure

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
	costsourcecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
)

// CostSourceComponent: Slice B known-cost recovery (20260927-usage-and-pass-through-charges, build-spec §6.2).
// Every by-id and list operation goes through the workspace-aware operations, entity registered
// workspaceScopeDirectRequired: a trusted, non-empty workspace predicate, no wildcard.

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.CostSourceComponent, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres cost_source_component repository requires *sql.DB, got %T", conn)
		}
		// Audited + workspace-aware: diff-audit joins the caller's transaction.
		dbOps := postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db))
		return NewPostgresCostSourceComponentRepository(dbOps, tableName), nil
	})
}

// PostgresCostSourceComponentRepository implements cost_source_component CRUD via the generic workspace-aware operations.
type PostgresCostSourceComponentRepository struct {
	costsourcecomponentpb.UnimplementedCostSourceComponentDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresCostSourceComponentRepository creates a cost_source_component repository.
func NewPostgresCostSourceComponentRepository(dbOps interfaces.DatabaseOperation, tableName string) costsourcecomponentpb.CostSourceComponentDomainServiceServer {
	if tableName == "" {
		tableName = entityid.CostSourceComponent
	}
	return &PostgresCostSourceComponentRepository{dbOps: dbOps, tableName: tableName}
}

func (r *PostgresCostSourceComponentRepository) readByID(ctx context.Context, id string) (*costsourcecomponentpb.CostSourceComponent, error) {
	res, err := r.dbOps.Read(ctx, r.tableName, id)
	if err != nil {
		return nil, fmt.Errorf("failed to read cost_source_component: %w", err)
	}
	item := &costsourcecomponentpb.CostSourceComponent{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *PostgresCostSourceComponentRepository) CreateCostSourceComponent(ctx context.Context, req *costsourcecomponentpb.CreateCostSourceComponentRequest) (*costsourcecomponentpb.CreateCostSourceComponentResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("cost_source_component data is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create cost_source_component: %w", err)
	}
	item := &costsourcecomponentpb.CostSourceComponent{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &costsourcecomponentpb.CreateCostSourceComponentResponse{Data: []*costsourcecomponentpb.CostSourceComponent{item}, Success: true}, nil
}

func (r *PostgresCostSourceComponentRepository) ReadCostSourceComponent(ctx context.Context, req *costsourcecomponentpb.ReadCostSourceComponentRequest) (*costsourcecomponentpb.ReadCostSourceComponentResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("cost_source_component ID is required")
	}
	item, err := r.readByID(ctx, req.Data.Id)
	if err != nil {
		return nil, err
	}
	return &costsourcecomponentpb.ReadCostSourceComponentResponse{Data: []*costsourcecomponentpb.CostSourceComponent{item}, Success: true}, nil
}

func (r *PostgresCostSourceComponentRepository) UpdateCostSourceComponent(ctx context.Context, req *costsourcecomponentpb.UpdateCostSourceComponentRequest) (*costsourcecomponentpb.UpdateCostSourceComponentResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("cost_source_component ID is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update cost_source_component: %w", err)
	}
	item := &costsourcecomponentpb.CostSourceComponent{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &costsourcecomponentpb.UpdateCostSourceComponentResponse{Data: []*costsourcecomponentpb.CostSourceComponent{item}, Success: true}, nil
}

// DeleteCostSourceComponent is a workspace-verified HARD delete; the use cases decide when it is allowed.
func (r *PostgresCostSourceComponentRepository) DeleteCostSourceComponent(ctx context.Context, req *costsourcecomponentpb.DeleteCostSourceComponentRequest) (*costsourcecomponentpb.DeleteCostSourceComponentResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("cost_source_component ID is required")
	}
	// A claimed component (ALLOCATION or RECOGNITION) backs charges / a recognition and is never
	// removed at the port (C6, defence in depth: the use case refuses first, under the row lock).
	cur, err := r.readByID(ctx, req.Data.Id) // workspace-aware: foreign / missing ids fail here
	if err != nil {
		return nil, err
	}
	if cur.ClaimKind != nil {
		return nil, postgresCore.ErrImmutableRecord("cost_source_component")
	}
	if err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete cost_source_component: %w", err)
	}
	return &costsourcecomponentpb.DeleteCostSourceComponentResponse{Success: true}, nil
}

func (r *PostgresCostSourceComponentRepository) ListCostSourceComponents(ctx context.Context, req *costsourcecomponentpb.ListCostSourceComponentsRequest) (*costsourcecomponentpb.ListCostSourceComponentsResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, _, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &costsourcecomponentpb.ListCostSourceComponentsResponse{Data: items, Success: true}, nil
}

func (r *PostgresCostSourceComponentRepository) GetCostSourceComponentListPageData(ctx context.Context, req *costsourcecomponentpb.GetCostSourceComponentListPageDataRequest) (*costsourcecomponentpb.GetCostSourceComponentListPageDataResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &costsourcecomponentpb.GetCostSourceComponentListPageDataResponse{CostSourceComponentList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresCostSourceComponentRepository) GetCostSourceComponentItemPageData(ctx context.Context, req *costsourcecomponentpb.GetCostSourceComponentItemPageDataRequest) (*costsourcecomponentpb.GetCostSourceComponentItemPageDataResponse, error) {
	if req == nil || req.CostSourceComponentId == "" {
		return nil, fmt.Errorf("cost_source_component ID is required")
	}
	item, err := r.readByID(ctx, req.CostSourceComponentId)
	if err != nil {
		return nil, err
	}
	return &costsourcecomponentpb.GetCostSourceComponentItemPageDataResponse{CostSourceComponent: item, Success: true}, nil
}

func (r *PostgresCostSourceComponentRepository) listPage(ctx context.Context, params *interfaces.ListParams) ([]*costsourcecomponentpb.CostSourceComponent, *commonpb.PaginationResponse, error) {
	lr, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list cost_source_component: %w", err)
	}
	items := make([]*costsourcecomponentpb.CostSourceComponent, 0, len(lr.Data))
	for _, row := range lr.Data {
		item := &costsourcecomponentpb.CostSourceComponent{}
		if err := postgresCore.ScopedResultToProto(row, item); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	return items, lr.Pagination, nil
}

// LockCostSourceComponentForUpdate takes one component row FOR UPDATE inside the ambient
// transaction (domainports.CostSourceComponentLocker). WAVE 3: use this before writing a claim.
func (r *PostgresCostSourceComponentRepository) LockCostSourceComponentForUpdate(ctx context.Context, id string) (*costsourcecomponentpb.CostSourceComponent, error) {
	if err := postgresCore.LockScopedRowForUpdate(ctx, r.dbOps, r.tableName, id); err != nil {
		return nil, err
	}
	return r.readByID(ctx, id)
}

// LockCostSourceComponentsByExpenditure locks every component of an expenditure FOR UPDATE, id
// ascending (deterministic order), through the transaction executor, never r.db
// (domainports.CostSourceComponentLocker). WAVE 3: RecognizeFromExpenditure and allocation
// publish claim the shared source through this lock (build-spec §6.1 shared source claim).
func (r *PostgresCostSourceComponentRepository) LockCostSourceComponentsByExpenditure(ctx context.Context, expenditureID string) ([]*costsourcecomponentpb.CostSourceComponent, error) {
	ids, err := postgresCore.LockScopedRowIDsForUpdate(ctx, r.dbOps, r.tableName, "expenditure_id", expenditureID, "id ASC")
	if err != nil {
		return nil, err
	}
	out := make([]*costsourcecomponentpb.CostSourceComponent, 0, len(ids))
	for _, id := range ids {
		item, err := r.readByID(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}
