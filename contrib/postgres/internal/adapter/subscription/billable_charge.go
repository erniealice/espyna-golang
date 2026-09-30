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
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
)

// BillableCharge: Slice B known-cost recovery (20260927-usage-and-pass-through-charges, build-spec §6.2).
// Every by-id and list operation goes through the workspace-aware operations, entity registered
// workspaceScopeDirectRequired: a trusted, non-empty workspace predicate, no wildcard.

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.BillableCharge, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres billable_charge repository requires *sql.DB, got %T", conn)
		}
		// Audited + workspace-aware: diff-audit joins the caller's transaction.
		dbOps := postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db))
		return NewPostgresBillableChargeRepository(dbOps, tableName), nil
	})
}

// PostgresBillableChargeRepository implements billable_charge CRUD via the generic workspace-aware operations.
type PostgresBillableChargeRepository struct {
	billablechargepb.UnimplementedBillableChargeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresBillableChargeRepository creates a billable_charge repository.
func NewPostgresBillableChargeRepository(dbOps interfaces.DatabaseOperation, tableName string) billablechargepb.BillableChargeDomainServiceServer {
	if tableName == "" {
		tableName = entityid.BillableCharge
	}
	return &PostgresBillableChargeRepository{dbOps: dbOps, tableName: tableName}
}

func (r *PostgresBillableChargeRepository) readByID(ctx context.Context, id string) (*billablechargepb.BillableCharge, error) {
	res, err := r.dbOps.Read(ctx, r.tableName, id)
	if err != nil {
		return nil, fmt.Errorf("failed to read billable_charge: %w", err)
	}
	item := &billablechargepb.BillableCharge{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *PostgresBillableChargeRepository) CreateBillableCharge(ctx context.Context, req *billablechargepb.CreateBillableChargeRequest) (*billablechargepb.CreateBillableChargeResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("billable_charge data is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create billable_charge: %w", err)
	}
	item := &billablechargepb.BillableCharge{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &billablechargepb.CreateBillableChargeResponse{Data: []*billablechargepb.BillableCharge{item}, Success: true}, nil
}

func (r *PostgresBillableChargeRepository) ReadBillableCharge(ctx context.Context, req *billablechargepb.ReadBillableChargeRequest) (*billablechargepb.ReadBillableChargeResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("billable_charge ID is required")
	}
	item, err := r.readByID(ctx, req.Data.Id)
	if err != nil {
		return nil, err
	}
	return &billablechargepb.ReadBillableChargeResponse{Data: []*billablechargepb.BillableCharge{item}, Success: true}, nil
}

func (r *PostgresBillableChargeRepository) UpdateBillableCharge(ctx context.Context, req *billablechargepb.UpdateBillableChargeRequest) (*billablechargepb.UpdateBillableChargeResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("billable_charge ID is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update billable_charge: %w", err)
	}
	item := &billablechargepb.BillableCharge{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &billablechargepb.UpdateBillableChargeResponse{Data: []*billablechargepb.BillableCharge{item}, Success: true}, nil
}

// DeleteBillableCharge is refused (C6): a billable charge is an immutable financial row; it is
// corrected by a new CORRECTION charge and never removed at the port.
func (r *PostgresBillableChargeRepository) DeleteBillableCharge(ctx context.Context, req *billablechargepb.DeleteBillableChargeRequest) (*billablechargepb.DeleteBillableChargeResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("billable_charge ID is required")
	}
	return nil, postgresCore.ErrImmutableRecord("billable_charge")
}

func (r *PostgresBillableChargeRepository) ListBillableCharges(ctx context.Context, req *billablechargepb.ListBillableChargesRequest) (*billablechargepb.ListBillableChargesResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &billablechargepb.ListBillableChargesResponse{Data: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresBillableChargeRepository) GetBillableChargeListPageData(ctx context.Context, req *billablechargepb.GetBillableChargeListPageDataRequest) (*billablechargepb.GetBillableChargeListPageDataResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &billablechargepb.GetBillableChargeListPageDataResponse{BillableChargeList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresBillableChargeRepository) GetBillableChargeItemPageData(ctx context.Context, req *billablechargepb.GetBillableChargeItemPageDataRequest) (*billablechargepb.GetBillableChargeItemPageDataResponse, error) {
	if req == nil || req.BillableChargeId == "" {
		return nil, fmt.Errorf("billable_charge ID is required")
	}
	item, err := r.readByID(ctx, req.BillableChargeId)
	if err != nil {
		return nil, err
	}
	return &billablechargepb.GetBillableChargeItemPageDataResponse{BillableCharge: item, Success: true}, nil
}

func (r *PostgresBillableChargeRepository) listPage(ctx context.Context, params *interfaces.ListParams) ([]*billablechargepb.BillableCharge, *commonpb.PaginationResponse, error) {
	lr, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list billable_charge: %w", err)
	}
	items := make([]*billablechargepb.BillableCharge, 0, len(lr.Data))
	for _, row := range lr.Data {
		item := &billablechargepb.BillableCharge{}
		if err := postgresCore.ScopedResultToProto(row, item); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	return items, lr.Pagination, nil
}

// LockBillableChargeForUpdate takes one charge row FOR UPDATE inside the ambient transaction
// (domainports.BillableChargeLocker). WAVE 3: issue/adjust run under this lock.
func (r *PostgresBillableChargeRepository) LockBillableChargeForUpdate(ctx context.Context, id string) (*billablechargepb.BillableCharge, error) {
	if err := postgresCore.LockScopedRowForUpdate(ctx, r.dbOps, r.tableName, id); err != nil {
		return nil, err
	}
	return r.readByID(ctx, id)
}
