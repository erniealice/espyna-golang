package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	accounttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account_template"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.AccountTemplate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres account_template repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresAccountTemplateRepository(dbOps, tableName), nil
	})
}

// PostgresAccountTemplateRepository implements account_template CRUD operations using PostgreSQL.
// Account templates are used to seed the chart of accounts for new workspaces.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_account_template_active ON account_template(active)
//   - CREATE INDEX idx_account_template_element ON account_template(element)
//   - CREATE INDEX idx_account_template_code ON account_template(code)
//
// TODO Phase 2: Implement GetAccountTemplateListPageData with search and pagination
// TODO Phase 2: Implement GetAccountTemplateItemPageData
type PostgresAccountTemplateRepository struct {
	accounttemplatepb.UnimplementedAccountTemplateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresAccountTemplateRepository creates a new PostgreSQL account_template repository.
func NewPostgresAccountTemplateRepository(dbOps interfaces.DatabaseOperation, tableName string) accounttemplatepb.AccountTemplateDomainServiceServer {
	if tableName == "" {
		tableName = "account_template"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresAccountTemplateRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateAccountTemplate creates a new account_template using common PostgreSQL operations.
func (r *PostgresAccountTemplateRepository) CreateAccountTemplate(ctx context.Context, req *accounttemplatepb.CreateAccountTemplateRequest) (*accounttemplatepb.CreateAccountTemplateResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("account_template data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create account_template: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	accountTemplate := &accounttemplatepb.AccountTemplate{}
	if err := protojson.Unmarshal(resultJSON, accountTemplate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &accounttemplatepb.CreateAccountTemplateResponse{
		Data: []*accounttemplatepb.AccountTemplate{accountTemplate},
	}, nil
}

// ReadAccountTemplate retrieves an account_template by ID using common PostgreSQL operations.
func (r *PostgresAccountTemplateRepository) ReadAccountTemplate(ctx context.Context, req *accounttemplatepb.ReadAccountTemplateRequest) (*accounttemplatepb.ReadAccountTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("account_template ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read account_template: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	accountTemplate := &accounttemplatepb.AccountTemplate{}
	if err := protojson.Unmarshal(resultJSON, accountTemplate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &accounttemplatepb.ReadAccountTemplateResponse{
		Data: []*accounttemplatepb.AccountTemplate{accountTemplate},
	}, nil
}

// UpdateAccountTemplate updates an account_template using common PostgreSQL operations.
func (r *PostgresAccountTemplateRepository) UpdateAccountTemplate(ctx context.Context, req *accounttemplatepb.UpdateAccountTemplateRequest) (*accounttemplatepb.UpdateAccountTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("account_template ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update account_template: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	accountTemplate := &accounttemplatepb.AccountTemplate{}
	if err := protojson.Unmarshal(resultJSON, accountTemplate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &accounttemplatepb.UpdateAccountTemplateResponse{
		Data: []*accounttemplatepb.AccountTemplate{accountTemplate},
	}, nil
}

// DeleteAccountTemplate soft-deletes an account_template using common PostgreSQL operations.
func (r *PostgresAccountTemplateRepository) DeleteAccountTemplate(ctx context.Context, req *accounttemplatepb.DeleteAccountTemplateRequest) (*accounttemplatepb.DeleteAccountTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("account_template ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete account_template: %w", err)
	}

	return &accounttemplatepb.DeleteAccountTemplateResponse{
		Success: true,
	}, nil
}

// ListAccountTemplates lists account_templates using common PostgreSQL operations.
func (r *PostgresAccountTemplateRepository) ListAccountTemplates(ctx context.Context, req *accounttemplatepb.ListAccountTemplatesRequest) (*accounttemplatepb.ListAccountTemplatesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list account_templates: %w", err)
	}

	var accountTemplates []*accounttemplatepb.AccountTemplate
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		accountTemplate := &accounttemplatepb.AccountTemplate{}
		if err := protojson.Unmarshal(resultJSON, accountTemplate); err != nil {
			continue
		}
		accountTemplates = append(accountTemplates, accountTemplate)
	}

	return &accounttemplatepb.ListAccountTemplatesResponse{
		Data: accountTemplates,
	}, nil
}

// GetAccountTemplateListPageData - TODO Phase 2: CTE with search by code/name, pagination.
func (r *PostgresAccountTemplateRepository) GetAccountTemplateListPageData(ctx context.Context, req *accounttemplatepb.GetAccountTemplateListPageDataRequest) (*accounttemplatepb.GetAccountTemplateListPageDataResponse, error) {
	// TODO Phase 2: CTE with element filter, search by code/name, sort by code ASC
	return nil, fmt.Errorf("GetAccountTemplateListPageData not yet implemented — Phase 2")
}

// GetAccountTemplateItemPageData - TODO Phase 2: implement single template view.
func (r *PostgresAccountTemplateRepository) GetAccountTemplateItemPageData(ctx context.Context, req *accounttemplatepb.GetAccountTemplateItemPageDataRequest) (*accounttemplatepb.GetAccountTemplateItemPageDataResponse, error) {
	// TODO Phase 2: fetch template with full field detail
	return nil, fmt.Errorf("GetAccountTemplateItemPageData not yet implemented — Phase 2")
}
