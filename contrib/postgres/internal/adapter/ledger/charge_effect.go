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
	chargeeffectpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_effect"
)

// ChargeEffect: Slice B known-cost recovery (20260927-usage-and-pass-through-charges, build-spec §6.2).
// Every by-id and list operation goes through the workspace-aware operations, entity registered
// workspaceScopeDirectRequired: a trusted, non-empty workspace predicate, no wildcard.

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ChargeEffect, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres charge_effect repository requires *sql.DB, got %T", conn)
		}
		// Audited + workspace-aware: diff-audit joins the caller's transaction.
		dbOps := postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db))
		return NewPostgresChargeEffectRepository(dbOps, tableName), nil
	})
}

// PostgresChargeEffectRepository implements charge_effect CRUD via the generic workspace-aware operations.
type PostgresChargeEffectRepository struct {
	chargeeffectpb.UnimplementedChargeEffectDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresChargeEffectRepository creates a charge_effect repository.
func NewPostgresChargeEffectRepository(dbOps interfaces.DatabaseOperation, tableName string) chargeeffectpb.ChargeEffectDomainServiceServer {
	if tableName == "" {
		tableName = entityid.ChargeEffect
	}
	return &PostgresChargeEffectRepository{dbOps: dbOps, tableName: tableName}
}

func (r *PostgresChargeEffectRepository) readByID(ctx context.Context, id string) (*chargeeffectpb.ChargeEffect, error) {
	res, err := r.dbOps.Read(ctx, r.tableName, id)
	if err != nil {
		return nil, fmt.Errorf("failed to read charge_effect: %w", err)
	}
	item := &chargeeffectpb.ChargeEffect{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *PostgresChargeEffectRepository) CreateChargeEffect(ctx context.Context, req *chargeeffectpb.CreateChargeEffectRequest) (*chargeeffectpb.CreateChargeEffectResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("charge_effect data is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create charge_effect: %w", err)
	}
	item := &chargeeffectpb.ChargeEffect{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &chargeeffectpb.CreateChargeEffectResponse{Data: []*chargeeffectpb.ChargeEffect{item}, Success: true}, nil
}

func (r *PostgresChargeEffectRepository) ReadChargeEffect(ctx context.Context, req *chargeeffectpb.ReadChargeEffectRequest) (*chargeeffectpb.ReadChargeEffectResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_effect ID is required")
	}
	item, err := r.readByID(ctx, req.Data.Id)
	if err != nil {
		return nil, err
	}
	return &chargeeffectpb.ReadChargeEffectResponse{Data: []*chargeeffectpb.ChargeEffect{item}, Success: true}, nil
}

func (r *PostgresChargeEffectRepository) UpdateChargeEffect(ctx context.Context, req *chargeeffectpb.UpdateChargeEffectRequest) (*chargeeffectpb.UpdateChargeEffectResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_effect ID is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update charge_effect: %w", err)
	}
	item := &chargeeffectpb.ChargeEffect{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &chargeeffectpb.UpdateChargeEffectResponse{Data: []*chargeeffectpb.ChargeEffect{item}, Success: true}, nil
}

// DeleteChargeEffect is refused: charge_effect rows are immutable financial records, corrected by a new
// row and never removed at the port (build-spec §7 C6).
func (r *PostgresChargeEffectRepository) DeleteChargeEffect(ctx context.Context, req *chargeeffectpb.DeleteChargeEffectRequest) (*chargeeffectpb.DeleteChargeEffectResponse, error) {
	return nil, postgresCore.ErrImmutableRecord(r.tableName)
}

func (r *PostgresChargeEffectRepository) ListChargeEffects(ctx context.Context, req *chargeeffectpb.ListChargeEffectsRequest) (*chargeeffectpb.ListChargeEffectsResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, _, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &chargeeffectpb.ListChargeEffectsResponse{Data: items, Success: true}, nil
}

func (r *PostgresChargeEffectRepository) GetChargeEffectListPageData(ctx context.Context, req *chargeeffectpb.GetChargeEffectListPageDataRequest) (*chargeeffectpb.GetChargeEffectListPageDataResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &chargeeffectpb.GetChargeEffectListPageDataResponse{ChargeEffectList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresChargeEffectRepository) GetChargeEffectItemPageData(ctx context.Context, req *chargeeffectpb.GetChargeEffectItemPageDataRequest) (*chargeeffectpb.GetChargeEffectItemPageDataResponse, error) {
	if req == nil || req.ChargeEffectId == "" {
		return nil, fmt.Errorf("charge_effect ID is required")
	}
	item, err := r.readByID(ctx, req.ChargeEffectId)
	if err != nil {
		return nil, err
	}
	return &chargeeffectpb.GetChargeEffectItemPageDataResponse{ChargeEffect: item, Success: true}, nil
}

func (r *PostgresChargeEffectRepository) listPage(ctx context.Context, params *interfaces.ListParams) ([]*chargeeffectpb.ChargeEffect, *commonpb.PaginationResponse, error) {
	lr, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list charge_effect: %w", err)
	}
	items := make([]*chargeeffectpb.ChargeEffect, 0, len(lr.Data))
	for _, row := range lr.Data {
		item := &chargeeffectpb.ChargeEffect{}
		if err := postgresCore.ScopedResultToProto(row, item); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	return items, lr.Pagination, nil
}
