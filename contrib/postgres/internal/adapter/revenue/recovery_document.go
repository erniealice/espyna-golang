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
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
)

// RecoveryDocument: Slice B known-cost recovery (20260927-usage-and-pass-through-charges, build-spec §6.2).
// Every by-id and list operation goes through the workspace-aware operations, entity registered
// workspaceScopeDirectRequired: a trusted, non-empty workspace predicate, no wildcard.

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.RecoveryDocument, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres recovery_document repository requires *sql.DB, got %T", conn)
		}
		// Audited + workspace-aware: diff-audit joins the caller's transaction.
		dbOps := postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db))
		return NewPostgresRecoveryDocumentRepository(dbOps, tableName), nil
	})
}

// PostgresRecoveryDocumentRepository implements recovery_document CRUD via the generic workspace-aware operations.
type PostgresRecoveryDocumentRepository struct {
	recoverydocumentpb.UnimplementedRecoveryDocumentDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresRecoveryDocumentRepository creates a recovery_document repository.
func NewPostgresRecoveryDocumentRepository(dbOps interfaces.DatabaseOperation, tableName string) recoverydocumentpb.RecoveryDocumentDomainServiceServer {
	if tableName == "" {
		tableName = entityid.RecoveryDocument
	}
	return &PostgresRecoveryDocumentRepository{dbOps: dbOps, tableName: tableName}
}

func (r *PostgresRecoveryDocumentRepository) readByID(ctx context.Context, id string) (*recoverydocumentpb.RecoveryDocument, error) {
	res, err := r.dbOps.Read(ctx, r.tableName, id)
	if err != nil {
		return nil, fmt.Errorf("failed to read recovery_document: %w", err)
	}
	item := &recoverydocumentpb.RecoveryDocument{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *PostgresRecoveryDocumentRepository) CreateRecoveryDocument(ctx context.Context, req *recoverydocumentpb.CreateRecoveryDocumentRequest) (*recoverydocumentpb.CreateRecoveryDocumentResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("recovery_document data is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create recovery_document: %w", err)
	}
	item := &recoverydocumentpb.RecoveryDocument{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &recoverydocumentpb.CreateRecoveryDocumentResponse{Data: []*recoverydocumentpb.RecoveryDocument{item}, Success: true}, nil
}

func (r *PostgresRecoveryDocumentRepository) ReadRecoveryDocument(ctx context.Context, req *recoverydocumentpb.ReadRecoveryDocumentRequest) (*recoverydocumentpb.ReadRecoveryDocumentResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("recovery_document ID is required")
	}
	item, err := r.readByID(ctx, req.Data.Id)
	if err != nil {
		return nil, err
	}
	return &recoverydocumentpb.ReadRecoveryDocumentResponse{Data: []*recoverydocumentpb.RecoveryDocument{item}, Success: true}, nil
}

func (r *PostgresRecoveryDocumentRepository) UpdateRecoveryDocument(ctx context.Context, req *recoverydocumentpb.UpdateRecoveryDocumentRequest) (*recoverydocumentpb.UpdateRecoveryDocumentResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("recovery_document ID is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update recovery_document: %w", err)
	}
	item := &recoverydocumentpb.RecoveryDocument{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &recoverydocumentpb.UpdateRecoveryDocumentResponse{Data: []*recoverydocumentpb.RecoveryDocument{item}, Success: true}, nil
}

// DeleteRecoveryDocument is refused: recovery_document rows are immutable financial records, corrected by a new
// row and never removed at the port (build-spec §7 C6).
func (r *PostgresRecoveryDocumentRepository) DeleteRecoveryDocument(ctx context.Context, req *recoverydocumentpb.DeleteRecoveryDocumentRequest) (*recoverydocumentpb.DeleteRecoveryDocumentResponse, error) {
	return nil, postgresCore.ErrImmutableRecord(r.tableName)
}

func (r *PostgresRecoveryDocumentRepository) ListRecoveryDocuments(ctx context.Context, req *recoverydocumentpb.ListRecoveryDocumentsRequest) (*recoverydocumentpb.ListRecoveryDocumentsResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &recoverydocumentpb.ListRecoveryDocumentsResponse{Data: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresRecoveryDocumentRepository) GetRecoveryDocumentListPageData(ctx context.Context, req *recoverydocumentpb.GetRecoveryDocumentListPageDataRequest) (*recoverydocumentpb.GetRecoveryDocumentListPageDataResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &recoverydocumentpb.GetRecoveryDocumentListPageDataResponse{RecoveryDocumentList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresRecoveryDocumentRepository) GetRecoveryDocumentItemPageData(ctx context.Context, req *recoverydocumentpb.GetRecoveryDocumentItemPageDataRequest) (*recoverydocumentpb.GetRecoveryDocumentItemPageDataResponse, error) {
	if req == nil || req.RecoveryDocumentId == "" {
		return nil, fmt.Errorf("recovery_document ID is required")
	}
	item, err := r.readByID(ctx, req.RecoveryDocumentId)
	if err != nil {
		return nil, err
	}
	return &recoverydocumentpb.GetRecoveryDocumentItemPageDataResponse{RecoveryDocument: item, Success: true}, nil
}

func (r *PostgresRecoveryDocumentRepository) listPage(ctx context.Context, params *interfaces.ListParams) ([]*recoverydocumentpb.RecoveryDocument, *commonpb.PaginationResponse, error) {
	lr, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list recovery_document: %w", err)
	}
	items := make([]*recoverydocumentpb.RecoveryDocument, 0, len(lr.Data))
	for _, row := range lr.Data {
		item := &recoverydocumentpb.RecoveryDocument{}
		if err := postgresCore.ScopedResultToProto(row, item); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	return items, lr.Pagination, nil
}

// LockRecoveryDocumentForUpdate takes one document row FOR UPDATE inside the ambient transaction
// (domainports.RecoveryDocumentLocker). WAVE 3: void / apply run under this lock.
func (r *PostgresRecoveryDocumentRepository) LockRecoveryDocumentForUpdate(ctx context.Context, id string) (*recoverydocumentpb.RecoveryDocument, error) {
	if err := postgresCore.LockScopedRowForUpdate(ctx, r.dbOps, r.tableName, id); err != nil {
		return nil, err
	}
	return r.readByID(ctx, id)
}
