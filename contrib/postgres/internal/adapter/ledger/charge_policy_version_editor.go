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
	editorpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version_editor"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ChargePolicyVersionEditor, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres charge_policy_version_editor repository requires *sql.DB, got %T", conn)
		}
		// Audited + workspace-aware: the generic write's diff-audit joins the caller's
		// transaction and every by-id/list query carries the trusted workspace predicate
		// (entity registered workspaceScopeDirectRequired — no empty-workspace wildcard).
		dbOps := postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db))
		return NewPostgresChargePolicyVersionEditorRepository(dbOps, tableName), nil
	})
}

// PostgresChargePolicyVersionEditorRepository implements charge_policy_version_editor CRUD via the generic workspace-aware operations.
type PostgresChargePolicyVersionEditorRepository struct {
	editorpb.UnimplementedChargePolicyVersionEditorDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresChargePolicyVersionEditorRepository creates a charge_policy_version_editor repository.
func NewPostgresChargePolicyVersionEditorRepository(dbOps interfaces.DatabaseOperation, tableName string) editorpb.ChargePolicyVersionEditorDomainServiceServer {
	if tableName == "" {
		tableName = entityid.ChargePolicyVersionEditor
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresChargePolicyVersionEditorRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresChargePolicyVersionEditorRepository) CreateChargePolicyVersionEditor(ctx context.Context, req *editorpb.CreateChargePolicyVersionEditorRequest) (*editorpb.CreateChargePolicyVersionEditorResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("charge_policy_version_editor data is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create charge_policy_version_editor: %w", err)
	}
	item := &editorpb.ChargePolicyVersionEditor{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &editorpb.CreateChargePolicyVersionEditorResponse{Data: []*editorpb.ChargePolicyVersionEditor{item}, Success: true}, nil
}

func (r *PostgresChargePolicyVersionEditorRepository) ReadChargePolicyVersionEditor(ctx context.Context, req *editorpb.ReadChargePolicyVersionEditorRequest) (*editorpb.ReadChargePolicyVersionEditorResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_policy_version_editor ID is required")
	}
	res, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read charge_policy_version_editor: %w", err)
	}
	item := &editorpb.ChargePolicyVersionEditor{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &editorpb.ReadChargePolicyVersionEditorResponse{Data: []*editorpb.ChargePolicyVersionEditor{item}, Success: true}, nil
}

func (r *PostgresChargePolicyVersionEditorRepository) UpdateChargePolicyVersionEditor(ctx context.Context, req *editorpb.UpdateChargePolicyVersionEditorRequest) (*editorpb.UpdateChargePolicyVersionEditorResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_policy_version_editor ID is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update charge_policy_version_editor: %w", err)
	}
	item := &editorpb.ChargePolicyVersionEditor{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &editorpb.UpdateChargePolicyVersionEditorResponse{Data: []*editorpb.ChargePolicyVersionEditor{item}, Success: true}, nil
}

func (r *PostgresChargePolicyVersionEditorRepository) DeleteChargePolicyVersionEditor(ctx context.Context, req *editorpb.DeleteChargePolicyVersionEditorRequest) (*editorpb.DeleteChargePolicyVersionEditorResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_policy_version_editor ID is required")
	}
	if err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete charge_policy_version_editor: %w", err)
	}
	return &editorpb.DeleteChargePolicyVersionEditorResponse{Success: true}, nil
}

func (r *PostgresChargePolicyVersionEditorRepository) ListChargePolicyVersionEditors(ctx context.Context, req *editorpb.ListChargePolicyVersionEditorsRequest) (*editorpb.ListChargePolicyVersionEditorsResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), andScopeFilter(req.GetFilters(), "charge_policy_version_id", req.ChargePolicyVersionId), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &editorpb.ListChargePolicyVersionEditorsResponse{Data: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresChargePolicyVersionEditorRepository) GetChargePolicyVersionEditorListPageData(ctx context.Context, req *editorpb.GetChargePolicyVersionEditorListPageDataRequest) (*editorpb.GetChargePolicyVersionEditorListPageDataResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &editorpb.GetChargePolicyVersionEditorListPageDataResponse{ChargePolicyVersionEditorList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresChargePolicyVersionEditorRepository) GetChargePolicyVersionEditorItemPageData(ctx context.Context, req *editorpb.GetChargePolicyVersionEditorItemPageDataRequest) (*editorpb.GetChargePolicyVersionEditorItemPageDataResponse, error) {
	if req == nil || req.ChargePolicyVersionEditorId == "" {
		return nil, fmt.Errorf("charge_policy_version_editor ID is required")
	}
	res, err := r.dbOps.Read(ctx, r.tableName, req.ChargePolicyVersionEditorId)
	if err != nil {
		return nil, fmt.Errorf("failed to read charge_policy_version_editor: %w", err)
	}
	item := &editorpb.ChargePolicyVersionEditor{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &editorpb.GetChargePolicyVersionEditorItemPageDataResponse{ChargePolicyVersionEditor: item, Success: true}, nil
}

func (r *PostgresChargePolicyVersionEditorRepository) listPage(ctx context.Context, params *interfaces.ListParams) ([]*editorpb.ChargePolicyVersionEditor, *commonpb.PaginationResponse, error) {
	lr, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list charge_policy_version_editor: %w", err)
	}
	items := make([]*editorpb.ChargePolicyVersionEditor, 0, len(lr.Data))
	for _, row := range lr.Data {
		item := &editorpb.ChargePolicyVersionEditor{}
		if err := postgresCore.ScopedResultToProto(row, item); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	return items, lr.Pagination, nil
}
