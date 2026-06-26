//go:build sqlserver

package document

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/template"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.DocumentTemplate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver document_template repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerDocumentTemplateRepository(dbOps, tableName), nil
	})
}

// SQLServerDocumentTemplateRepository implements document template CRUD operations using SQL Server.
//
// SQL Server dialect differences from the postgres gold standard:
//   - Placeholders: @p1, @p2, … (not $1, $2, …).
//   - Identifier quoting: [ident] (not "ident").
//   - ILIKE → LIKE (SQL Server default CI collation is case-insensitive).
//   - No RETURNING clause — OUTPUT inserted.* (handled by core operations).
type SQLServerDocumentTemplateRepository struct {
	documenttemplatepb.UnimplementedDocumentTemplateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerDocumentTemplateRepository creates a new SQL Server document template repository.
func NewSQLServerDocumentTemplateRepository(dbOps interfaces.DatabaseOperation, tableName string) documenttemplatepb.DocumentTemplateDomainServiceServer {
	if tableName == "" {
		tableName = "document_template"
	}
	return &SQLServerDocumentTemplateRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateDocumentTemplate creates a new document template record.
func (r *SQLServerDocumentTemplateRepository) CreateDocumentTemplate(ctx context.Context, req *documenttemplatepb.CreateDocumentTemplateRequest) (*documenttemplatepb.CreateDocumentTemplateResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("document template data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Convert millis timestamps to time.Time for SQL Server datetime2 columns.
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create document template: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	documentTemplate := &documenttemplatepb.DocumentTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, documentTemplate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &documenttemplatepb.CreateDocumentTemplateResponse{
		Success: true,
		Data:    []*documenttemplatepb.DocumentTemplate{documentTemplate},
	}, nil
}

// ReadDocumentTemplate retrieves a document template record by ID.
func (r *SQLServerDocumentTemplateRepository) ReadDocumentTemplate(ctx context.Context, req *documenttemplatepb.ReadDocumentTemplateRequest) (*documenttemplatepb.ReadDocumentTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("document template ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read document template: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	documentTemplate := &documenttemplatepb.DocumentTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, documentTemplate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &documenttemplatepb.ReadDocumentTemplateResponse{
		Success: true,
		Data:    []*documenttemplatepb.DocumentTemplate{documentTemplate},
	}, nil
}

// UpdateDocumentTemplate updates a document template record.
func (r *SQLServerDocumentTemplateRepository) UpdateDocumentTemplate(ctx context.Context, req *documenttemplatepb.UpdateDocumentTemplateRequest) (*documenttemplatepb.UpdateDocumentTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("document template ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Convert millis timestamps to time.Time for SQL Server datetime2 columns.
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update document template: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	documentTemplate := &documenttemplatepb.DocumentTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, documentTemplate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &documenttemplatepb.UpdateDocumentTemplateResponse{
		Success: true,
		Data:    []*documenttemplatepb.DocumentTemplate{documentTemplate},
	}, nil
}

// DeleteDocumentTemplate soft-deletes a document template record.
func (r *SQLServerDocumentTemplateRepository) DeleteDocumentTemplate(ctx context.Context, req *documenttemplatepb.DeleteDocumentTemplateRequest) (*documenttemplatepb.DeleteDocumentTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("document template ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete document template: %w", err)
	}

	return &documenttemplatepb.DeleteDocumentTemplateResponse{
		Success: true,
	}, nil
}

// ListDocumentTemplates lists document template records with optional filters.
func (r *SQLServerDocumentTemplateRepository) ListDocumentTemplates(ctx context.Context, req *documenttemplatepb.ListDocumentTemplatesRequest) (*documenttemplatepb.ListDocumentTemplatesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list document templates: %w", err)
	}

	var documentTemplates []*documenttemplatepb.DocumentTemplate
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal document_template row: %v", err)
			continue
		}

		documentTemplate := &documenttemplatepb.DocumentTemplate{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, documentTemplate); err != nil {
			log.Printf("WARN: protojson unmarshal document_template: %v", err)
			continue
		}
		documentTemplates = append(documentTemplates, documentTemplate)
	}

	return &documenttemplatepb.ListDocumentTemplatesResponse{
		Success: true,
		Data:    documentTemplates,
	}, nil
}

// NewDocumentTemplateRepository creates a new SQL Server document template repository (old-style constructor).
func NewDocumentTemplateRepository(db *sql.DB, tableName string) documenttemplatepb.DocumentTemplateDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerDocumentTemplateRepository(dbOps, tableName)
}
