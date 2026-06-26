//go:build sqlserver

package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	accounttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account_template"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.AccountTemplate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver account_template repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerAccountTemplateRepository(dbOps, tableName), nil
	})
}

// SQLServerAccountTemplateRepository implements account_template CRUD using SQL Server.
type SQLServerAccountTemplateRepository struct {
	accounttemplatepb.UnimplementedAccountTemplateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerAccountTemplateRepository creates a new SQL Server account_template repository.
func NewSQLServerAccountTemplateRepository(dbOps interfaces.DatabaseOperation, tableName string) accounttemplatepb.AccountTemplateDomainServiceServer {
	if tableName == "" {
		tableName = "account_template"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &SQLServerAccountTemplateRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func (r *SQLServerAccountTemplateRepository) CreateAccountTemplate(ctx context.Context, req *accounttemplatepb.CreateAccountTemplateRequest) (*accounttemplatepb.CreateAccountTemplateResponse, error) {
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
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, accountTemplate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &accounttemplatepb.CreateAccountTemplateResponse{Data: []*accounttemplatepb.AccountTemplate{accountTemplate}}, nil
}

func (r *SQLServerAccountTemplateRepository) ReadAccountTemplate(ctx context.Context, req *accounttemplatepb.ReadAccountTemplateRequest) (*accounttemplatepb.ReadAccountTemplateResponse, error) {
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
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, accountTemplate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &accounttemplatepb.ReadAccountTemplateResponse{Data: []*accounttemplatepb.AccountTemplate{accountTemplate}}, nil
}

func (r *SQLServerAccountTemplateRepository) UpdateAccountTemplate(ctx context.Context, req *accounttemplatepb.UpdateAccountTemplateRequest) (*accounttemplatepb.UpdateAccountTemplateResponse, error) {
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
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, accountTemplate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &accounttemplatepb.UpdateAccountTemplateResponse{Data: []*accounttemplatepb.AccountTemplate{accountTemplate}}, nil
}

func (r *SQLServerAccountTemplateRepository) DeleteAccountTemplate(ctx context.Context, req *accounttemplatepb.DeleteAccountTemplateRequest) (*accounttemplatepb.DeleteAccountTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("account_template ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete account_template: %w", err)
	}
	return &accounttemplatepb.DeleteAccountTemplateResponse{Success: true}, nil
}

func (r *SQLServerAccountTemplateRepository) ListAccountTemplates(ctx context.Context, req *accounttemplatepb.ListAccountTemplatesRequest) (*accounttemplatepb.ListAccountTemplatesResponse, error) {
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
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, accountTemplate); err != nil {
			continue
		}
		accountTemplates = append(accountTemplates, accountTemplate)
	}
	return &accounttemplatepb.ListAccountTemplatesResponse{Data: accountTemplates}, nil
}

func (r *SQLServerAccountTemplateRepository) GetAccountTemplateListPageData(ctx context.Context, req *accounttemplatepb.GetAccountTemplateListPageDataRequest) (*accounttemplatepb.GetAccountTemplateListPageDataResponse, error) {
	return nil, fmt.Errorf("GetAccountTemplateListPageData not yet implemented — Phase 2")
}

func (r *SQLServerAccountTemplateRepository) GetAccountTemplateItemPageData(ctx context.Context, req *accounttemplatepb.GetAccountTemplateItemPageDataRequest) (*accounttemplatepb.GetAccountTemplateItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetAccountTemplateItemPageData not yet implemented — Phase 2")
}
