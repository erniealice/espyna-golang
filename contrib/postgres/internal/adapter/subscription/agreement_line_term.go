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
	agreementlinetermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/agreement_line_term"
)

// AgreementLineTerm: Slice B known-cost recovery (20260927-usage-and-pass-through-charges, build-spec §6.2).
// Every by-id and list operation goes through the workspace-aware operations, entity registered
// workspaceScopeDirectRequired: a trusted, non-empty workspace predicate, no wildcard.

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.AgreementLineTerm, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres agreement_line_term repository requires *sql.DB, got %T", conn)
		}
		// Audited + workspace-aware: diff-audit joins the caller's transaction.
		dbOps := postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db))
		return NewPostgresAgreementLineTermRepository(dbOps, tableName), nil
	})
}

// PostgresAgreementLineTermRepository implements agreement_line_term CRUD via the generic workspace-aware operations.
type PostgresAgreementLineTermRepository struct {
	agreementlinetermpb.UnimplementedAgreementLineTermDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresAgreementLineTermRepository creates a agreement_line_term repository.
func NewPostgresAgreementLineTermRepository(dbOps interfaces.DatabaseOperation, tableName string) agreementlinetermpb.AgreementLineTermDomainServiceServer {
	if tableName == "" {
		tableName = entityid.AgreementLineTerm
	}
	return &PostgresAgreementLineTermRepository{dbOps: dbOps, tableName: tableName}
}

func (r *PostgresAgreementLineTermRepository) readByID(ctx context.Context, id string) (*agreementlinetermpb.AgreementLineTerm, error) {
	res, err := r.dbOps.Read(ctx, r.tableName, id)
	if err != nil {
		return nil, fmt.Errorf("failed to read agreement_line_term: %w", err)
	}
	item := &agreementlinetermpb.AgreementLineTerm{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *PostgresAgreementLineTermRepository) CreateAgreementLineTerm(ctx context.Context, req *agreementlinetermpb.CreateAgreementLineTermRequest) (*agreementlinetermpb.CreateAgreementLineTermResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("agreement_line_term data is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create agreement_line_term: %w", err)
	}
	item := &agreementlinetermpb.AgreementLineTerm{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &agreementlinetermpb.CreateAgreementLineTermResponse{Data: []*agreementlinetermpb.AgreementLineTerm{item}, Success: true}, nil
}

func (r *PostgresAgreementLineTermRepository) ReadAgreementLineTerm(ctx context.Context, req *agreementlinetermpb.ReadAgreementLineTermRequest) (*agreementlinetermpb.ReadAgreementLineTermResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("agreement_line_term ID is required")
	}
	item, err := r.readByID(ctx, req.Data.Id)
	if err != nil {
		return nil, err
	}
	return &agreementlinetermpb.ReadAgreementLineTermResponse{Data: []*agreementlinetermpb.AgreementLineTerm{item}, Success: true}, nil
}

func (r *PostgresAgreementLineTermRepository) UpdateAgreementLineTerm(ctx context.Context, req *agreementlinetermpb.UpdateAgreementLineTermRequest) (*agreementlinetermpb.UpdateAgreementLineTermResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("agreement_line_term ID is required")
	}
	data, err := postgresCore.ScopedProtoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	res, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update agreement_line_term: %w", err)
	}
	item := &agreementlinetermpb.AgreementLineTerm{}
	if err := postgresCore.ScopedResultToProto(res, item); err != nil {
		return nil, err
	}
	return &agreementlinetermpb.UpdateAgreementLineTermResponse{Data: []*agreementlinetermpb.AgreementLineTerm{item}, Success: true}, nil
}

// DeleteAgreementLineTerm refuses (C6): a term is the frozen policy-version snapshot that
// billable_charge references, so it ends through effective_to, never by deletion (R4 m8).
func (r *PostgresAgreementLineTermRepository) DeleteAgreementLineTerm(_ context.Context, _ *agreementlinetermpb.DeleteAgreementLineTermRequest) (*agreementlinetermpb.DeleteAgreementLineTermResponse, error) {
	return nil, postgresCore.ErrImmutableRecord("agreement_line_term")
}

func (r *PostgresAgreementLineTermRepository) ListAgreementLineTerms(ctx context.Context, req *agreementlinetermpb.ListAgreementLineTermsRequest) (*agreementlinetermpb.ListAgreementLineTermsResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &agreementlinetermpb.ListAgreementLineTermsResponse{Data: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresAgreementLineTermRepository) GetAgreementLineTermListPageData(ctx context.Context, req *agreementlinetermpb.GetAgreementLineTermListPageDataRequest) (*agreementlinetermpb.GetAgreementLineTermListPageDataResponse, error) {
	params, err := postgresCore.ScopedListParams(req.GetSearch(), req.GetFilters(), req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	items, pagination, err := r.listPage(ctx, params)
	if err != nil {
		return nil, err
	}
	return &agreementlinetermpb.GetAgreementLineTermListPageDataResponse{AgreementLineTermList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresAgreementLineTermRepository) GetAgreementLineTermItemPageData(ctx context.Context, req *agreementlinetermpb.GetAgreementLineTermItemPageDataRequest) (*agreementlinetermpb.GetAgreementLineTermItemPageDataResponse, error) {
	if req == nil || req.AgreementLineTermId == "" {
		return nil, fmt.Errorf("agreement_line_term ID is required")
	}
	item, err := r.readByID(ctx, req.AgreementLineTermId)
	if err != nil {
		return nil, err
	}
	return &agreementlinetermpb.GetAgreementLineTermItemPageDataResponse{AgreementLineTerm: item, Success: true}, nil
}

func (r *PostgresAgreementLineTermRepository) listPage(ctx context.Context, params *interfaces.ListParams) ([]*agreementlinetermpb.AgreementLineTerm, *commonpb.PaginationResponse, error) {
	lr, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list agreement_line_term: %w", err)
	}
	items := make([]*agreementlinetermpb.AgreementLineTerm, 0, len(lr.Data))
	for _, row := range lr.Data {
		item := &agreementlinetermpb.AgreementLineTerm{}
		if err := postgresCore.ScopedResultToProto(row, item); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	return items, lr.Pagination, nil
}
