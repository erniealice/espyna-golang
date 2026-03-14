
package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	invoiceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.InvoiceAttribute, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres invoice_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresInvoiceAttributeRepository(dbOps, tableName), nil
	})
}

// PostgresInvoiceAttributeRepository implements invoice attribute CRUD operations using PostgreSQL
type PostgresInvoiceAttributeRepository struct {
	invoiceattributepb.UnimplementedInvoiceAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresInvoiceAttributeRepository creates a new PostgreSQL invoice attribute repository
func NewPostgresInvoiceAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) invoiceattributepb.InvoiceAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "invoice_attribute"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresInvoiceAttributeRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateInvoiceAttribute creates a new invoice attribute using common PostgreSQL operations
func (r *PostgresInvoiceAttributeRepository) CreateInvoiceAttribute(ctx context.Context, req *invoiceattributepb.CreateInvoiceAttributeRequest) (*invoiceattributepb.CreateInvoiceAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("invoice attribute data is required")
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
		return nil, fmt.Errorf("failed to create invoice attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	invoiceAttribute := &invoiceattributepb.InvoiceAttribute{}
	if err := protojson.Unmarshal(resultJSON, invoiceAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &invoiceattributepb.CreateInvoiceAttributeResponse{
		Data: []*invoiceattributepb.InvoiceAttribute{invoiceAttribute},
	}, nil
}

// ReadInvoiceAttribute retrieves an invoice attribute using common PostgreSQL operations
func (r *PostgresInvoiceAttributeRepository) ReadInvoiceAttribute(ctx context.Context, req *invoiceattributepb.ReadInvoiceAttributeRequest) (*invoiceattributepb.ReadInvoiceAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice attribute ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read invoice attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	invoiceAttribute := &invoiceattributepb.InvoiceAttribute{}
	if err := protojson.Unmarshal(resultJSON, invoiceAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &invoiceattributepb.ReadInvoiceAttributeResponse{
		Data: []*invoiceattributepb.InvoiceAttribute{invoiceAttribute},
	}, nil
}

// UpdateInvoiceAttribute updates an invoice attribute using common PostgreSQL operations
func (r *PostgresInvoiceAttributeRepository) UpdateInvoiceAttribute(ctx context.Context, req *invoiceattributepb.UpdateInvoiceAttributeRequest) (*invoiceattributepb.UpdateInvoiceAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice attribute ID is required")
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
		return nil, fmt.Errorf("failed to update invoice attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	invoiceAttribute := &invoiceattributepb.InvoiceAttribute{}
	if err := protojson.Unmarshal(resultJSON, invoiceAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &invoiceattributepb.UpdateInvoiceAttributeResponse{
		Data: []*invoiceattributepb.InvoiceAttribute{invoiceAttribute},
	}, nil
}

// DeleteInvoiceAttribute deletes an invoice attribute using common PostgreSQL operations
func (r *PostgresInvoiceAttributeRepository) DeleteInvoiceAttribute(ctx context.Context, req *invoiceattributepb.DeleteInvoiceAttributeRequest) (*invoiceattributepb.DeleteInvoiceAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice attribute ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete invoice attribute: %w", err)
	}

	return &invoiceattributepb.DeleteInvoiceAttributeResponse{
		Success: true,
	}, nil
}

// ListInvoiceAttributes lists invoice attributes using common PostgreSQL operations
func (r *PostgresInvoiceAttributeRepository) ListInvoiceAttributes(ctx context.Context, req *invoiceattributepb.ListInvoiceAttributesRequest) (*invoiceattributepb.ListInvoiceAttributesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list invoice attributes: %w", err)
	}

	var invoiceAttributes []*invoiceattributepb.InvoiceAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		invoiceAttribute := &invoiceattributepb.InvoiceAttribute{}
		if err := protojson.Unmarshal(resultJSON, invoiceAttribute); err != nil {
			continue
		}
		invoiceAttributes = append(invoiceAttributes, invoiceAttribute)
	}

	return &invoiceattributepb.ListInvoiceAttributesResponse{
		Data: invoiceAttributes,
	}, nil
}

// GetInvoiceAttributeListPageData retrieves paginated invoice attribute list data with CTE
func (r *PostgresInvoiceAttributeRepository) GetInvoiceAttributeListPageData(ctx context.Context, req *invoiceattributepb.GetInvoiceAttributeListPageDataRequest) (*invoiceattributepb.GetInvoiceAttributeListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}
	limit, offset, page := int32(50), int32(0), int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			page = offsetPag.Page
			offset = (page - 1) * limit
		}
	}
	sortField, sortOrder := "date_created", "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	query := `WITH enriched AS (SELECT id, invoice_id, attribute_id, value, active, date_created, date_modified FROM invoice_attribute WHERE active = true AND ($1::text IS NULL OR $1::text = '' OR value ILIKE $1)), counted AS (SELECT COUNT(*) as total FROM enriched) SELECT e.*, c.total FROM enriched e, counted c ORDER BY ` + sortField + ` ` + sortOrder + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var invoiceAttributes []*invoiceattributepb.InvoiceAttribute
	var totalCount int64
	for rows.Next() {
		var id, invoiceId, attributeId, attributeValue string
		var active bool
		var dateCreated, dateModified time.Time
		var total int64
		if err := rows.Scan(&id, &invoiceId, &attributeId, &attributeValue, &active, &dateCreated, &dateModified, &total); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		totalCount = total

		rawData := map[string]interface{}{
			"id":          id,
			"invoiceId":   invoiceId,
			"attributeId": attributeId,
			"value":       attributeValue,
			"active":      active,
		}

		if !dateCreated.IsZero() {
			rawData["dateCreated"] = dateCreated.UnixMilli()
			rawData["dateCreatedString"] = dateCreated.Format(time.RFC3339)
		}
		if !dateModified.IsZero() {
			rawData["dateModified"] = dateModified.UnixMilli()
			rawData["dateModifiedString"] = dateModified.Format(time.RFC3339)
		}

		dataJSON, _ := json.Marshal(rawData)
		invoiceAttribute := &invoiceattributepb.InvoiceAttribute{}
		if err := protojson.Unmarshal(dataJSON, invoiceAttribute); err == nil {
			invoiceAttributes = append(invoiceAttributes, invoiceAttribute)
		}
	}
	totalPages := int32((totalCount + int64(limit) - 1) / int64(limit))
	return &invoiceattributepb.GetInvoiceAttributeListPageDataResponse{InvoiceAttributeList: invoiceAttributes, Pagination: &commonpb.PaginationResponse{TotalItems: int32(totalCount), CurrentPage: &page, TotalPages: &totalPages, HasNext: page < totalPages, HasPrev: page > 1}, Success: true}, nil
}

