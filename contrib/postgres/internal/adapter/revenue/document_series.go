//go:build postgresql

package revenue

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
	documentseriespb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/document_series"
)

// DocumentSeries: Slice B known-cost recovery (20260927-usage-and-pass-through-charges, build-spec §6.2).
// Every by-id and list operation goes through the workspace-aware operations, entity registered
// workspaceScopeDirectRequired: a trusted, non-empty workspace predicate, no wildcard.

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.DocumentSeries, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres document_series repository requires *sql.DB, got %T", conn)
		}
		// Audited + workspace-aware: diff-audit joins the caller's transaction.
		dbOps := postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db))
		return NewPostgresDocumentSeriesRepository(dbOps, tableName), nil
	})
}

// PostgresDocumentSeriesRepository implements document_series CRUD via the generic workspace-aware operations.
type PostgresDocumentSeriesRepository struct {
	documentseriespb.UnimplementedDocumentSeriesDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresDocumentSeriesRepository creates a document_series repository.
func NewPostgresDocumentSeriesRepository(dbOps interfaces.DatabaseOperation, tableName string) documentseriespb.DocumentSeriesDomainServiceServer {
	if tableName == "" {
		tableName = entityid.DocumentSeries
	}
	return &PostgresDocumentSeriesRepository{dbOps: dbOps, tableName: tableName}
}

func (r *PostgresDocumentSeriesRepository) readByID(ctx context.Context, id string) (*documentseriespb.DocumentSeries, error) {
	res, err := r.dbOps.Read(ctx, r.tableName, id)
	if err != nil {
		return nil, fmt.Errorf("failed to read document_series: %w", err)
	}
	item := &documentseriespb.DocumentSeries{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *PostgresDocumentSeriesRepository) CreateDocumentSeries(ctx context.Context, req *documentseriespb.CreateDocumentSeriesRequest) (*documentseriespb.CreateDocumentSeriesResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("document_series data is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create document_series: %w", err)
	}
	item := &documentseriespb.DocumentSeries{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &documentseriespb.CreateDocumentSeriesResponse{Data: []*documentseriespb.DocumentSeries{item}, Success: true}, nil
}

func (r *PostgresDocumentSeriesRepository) ReadDocumentSeries(ctx context.Context, req *documentseriespb.ReadDocumentSeriesRequest) (*documentseriespb.ReadDocumentSeriesResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("document_series ID is required")
	}
	item, err := r.readByID(ctx, req.Data.Id)
	if err != nil {
		return nil, err
	}
	return &documentseriespb.ReadDocumentSeriesResponse{Data: []*documentseriespb.DocumentSeries{item}, Success: true}, nil
}

func (r *PostgresDocumentSeriesRepository) UpdateDocumentSeries(ctx context.Context, req *documentseriespb.UpdateDocumentSeriesRequest) (*documentseriespb.UpdateDocumentSeriesResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("document_series ID is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update document_series: %w", err)
	}
	item := &documentseriespb.DocumentSeries{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &documentseriespb.UpdateDocumentSeriesResponse{Data: []*documentseriespb.DocumentSeries{item}, Success: true}, nil
}

// DeleteDocumentSeries is a workspace-verified HARD delete; the use cases decide when it is allowed.
func (r *PostgresDocumentSeriesRepository) DeleteDocumentSeries(ctx context.Context, req *documentseriespb.DeleteDocumentSeriesRequest) (*documentseriespb.DeleteDocumentSeriesResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("document_series ID is required")
	}
	if err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete document_series: %w", err)
	}
	return &documentseriespb.DeleteDocumentSeriesResponse{Success: true}, nil
}

func (r *PostgresDocumentSeriesRepository) ListDocumentSeries(ctx context.Context, req *documentseriespb.ListDocumentSeriesRequest) (*documentseriespb.ListDocumentSeriesResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, _, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &documentseriespb.ListDocumentSeriesResponse{Data: items, Success: true}, nil
}

func (r *PostgresDocumentSeriesRepository) GetDocumentSeriesListPageData(ctx context.Context, req *documentseriespb.GetDocumentSeriesListPageDataRequest) (*documentseriespb.GetDocumentSeriesListPageDataResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &documentseriespb.GetDocumentSeriesListPageDataResponse{DocumentSeriesList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresDocumentSeriesRepository) GetDocumentSeriesItemPageData(ctx context.Context, req *documentseriespb.GetDocumentSeriesItemPageDataRequest) (*documentseriespb.GetDocumentSeriesItemPageDataResponse, error) {
	if req == nil || req.DocumentSeriesId == "" {
		return nil, fmt.Errorf("document_series ID is required")
	}
	item, err := r.readByID(ctx, req.DocumentSeriesId)
	if err != nil {
		return nil, err
	}
	return &documentseriespb.GetDocumentSeriesItemPageDataResponse{DocumentSeries: item, Success: true}, nil
}

func (r *PostgresDocumentSeriesRepository) listPage(ctx context.Context, params *interfaces.ListParams) ([]*documentseriespb.DocumentSeries, *commonpb.PaginationResponse, error) {
	lr, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list document_series: %w", err)
	}
	items := make([]*documentseriespb.DocumentSeries, 0, len(lr.Data))
	for _, row := range lr.Data {
		item := &documentseriespb.DocumentSeries{}
		if err := postgresCore.ScopedResultToProto(row, item); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	return items, lr.Pagination, nil
}

// LockDocumentSeriesForUpdate takes the series row FOR UPDATE inside the ambient transaction
// (domainports.DocumentSeriesLocker). WAVE 3: number allocation reads and bumps next_number
// only while holding this lock.
func (r *PostgresDocumentSeriesRepository) LockDocumentSeriesForUpdate(ctx context.Context, id string) (*documentseriespb.DocumentSeries, error) {
	if err := postgresCore.LockScopedRowForUpdate(ctx, r.dbOps, r.tableName, id); err != nil {
		return nil, err
	}
	return r.readByID(ctx, id)
}
