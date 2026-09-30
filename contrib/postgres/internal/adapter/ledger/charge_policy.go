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
	policypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	"github.com/lib/pq"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ChargePolicy, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres charge_policy repository requires *sql.DB, got %T", conn)
		}
		// Audited + workspace-aware: the generic write's diff-audit joins the caller's
		// transaction and every by-id/list query carries the trusted workspace predicate
		// (entity registered workspaceScopeDirectRequired — no empty-workspace wildcard).
		dbOps := postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db))
		return NewPostgresChargePolicyRepository(dbOps, tableName), nil
	})
}

// PostgresChargePolicyRepository implements charge_policy CRUD via the generic workspace-aware operations.
type PostgresChargePolicyRepository struct {
	policypb.UnimplementedChargePolicyDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresChargePolicyRepository creates a charge_policy repository.
func NewPostgresChargePolicyRepository(dbOps interfaces.DatabaseOperation, tableName string) policypb.ChargePolicyDomainServiceServer {
	if tableName == "" {
		tableName = entityid.ChargePolicy
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresChargePolicyRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresChargePolicyRepository) CreateChargePolicy(ctx context.Context, req *policypb.CreateChargePolicyRequest) (*policypb.CreateChargePolicyResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("charge_policy data is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create charge_policy: %w", err)
	}
	item := &policypb.ChargePolicy{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &policypb.CreateChargePolicyResponse{Data: []*policypb.ChargePolicy{item}, Success: true}, nil
}

func (r *PostgresChargePolicyRepository) ReadChargePolicy(ctx context.Context, req *policypb.ReadChargePolicyRequest) (*policypb.ReadChargePolicyResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_policy ID is required")
	}
	res, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read charge_policy: %w", err)
	}
	item := &policypb.ChargePolicy{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &policypb.ReadChargePolicyResponse{Data: []*policypb.ChargePolicy{item}, Success: true}, nil
}

func (r *PostgresChargePolicyRepository) UpdateChargePolicy(ctx context.Context, req *policypb.UpdateChargePolicyRequest) (*policypb.UpdateChargePolicyResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_policy ID is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update charge_policy: %w", err)
	}
	item := &policypb.ChargePolicy{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &policypb.UpdateChargePolicyResponse{Data: []*policypb.ChargePolicy{item}, Success: true}, nil
}

func (r *PostgresChargePolicyRepository) DeleteChargePolicy(ctx context.Context, req *policypb.DeleteChargePolicyRequest) (*policypb.DeleteChargePolicyResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_policy ID is required")
	}
	if err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete charge_policy: %w", err)
	}
	return &policypb.DeleteChargePolicyResponse{Success: true}, nil
}

func (r *PostgresChargePolicyRepository) ListChargePolicies(ctx context.Context, req *policypb.ListChargePoliciesRequest) (*policypb.ListChargePoliciesResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, _, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &policypb.ListChargePoliciesResponse{Data: items, Success: true}, nil
}

func (r *PostgresChargePolicyRepository) GetChargePolicyListPageData(ctx context.Context, req *policypb.GetChargePolicyListPageDataRequest) (*policypb.GetChargePolicyListPageDataResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &policypb.GetChargePolicyListPageDataResponse{ChargePolicyList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresChargePolicyRepository) GetChargePolicyItemPageData(ctx context.Context, req *policypb.GetChargePolicyItemPageDataRequest) (*policypb.GetChargePolicyItemPageDataResponse, error) {
	if req == nil || req.ChargePolicyId == "" {
		return nil, fmt.Errorf("charge_policy ID is required")
	}
	res, err := r.dbOps.Read(ctx, r.tableName, req.ChargePolicyId)
	if err != nil {
		return nil, fmt.Errorf("failed to read charge_policy: %w", err)
	}
	item := &policypb.ChargePolicy{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &policypb.GetChargePolicyItemPageDataResponse{ChargePolicy: item, Success: true}, nil
}

func (r *PostgresChargePolicyRepository) listPage(ctx context.Context, params *interfaces.ListParams) ([]*policypb.ChargePolicy, *commonpb.PaginationResponse, error) {
	lr, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list charge_policy: %w", err)
	}
	items := make([]*policypb.ChargePolicy, 0, len(lr.Data))
	for _, row := range lr.Data {
		item := &policypb.ChargePolicy{}
		if err := postgresCore.ScopedResultToProto(row, item); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	return items, lr.Pagination, nil
}

// LockChargePolicyForUpdate takes the policy row FOR UPDATE inside the ambient transaction
// (domainports.ChargePolicyLocker). Fails closed without a transaction or trusted workspace,
// and reports a foreign-workspace row as not found.
func (r *PostgresChargePolicyRepository) LockChargePolicyForUpdate(ctx context.Context, id string) (*policypb.ChargePolicy, error) {
	if err := postgresCore.LockScopedRowForUpdate(ctx, r.dbOps, r.tableName, id); err != nil {
		return nil, err
	}
	res, err := r.dbOps.Read(ctx, r.tableName, id)
	if err != nil {
		return nil, fmt.Errorf("charge_policy lock: %w", err)
	}
	item := &policypb.ChargePolicy{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return item, nil
}

// ChargePolicyIDsReferencedByPricePlans reports which of the given policies are referenced by
// a product_price_plan row (domainports.ChargePolicyReferenceReader). product_price_plan has no
// workspace_id column (its tenancy is inherited from price_plan), so the workspace predicate
// is anchored on the policy row: only policies of the caller's workspace can be reported.
func (r *PostgresChargePolicyRepository) ChargePolicyIDsReferencedByPricePlans(ctx context.Context, policyIDs []string) (map[string]bool, error) {
	out := map[string]bool{}
	if len(policyIDs) == 0 {
		return out, nil
	}
	wsID, err := postgresCore.RequireTrustedWorkspace(ctx)
	if err != nil {
		return nil, err
	}
	exec := postgresCore.TxExecutor(ctx, r.dbOps)
	if exec == nil && r.db != nil {
		exec = r.db
	}
	if exec == nil {
		return nil, fmt.Errorf("charge_policy reference read: no SQL executor available")
	}
	rows, err := exec.QueryContext(ctx,
		`SELECT DISTINCT ppp.charge_policy_id FROM product_price_plan ppp JOIN charge_policy cp ON cp.id = ppp.charge_policy_id WHERE cp.workspace_id = $1 AND ppp.charge_policy_id = ANY($2)`,
		wsID, pq.Array(policyIDs))
	if err != nil {
		return nil, fmt.Errorf("charge_policy reference read: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}