// GetInvoiceAttributeItemPageData retrieves invoice attribute item page data
func (r *PostgresInvoiceAttributeRepository) GetInvoiceAttributeItemPageData(ctx context.Context, req *invoiceattributepb.GetInvoiceAttributeItemPageDataRequest) (*invoiceattributepb.GetInvoiceAttributeItemPageDataResponse, error) {
	if req == nil || req.InvoiceAttributeId == "" {
		return nil, fmt.Errorf("invoice attribute ID required")
	}
	query := `SELECT id, invoice_id, attribute_id, value, active, date_created, date_modified FROM invoice_attribute WHERE id = $1 AND active = true`
	row := r.db.QueryRowContext(ctx, query, req.InvoiceAttributeId)
	var id, invoiceId, attributeId, attributeValue string
	var active bool
	var dateCreated, dateModified time.Time
	if err := row.Scan(&id, &invoiceId, &attributeId, &attributeValue, &active, &dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("invoice attribute not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	rawData := map[string]interface{}{
		"id":          id,
		"invoiceId":   invoiceId,
		"attributeId": attributeId,
		"value":       attributeValue,
		"active":      active,
	}

	if !dateCreated.IsZero() {
		rawData["dateCreated"] = dateCreated.UnixMilli()
		rawData["dateCreatedString"] = dateCreated.Format(time.RFC3339)
	}
	if !dateModified.IsZero() {
		rawData["dateModified"] = dateModified.UnixMilli()
		rawData["dateModifiedString"] = dateModified.Format(time.RFC3339)
	}

	dataJSON, _ := json.Marshal(rawData)
	invoiceAttribute := &invoiceattributepb.InvoiceAttribute{}
	if err := protojson.Unmarshal(dataJSON, invoiceAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	return &invoiceattributepb.GetInvoiceAttributeItemPageDataResponse{InvoiceAttribute: invoiceAttribute, Success: true}, nil
}

// NewInvoiceAttributeRepository creates a new PostgreSQL invoice_attribute repository (old-style constructor)
func NewInvoiceAttributeRepository(db *sql.DB, tableName string) invoiceattributepb.InvoiceAttributeDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresInvoiceAttributeRepository(dbOps, tableName)
}
