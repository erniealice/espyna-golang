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
)

// AllocationBatch: Slice B known-cost recovery (20260927-usage-and-pass-through-charges, build-spec §6.2).
// Every by-id and list operation goes through the workspace-aware operations, entity registered
// workspaceScopeDirectRequired: a trusted, non-empty workspace predicate, no wildcard.

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.AllocationBatch, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres allocation_batch repository requires *sql.DB, got %T", conn)
		}
		// Audited + workspace-aware: diff-audit joins the caller's transaction.
		dbOps := postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db))
		return NewPostgresAllocationBatchRepository(dbOps, tableName), nil
	})
}

// PostgresAllocationBatchRepository implements allocation_batch CRUD via the generic workspace-aware operations.
type PostgresAllocationBatchRepository struct {
	allocationbatchpb.UnimplementedAllocationBatchDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresAllocationBatchRepository creates a allocation_batch repository.
func NewPostgresAllocationBatchRepository(dbOps interfaces.DatabaseOperation, tableName string) allocationbatchpb.AllocationBatchDomainServiceServer {
	if tableName == "" {
		tableName = entityid.AllocationBatch
	}
	return &PostgresAllocationBatchRepository{dbOps: dbOps, tableName: tableName}
}

func (r *PostgresAllocationBatchRepository) readByID(ctx context.Context, id string) (*allocationbatchpb.AllocationBatch, error) {
	res, err := r.dbOps.Read(ctx, r.tableName, id)
	if err != nil {
		return nil, fmt.Errorf("failed to read allocation_batch: %w", err)
	}
	item := &allocationbatchpb.AllocationBatch{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *PostgresAllocationBatchRepository) CreateAllocationBatch(ctx context.Context, req *allocationbatchpb.CreateAllocationBatchRequest) (*allocationbatchpb.CreateAllocationBatchResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("allocation_batch data is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create allocation_batch: %w", err)
	}
	item := &allocationbatchpb.AllocationBatch{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &allocationbatchpb.CreateAllocationBatchResponse{Data: []*allocationbatchpb.AllocationBatch{item}, Success: true}, nil
}

func (r *PostgresAllocationBatchRepository) ReadAllocationBatch(ctx context.Context, req *allocationbatchpb.ReadAllocationBatchRequest) (*allocationbatchpb.ReadAllocationBatchResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("allocation_batch ID is required")
	}
	item, err := r.readByID(ctx, req.Data.Id)
	if err != nil {
		return nil, err
	}
	return &allocationbatchpb.ReadAllocationBatchResponse{Data: []*allocationbatchpb.AllocationBatch{item}, Success: true}, nil
}

func (r *PostgresAllocationBatchRepository) UpdateAllocationBatch(ctx context.Context, req *allocationbatchpb.UpdateAllocationBatchRequest) (*allocationbatchpb.UpdateAllocationBatchResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("allocation_batch ID is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update allocation_batch: %w", err)
	}
	item := &allocationbatchpb.AllocationBatch{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &allocationbatchpb.UpdateAllocationBatchResponse{Data: []*allocationbatchpb.AllocationBatch{item}, Success: true}, nil
}

// DeleteAllocationBatch is a workspace-verified HARD delete of a DRAFT batch only (C6): a
// PUBLISHED or SUPERSEDED batch is an immutable financial record (its charges and the source claim
// hang off it) and is refused with ErrImmutableRow.
func (r *PostgresAllocationBatchRepository) DeleteAllocationBatch(ctx context.Context, req *allocationbatchpb.DeleteAllocationBatchRequest) (*allocationbatchpb.DeleteAllocationBatchResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("allocation_batch ID is required")
	}
	cur, err := r.readByID(ctx, req.Data.Id) // workspace-aware: foreign / missing ids fail here
	if err != nil {
		return nil, err
	}
	if cur.GetStatus() != allocationbatchpb.AllocationBatchStatus_ALLOCATION_BATCH_STATUS_DRAFT {
		return nil, postgresCore.ErrImmutableRecord("allocation_batch")
	}
	if err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete allocation_batch: %w", err)
	}
	return &allocationbatchpb.DeleteAllocationBatchResponse{Success: true}, nil
}

func (r *PostgresAllocationBatchRepository) ListAllocationBatches(ctx context.Context, req *allocationbatchpb.ListAllocationBatchesRequest) (*allocationbatchpb.ListAllocationBatchesResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, _, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &allocationbatchpb.ListAllocationBatchesResponse{Data: items, Success: true}, nil
}

func (r *PostgresAllocationBatchRepository) GetAllocationBatchListPageData(ctx context.Context, req *allocationbatchpb.GetAllocationBatchListPageDataRequest) (*allocationbatchpb.GetAllocationBatchListPageDataResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &allocationbatchpb.GetAllocationBatchListPageDataResponse{AllocationBatchList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresAllocationBatchRepository) GetAllocationBatchItemPageData(ctx context.Context, req *allocationbatchpb.GetAllocationBatchItemPageDataRequest) (*allocationbatchpb.GetAllocationBatchItemPageDataResponse, error) {
	if req == nil || req.AllocationBatchId == "" {
		return nil, fmt.Errorf("allocation_batch ID is required")
	}
	item, err := r.readByID(ctx, req.AllocationBatchId)
	if err != nil {
		return nil, err
	}
	return &allocationbatchpb.GetAllocationBatchItemPageDataResponse{AllocationBatch: item, Success: true}, nil
}

func (r *PostgresAllocationBatchRepository) listPage(ctx context.Context, params *interfaces.ListParams) ([]*allocationbatchpb.AllocationBatch, *commonpb.PaginationResponse, error) {
	lr, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list allocation_batch: %w", err)
	}
	items := make([]*allocationbatchpb.AllocationBatch, 0, len(lr.Data))
	for _, row := range lr.Data {
		item := &allocationbatchpb.AllocationBatch{}
		if err := postgresCore.ScopedResultToProto(row, item); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	return items, lr.Pagination, nil
}

// LockAllocationBatchesByComponent locks every batch of a cost source component FOR UPDATE,
// revision ascending, through the transaction executor, never r.db
// (domainports.AllocationBatchLocker). WAVE 3: publish/supersede runs under this lock.
func (r *PostgresAllocationBatchRepository) LockAllocationBatchesByComponent(ctx context.Context, componentID string) ([]*allocationbatchpb.AllocationBatch, error) {
	ids, err := postgresCore.LockScopedRowIDsForUpdate(ctx, r.dbOps, r.tableName, "cost_source_component_id", componentID, "revision ASC")
	if err != nil {
		return nil, err
	}
	out := make([]*allocationbatchpb.AllocationBatch, 0, len(ids))
	for _, id := range ids {
		item, err := r.readByID(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

// LockAllocationBatchForUpdate locks one batch row FOR UPDATE inside the ambient transaction.
func (r *PostgresAllocationBatchRepository) LockAllocationBatchForUpdate(ctx context.Context, id string) (*allocationbatchpb.AllocationBatch, error) {
	if err := postgresCore.LockScopedRowForUpdate(ctx, r.dbOps, r.tableName, id); err != nil {
		return nil, err
	}
	return r.readByID(ctx, id)
}
