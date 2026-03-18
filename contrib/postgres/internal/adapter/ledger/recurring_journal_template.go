
package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	recurringpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/recurring_journal_template"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.RecurringJournalTemplate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres recurring_journal_template repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresRecurringJournalTemplateRepository(dbOps, tableName), nil
	})
}

// PostgresRecurringJournalTemplateRepository implements recurring_journal_template CRUD operations using PostgreSQL.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_recurring_journal_template_active ON recurring_journal_template(active)
//   - CREATE INDEX idx_recurring_journal_template_frequency ON recurring_journal_template(frequency)
//   - CREATE INDEX idx_recurring_journal_template_next_run ON recurring_journal_template(next_run_date)
//
// TODO Phase 2: Implement GetRecurringJournalTemplateListPageData with frequency filter and pagination
// TODO Phase 2: Implement GetRecurringJournalTemplateItemPageData
// TODO Phase 2: Implement GenerateFromTemplate — creates a JournalEntry from this template's lines
type PostgresRecurringJournalTemplateRepository struct {
	recurringpb.UnimplementedRecurringJournalTemplateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresRecurringJournalTemplateRepository creates a new PostgreSQL recurring_journal_template repository.
func NewPostgresRecurringJournalTemplateRepository(dbOps interfaces.DatabaseOperation, tableName string) recurringpb.RecurringJournalTemplateDomainServiceServer {
	if tableName == "" {
		tableName = "recurring_journal_template"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresRecurringJournalTemplateRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateRecurringJournalTemplate creates a new recurring_journal_template using common PostgreSQL operations.
func (r *PostgresRecurringJournalTemplateRepository) CreateRecurringJournalTemplate(ctx context.Context, req *recurringpb.CreateRecurringJournalTemplateRequest) (*recurringpb.CreateRecurringJournalTemplateResponse, error) {
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
	if err := protojson.Unmarshal(resultJSON, template); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &recurringpb.CreateRecurringJournalTemplateResponse{
		Data: []*recurringpb.RecurringJournalTemplate{template},
	}, nil
}

// ReadRecurringJournalTemplate retrieves a recurring_journal_template by ID using common PostgreSQL operations.
func (r *PostgresRecurringJournalTemplateRepository) ReadRecurringJournalTemplate(ctx context.Context, req *recurringpb.ReadRecurringJournalTemplateRequest) (*recurringpb.ReadRecurringJournalTemplateResponse, error) {
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
	if err := protojson.Unmarshal(resultJSON, template); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &recurringpb.ReadRecurringJournalTemplateResponse{
		Data: []*recurringpb.RecurringJournalTemplate{template},
	}, nil
}

// UpdateRecurringJournalTemplate updates a recurring_journal_template using common PostgreSQL operations.
func (r *PostgresRecurringJournalTemplateRepository) UpdateRecurringJournalTemplate(ctx context.Context, req *recurringpb.UpdateRecurringJournalTemplateRequest) (*recurringpb.UpdateRecurringJournalTemplateResponse, error) {
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
	if err := protojson.Unmarshal(resultJSON, template); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &recurringpb.UpdateRecurringJournalTemplateResponse{
		Data: []*recurringpb.RecurringJournalTemplate{template},
	}, nil
}

// DeleteRecurringJournalTemplate soft-deletes a recurring_journal_template using common PostgreSQL operations.
func (r *PostgresRecurringJournalTemplateRepository) DeleteRecurringJournalTemplate(ctx context.Context, req *recurringpb.DeleteRecurringJournalTemplateRequest) (*recurringpb.DeleteRecurringJournalTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("recurring_journal_template ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete recurring_journal_template: %w", err)
	}

	return &recurringpb.DeleteRecurringJournalTemplateResponse{
		Success: true,
	}, nil
}

// ListRecurringJournalTemplates lists recurring_journal_templates using common PostgreSQL operations.
func (r *PostgresRecurringJournalTemplateRepository) ListRecurringJournalTemplates(ctx context.Context, req *recurringpb.ListRecurringJournalTemplatesRequest) (*recurringpb.ListRecurringJournalTemplatesResponse, error) {
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
		if err := protojson.Unmarshal(resultJSON, template); err != nil {
			continue
		}
		templates = append(templates, template)
	}

	return &recurringpb.ListRecurringJournalTemplatesResponse{
		Data: templates,
	}, nil
}

// GetRecurringJournalTemplateListPageData - TODO Phase 2: CTE with frequency filter, next_run_date, pagination.
func (r *PostgresRecurringJournalTemplateRepository) GetRecurringJournalTemplateListPageData(ctx context.Context, req *recurringpb.GetRecurringJournalTemplateListPageDataRequest) (*recurringpb.GetRecurringJournalTemplateListPageDataResponse, error) {
	// TODO Phase 2: CTE with search by name, frequency filter, sort by next_run_date ASC
	return nil, fmt.Errorf("GetRecurringJournalTemplateListPageData not yet implemented — Phase 2")
}

// GetRecurringJournalTemplateItemPageData - TODO Phase 2: implement with template line items.
func (r *PostgresRecurringJournalTemplateRepository) GetRecurringJournalTemplateItemPageData(ctx context.Context, req *recurringpb.GetRecurringJournalTemplateItemPageDataRequest) (*recurringpb.GetRecurringJournalTemplateItemPageDataResponse, error) {
	// TODO Phase 2: fetch template + line items + account names
	return nil, fmt.Errorf("GetRecurringJournalTemplateItemPageData not yet implemented — Phase 2")
}

// GenerateFromTemplate - TODO Phase 2: create JournalEntry from template lines for the given period.
func (r *PostgresRecurringJournalTemplateRepository) GenerateFromTemplate(ctx context.Context, req *recurringpb.GenerateFromTemplateRequest) (*recurringpb.GenerateFromTemplateResponse, error) {
	// TODO Phase 2: read template lines, create JournalEntry + JournalLines, update next_run_date
	return nil, fmt.Errorf("GenerateFromTemplate not yet implemented — Phase 2")
}
