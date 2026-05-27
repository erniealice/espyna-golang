//go:build sqlserver

package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	recurringpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/recurring_journal_template"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.RecurringJournalTemplate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver recurring_journal_template repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerRecurringJournalTemplateRepository(dbOps, tableName), nil
	})
}

// SQLServerRecurringJournalTemplateRepository implements recurring_journal_template CRUD using SQL Server.
type SQLServerRecurringJournalTemplateRepository struct {
	recurringpb.UnimplementedRecurringJournalTemplateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerRecurringJournalTemplateRepository creates a new SQL Server recurring_journal_template repository.
func NewSQLServerRecurringJournalTemplateRepository(dbOps interfaces.DatabaseOperation, tableName string) recurringpb.RecurringJournalTemplateDomainServiceServer {
	if tableName == "" {
		tableName = "recurring_journal_template"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &SQLServerRecurringJournalTemplateRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func (r *SQLServerRecurringJournalTemplateRepository) CreateRecurringJournalTemplate(ctx context.Context, req *recurringpb.CreateRecurringJournalTemplateRequest) (*recurringpb.CreateRecurringJournalTemplateResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("recurring_journal_template data is required")
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
		return nil, fmt.Errorf("failed to create recurring_journal_template: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	template := &recurringpb.RecurringJournalTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, template); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &recurringpb.CreateRecurringJournalTemplateResponse{Data: []*recurringpb.RecurringJournalTemplate{template}}, nil
}

func (r *SQLServerRecurringJournalTemplateRepository) ReadRecurringJournalTemplate(ctx context.Context, req *recurringpb.ReadRecurringJournalTemplateRequest) (*recurringpb.ReadRecurringJournalTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("recurring_journal_template ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read recurring_journal_template: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	template := &recurringpb.RecurringJournalTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, template); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &recurringpb.ReadRecurringJournalTemplateResponse{Data: []*recurringpb.RecurringJournalTemplate{template}}, nil
}

func (r *SQLServerRecurringJournalTemplateRepository) UpdateRecurringJournalTemplate(ctx context.Context, req *recurringpb.UpdateRecurringJournalTemplateRequest) (*recurringpb.UpdateRecurringJournalTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("recurring_journal_template ID is required")
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
		return nil, fmt.Errorf("failed to update recurring_journal_template: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	template := &recurringpb.RecurringJournalTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, template); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &recurringpb.UpdateRecurringJournalTemplateResponse{Data: []*recurringpb.RecurringJournalTemplate{template}}, nil
}

func (r *SQLServerRecurringJournalTemplateRepository) DeleteRecurringJournalTemplate(ctx context.Context, req *recurringpb.DeleteRecurringJournalTemplateRequest) (*recurringpb.DeleteRecurringJournalTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("recurring_journal_template ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete recurring_journal_template: %w", err)
	}
	return &recurringpb.DeleteRecurringJournalTemplateResponse{Success: true}, nil
}

func (r *SQLServerRecurringJournalTemplateRepository) ListRecurringJournalTemplates(ctx context.Context, req *recurringpb.ListRecurringJournalTemplatesRequest) (*recurringpb.ListRecurringJournalTemplatesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list recurring_journal_templates: %w", err)
	}
	var templates []*recurringpb.RecurringJournalTemplate
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		template := &recurringpb.RecurringJournalTemplate{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, template); err != nil {
			continue
		}
		templates = append(templates, template)
	}
	return &recurringpb.ListRecurringJournalTemplatesResponse{Data: templates}, nil
}

func (r *SQLServerRecurringJournalTemplateRepository) GetRecurringJournalTemplateListPageData(ctx context.Context, req *recurringpb.GetRecurringJournalTemplateListPageDataRequest) (*recurringpb.GetRecurringJournalTemplateListPageDataResponse, error) {
	return nil, fmt.Errorf("GetRecurringJournalTemplateListPageData not yet implemented — Phase 2")
}

func (r *SQLServerRecurringJournalTemplateRepository) GetRecurringJournalTemplateItemPageData(ctx context.Context, req *recurringpb.GetRecurringJournalTemplateItemPageDataRequest) (*recurringpb.GetRecurringJournalTemplateItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetRecurringJournalTemplateItemPageData not yet implemented — Phase 2")
}

func (r *SQLServerRecurringJournalTemplateRepository) GenerateFromTemplate(ctx context.Context, req *recurringpb.GenerateFromTemplateRequest) (*recurringpb.GenerateFromTemplateResponse, error) {
	return nil, fmt.Errorf("GenerateFromTemplate not yet implemented — Phase 2")
}
