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
	recoverydocumentlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document_line"
)

// RecoveryDocumentLine: Slice B known-cost recovery (20260927-usage-and-pass-through-charges, build-spec §6.2).
// Every by-id and list operation goes through the workspace-aware operations, entity registered
// workspaceScopeDirectRequired: a trusted, non-empty workspace predicate, no wildcard.

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.RecoveryDocumentLine, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres recovery_document_line repository requires *sql.DB, got %T", conn)
		}
		// Audited + workspace-aware: diff-audit joins the caller's transaction.
		dbOps := postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db))
		return NewPostgresRecoveryDocumentLineRepository(dbOps, tableName), nil
	})
}

// PostgresRecoveryDocumentLineRepository implements recovery_document_line CRUD via the generic workspace-aware operations.
type PostgresRecoveryDocumentLineRepository struct {
	recoverydocumentlinepb.UnimplementedRecoveryDocumentLineDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresRecoveryDocumentLineRepository creates a recovery_document_line repository.
func NewPostgresRecoveryDocumentLineRepository(dbOps interfaces.DatabaseOperation, tableName string) recoverydocumentlinepb.RecoveryDocumentLineDomainServiceServer {
	if tableName == "" {
		tableName = entityid.RecoveryDocumentLine
	}
	return &PostgresRecoveryDocumentLineRepository{dbOps: dbOps, tableName: tableName}
}

func (r *PostgresRecoveryDocumentLineRepository) readByID(ctx context.Context, id string) (*recoverydocumentlinepb.RecoveryDocumentLine, error) {
	res, err := r.dbOps.Read(ctx, r.tableName, id)
	if err != nil {
		return nil, fmt.Errorf("failed to read recovery_document_line: %w", err)
	}
	item := &recoverydocumentlinepb.RecoveryDocumentLine{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *PostgresRecoveryDocumentLineRepository) CreateRecoveryDocumentLine(ctx context.Context, req *recoverydocumentlinepb.CreateRecoveryDocumentLineRequest) (*recoverydocumentlinepb.CreateRecoveryDocumentLineResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("recovery_document_line data is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create recovery_document_line: %w", err)
	}
	item := &recoverydocumentlinepb.RecoveryDocumentLine{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &recoverydocumentlinepb.CreateRecoveryDocumentLineResponse{Data: []*recoverydocumentlinepb.RecoveryDocumentLine{item}, Success: true}, nil
}

func (r *PostgresRecoveryDocumentLineRepository) ReadRecoveryDocumentLine(ctx context.Context, req *recoverydocumentlinepb.ReadRecoveryDocumentLineRequest) (*recoverydocumentlinepb.ReadRecoveryDocumentLineResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("recovery_document_line ID is required")
	}
	item, err := r.readByID(ctx, req.Data.Id)
	if err != nil {
		return nil, err
	}
	return &recoverydocumentlinepb.ReadRecoveryDocumentLineResponse{Data: []*recoverydocumentlinepb.RecoveryDocumentLine{item}, Success: true}, nil
}

func (r *PostgresRecoveryDocumentLineRepository) UpdateRecoveryDocumentLine(ctx context.Context, req *recoverydocumentlinepb.UpdateRecoveryDocumentLineRequest) (*recoverydocumentlinepb.UpdateRecoveryDocumentLineResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("recovery_document_line ID is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update recovery_document_line: %w", err)
	}
	item := &recoverydocumentlinepb.RecoveryDocumentLine{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &recoverydocumentlinepb.UpdateRecoveryDocumentLineResponse{Data: []*recoverydocumentlinepb.RecoveryDocumentLine{item}, Success: true}, nil
}

// DeleteRecoveryDocumentLine is refused: recovery_document_line rows are immutable financial records, corrected by a new
// row and never removed at the port (build-spec §7 C6).
func (r *PostgresRecoveryDocumentLineRepository) DeleteRecoveryDocumentLine(ctx context.Context, req *recoverydocumentlinepb.DeleteRecoveryDocumentLineRequest) (*recoverydocumentlinepb.DeleteRecoveryDocumentLineResponse, error) {
	return nil, postgresCore.ErrImmutableRecord(r.tableName)
}

func (r *PostgresRecoveryDocumentLineRepository) ListRecoveryDocumentLines(ctx context.Context, req *recoverydocumentlinepb.ListRecoveryDocumentLinesRequest) (*recoverydocumentlinepb.ListRecoveryDocumentLinesResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, _, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &recoverydocumentlinepb.ListRecoveryDocumentLinesResponse{Data: items, Success: true}, nil
}

func (r *PostgresRecoveryDocumentLineRepository) GetRecoveryDocumentLineListPageData(ctx context.Context, req *recoverydocumentlinepb.GetRecoveryDocumentLineListPageDataRequest) (*recoverydocumentlinepb.GetRecoveryDocumentLineListPageDataResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &recoverydocumentlinepb.GetRecoveryDocumentLineListPageDataResponse{RecoveryDocumentLineList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresRecoveryDocumentLineRepository) GetRecoveryDocumentLineItemPageData(ctx context.Context, req *recoverydocumentlinepb.GetRecoveryDocumentLineItemPageDataRequest) (*recoverydocumentlinepb.GetRecoveryDocumentLineItemPageDataResponse, error) {
	if req == nil || req.RecoveryDocumentLineId == "" {
		return nil, fmt.Errorf("recovery_document_line ID is required")
	}
	item, err := r.readByID(ctx, req.RecoveryDocumentLineId)
	if err != nil {
		return nil, err
	}
	return &recoverydocumentlinepb.GetRecoveryDocumentLineItemPageDataResponse{RecoveryDocumentLine: item, Success: true}, nil
}

func (r *PostgresRecoveryDocumentLineRepository) listPage(ctx context.Context, params *interfaces.ListParams) ([]*recoverydocumentlinepb.RecoveryDocumentLine, *commonpb.PaginationResponse, error) {
	lr, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list recovery_document_line: %w", err)
	}
	items := make([]*recoverydocumentlinepb.RecoveryDocumentLine, 0, len(lr.Data))
	for _, row := range lr.Data {
		item := &recoverydocumentlinepb.RecoveryDocumentLine{}
		if err := postgresCore.ScopedResultToProto(row, item); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	return items, lr.Pagination, nil
}
