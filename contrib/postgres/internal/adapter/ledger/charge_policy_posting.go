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
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ChargePolicyPosting, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres charge_policy_posting repository requires *sql.DB, got %T", conn)
		}
		// Audited + workspace-aware: the generic write's diff-audit joins the caller's
		// transaction and every by-id/list query carries the trusted workspace predicate
		// (entity registered workspaceScopeDirectRequired — no empty-workspace wildcard).
		dbOps := postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db))
		return NewPostgresChargePolicyPostingRepository(dbOps, tableName), nil
	})
}

// PostgresChargePolicyPostingRepository implements charge_policy_posting CRUD via the generic workspace-aware operations.
type PostgresChargePolicyPostingRepository struct {
	postingpb.UnimplementedChargePolicyPostingDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresChargePolicyPostingRepository creates a charge_policy_posting repository.
func NewPostgresChargePolicyPostingRepository(dbOps interfaces.DatabaseOperation, tableName string) postingpb.ChargePolicyPostingDomainServiceServer {
	if tableName == "" {
		tableName = entityid.ChargePolicyPosting
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresChargePolicyPostingRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresChargePolicyPostingRepository) CreateChargePolicyPosting(ctx context.Context, req *postingpb.CreateChargePolicyPostingRequest) (*postingpb.CreateChargePolicyPostingResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("charge_policy_posting data is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create charge_policy_posting: %w", err)
	}
	item := &postingpb.ChargePolicyPosting{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &postingpb.CreateChargePolicyPostingResponse{Data: []*postingpb.ChargePolicyPosting{item}, Success: true}, nil
}

func (r *PostgresChargePolicyPostingRepository) ReadChargePolicyPosting(ctx context.Context, req *postingpb.ReadChargePolicyPostingRequest) (*postingpb.ReadChargePolicyPostingResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_policy_posting ID is required")
	}
	res, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read charge_policy_posting: %w", err)
	}
	item := &postingpb.ChargePolicyPosting{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &postingpb.ReadChargePolicyPostingResponse{Data: []*postingpb.ChargePolicyPosting{item}, Success: true}, nil
}

func (r *PostgresChargePolicyPostingRepository) UpdateChargePolicyPosting(ctx context.Context, req *postingpb.UpdateChargePolicyPostingRequest) (*postingpb.UpdateChargePolicyPostingResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_policy_posting ID is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update charge_policy_posting: %w", err)
	}
	item := &postingpb.ChargePolicyPosting{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &postingpb.UpdateChargePolicyPostingResponse{Data: []*postingpb.ChargePolicyPosting{item}, Success: true}, nil
}

func (r *PostgresChargePolicyPostingRepository) DeleteChargePolicyPosting(ctx context.Context, req *postingpb.DeleteChargePolicyPostingRequest) (*postingpb.DeleteChargePolicyPostingResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_policy_posting ID is required")
	}
	if err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete charge_policy_posting: %w", err)
	}
	return &postingpb.DeleteChargePolicyPostingResponse{Success: true}, nil
}

func (r *PostgresChargePolicyPostingRepository) ListChargePolicyPostings(ctx context.Context, req *postingpb.ListChargePolicyPostingsRequest) (*postingpb.ListChargePolicyPostingsResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), andScopeFilter(req.GetFilters(), "charge_policy_version_id", req.ChargePolicyVersionId), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &postingpb.ListChargePolicyPostingsResponse{Data: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresChargePolicyPostingRepository) GetChargePolicyPostingListPageData(ctx context.Context, req *postingpb.GetChargePolicyPostingListPageDataRequest) (*postingpb.GetChargePolicyPostingListPageDataResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &postingpb.GetChargePolicyPostingListPageDataResponse{ChargePolicyPostingList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresChargePolicyPostingRepository) GetChargePolicyPostingItemPageData(ctx context.Context, req *postingpb.GetChargePolicyPostingItemPageDataRequest) (*postingpb.GetChargePolicyPostingItemPageDataResponse, error) {
	if req == nil || req.ChargePolicyPostingId == "" {
		return nil, fmt.Errorf("charge_policy_posting ID is required")
	}
	res, err := r.dbOps.Read(ctx, r.tableName, req.ChargePolicyPostingId)
	if err != nil {
		return nil, fmt.Errorf("failed to read charge_policy_posting: %w", err)
	}
	item := &postingpb.ChargePolicyPosting{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &postingpb.GetChargePolicyPostingItemPageDataResponse{ChargePolicyPosting: item, Success: true}, nil
}

func (r *PostgresChargePolicyPostingRepository) listPage(ctx context.Context, params *interfaces.ListParams) ([]*postingpb.ChargePolicyPosting, *commonpb.PaginationResponse, error) {
	lr, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list charge_policy_posting: %w", err)
	}
	items := make([]*postingpb.ChargePolicyPosting, 0, len(lr.Data))
	for _, row := range lr.Data {
		item := &postingpb.ChargePolicyPosting{}
		if err := postgresCore.ScopedResultToProto(row, item); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	return items, lr.Pagination, nil
}
