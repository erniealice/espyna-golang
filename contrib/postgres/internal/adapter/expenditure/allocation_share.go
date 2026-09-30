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
	allocationbatchpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_batch"
	allocationsharepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_share"
)

// AllocationShare: Slice B known-cost recovery (20260927-usage-and-pass-through-charges, build-spec §6.2).
// Every by-id and list operation goes through the workspace-aware operations, entity registered
// workspaceScopeDirectRequired: a trusted, non-empty workspace predicate, no wildcard.

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.AllocationShare, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres allocation_share repository requires *sql.DB, got %T", conn)
		}
		// Audited + workspace-aware: diff-audit joins the caller's transaction.
		dbOps := postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db))
		return NewPostgresAllocationShareRepository(dbOps, tableName), nil
	})
}

// PostgresAllocationShareRepository implements allocation_share CRUD via the generic workspace-aware operations.
type PostgresAllocationShareRepository struct {
	allocationsharepb.UnimplementedAllocationShareDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresAllocationShareRepository creates a allocation_share repository.
func NewPostgresAllocationShareRepository(dbOps interfaces.DatabaseOperation, tableName string) allocationsharepb.AllocationShareDomainServiceServer {
	if tableName == "" {
		tableName = entityid.AllocationShare
	}
	return &PostgresAllocationShareRepository{dbOps: dbOps, tableName: tableName}
}

func (r *PostgresAllocationShareRepository) readByID(ctx context.Context, id string) (*allocationsharepb.AllocationShare, error) {
	res, err := r.dbOps.Read(ctx, r.tableName, id)
	if err != nil {
		return nil, fmt.Errorf("failed to read allocation_share: %w", err)
	}
	item := &allocationsharepb.AllocationShare{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *PostgresAllocationShareRepository) CreateAllocationShare(ctx context.Context, req *allocationsharepb.CreateAllocationShareRequest) (*allocationsharepb.CreateAllocationShareResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("allocation_share data is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create allocation_share: %w", err)
	}
	item := &allocationsharepb.AllocationShare{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &allocationsharepb.CreateAllocationShareResponse{Data: []*allocationsharepb.AllocationShare{item}, Success: true}, nil
}

func (r *PostgresAllocationShareRepository) ReadAllocationShare(ctx context.Context, req *allocationsharepb.ReadAllocationShareRequest) (*allocationsharepb.ReadAllocationShareResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("allocation_share ID is required")
	}
	item, err := r.readByID(ctx, req.Data.Id)
	if err != nil {
		return nil, err
	}
	return &allocationsharepb.ReadAllocationShareResponse{Data: []*allocationsharepb.AllocationShare{item}, Success: true}, nil
}

func (r *PostgresAllocationShareRepository) UpdateAllocationShare(ctx context.Context, req *allocationsharepb.UpdateAllocationShareRequest) (*allocationsharepb.UpdateAllocationShareResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("allocation_share ID is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update allocation_share: %w", err)
	}
	item := &allocationsharepb.AllocationShare{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &allocationsharepb.UpdateAllocationShareResponse{Data: []*allocationsharepb.AllocationShare{item}, Success: true}, nil
}

// DeleteAllocationShare is a workspace-verified HARD delete of a share of a DRAFT batch only
// (C6): once the batch is PUBLISHED or SUPERSEDED its shares are immutable (their amounts back the
// charges) and the delete is refused with ErrImmutableRow.
func (r *PostgresAllocationShareRepository) DeleteAllocationShare(ctx context.Context, req *allocationsharepb.DeleteAllocationShareRequest) (*allocationsharepb.DeleteAllocationShareResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("allocation_share ID is required")
	}
	share, err := r.readByID(ctx, req.Data.Id) // workspace-aware: foreign / missing ids fail here
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Read(ctx, entityid.AllocationBatch, share.GetAllocationBatchId())
	if err != nil {
		return nil, fmt.Errorf("failed to read allocation_batch of share: %w", err)
	}
	batch := &allocationbatchpb.AllocationBatch{}
	if err := postgresCore.ScopedResultToProto(res, batch); err != nil {
		return nil, err
	}
	if batch.GetStatus() != allocationbatchpb.AllocationBatchStatus_ALLOCATION_BATCH_STATUS_DRAFT {
		return nil, postgresCore.ErrImmutableRecord("allocation_share")
	}
	if err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete allocation_share: %w", err)
	}
	return &allocationsharepb.DeleteAllocationShareResponse{Success: true}, nil
}

func (r *PostgresAllocationShareRepository) ListAllocationShares(ctx context.Context, req *allocationsharepb.ListAllocationSharesRequest) (*allocationsharepb.ListAllocationSharesResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, _, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &allocationsharepb.ListAllocationSharesResponse{Data: items, Success: true}, nil
}

func (r *PostgresAllocationShareRepository) GetAllocationShareListPageData(ctx context.Context, req *allocationsharepb.GetAllocationShareListPageDataRequest) (*allocationsharepb.GetAllocationShareListPageDataResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &allocationsharepb.GetAllocationShareListPageDataResponse{AllocationShareList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresAllocationShareRepository) GetAllocationShareItemPageData(ctx context.Context, req *allocationsharepb.GetAllocationShareItemPageDataRequest) (*allocationsharepb.GetAllocationShareItemPageDataResponse, error) {
	if req == nil || req.AllocationShareId == "" {
		return nil, fmt.Errorf("allocation_share ID is required")
	}
	item, err := r.readByID(ctx, req.AllocationShareId)
	if err != nil {
		return nil, err
	}
	return &allocationsharepb.GetAllocationShareItemPageDataResponse{AllocationShare: item, Success: true}, nil
}

func (r *PostgresAllocationShareRepository) listPage(ctx context.Context, params *interfaces.ListParams) ([]*allocationsharepb.AllocationShare, *commonpb.PaginationResponse, error) {
	lr, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list allocation_share: %w", err)
	}
	items := make([]*allocationsharepb.AllocationShare, 0, len(lr.Data))
	for _, row := range lr.Data {
		item := &allocationsharepb.AllocationShare{}
		if err := postgresCore.ScopedResultToProto(row, item); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	return items, lr.Pagination, nil
}
