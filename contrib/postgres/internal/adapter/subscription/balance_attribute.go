
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
	balanceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.BalanceAttribute, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres balance_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresBalanceAttributeRepository(dbOps, tableName), nil
	})
}

// PostgresBalanceAttributeRepository implements balance attribute CRUD operations using PostgreSQL
type PostgresBalanceAttributeRepository struct {
	balanceattributepb.UnimplementedBalanceAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresBalanceAttributeRepository creates a new PostgreSQL balance attribute repository
func NewPostgresBalanceAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) balanceattributepb.BalanceAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "balance_attribute"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresBalanceAttributeRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateBalanceAttribute creates a new balance attribute using common PostgreSQL operations
func (r *PostgresBalanceAttributeRepository) CreateBalanceAttribute(ctx context.Context, req *balanceattributepb.CreateBalanceAttributeRequest) (*balanceattributepb.CreateBalanceAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("balance attribute data is required")
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
		return nil, fmt.Errorf("failed to create balance attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	balanceAttribute := &balanceattributepb.BalanceAttribute{}
	if err := protojson.Unmarshal(resultJSON, balanceAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &balanceattributepb.CreateBalanceAttributeResponse{
		Data: []*balanceattributepb.BalanceAttribute{balanceAttribute},
	}, nil
}

// ReadBalanceAttribute retrieves a balance attribute using common PostgreSQL operations
func (r *PostgresBalanceAttributeRepository) ReadBalanceAttribute(ctx context.Context, req *balanceattributepb.ReadBalanceAttributeRequest) (*balanceattributepb.ReadBalanceAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance attribute ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read balance attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	balanceAttribute := &balanceattributepb.BalanceAttribute{}
	if err := protojson.Unmarshal(resultJSON, balanceAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &balanceattributepb.ReadBalanceAttributeResponse{
		Data: []*balanceattributepb.BalanceAttribute{balanceAttribute},
	}, nil
}

// UpdateBalanceAttribute updates a balance attribute using common PostgreSQL operations
func (r *PostgresBalanceAttributeRepository) UpdateBalanceAttribute(ctx context.Context, req *balanceattributepb.UpdateBalanceAttributeRequest) (*balanceattributepb.UpdateBalanceAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance attribute ID is required")
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
		return nil, fmt.Errorf("failed to update balance attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	balanceAttribute := &balanceattributepb.BalanceAttribute{}
	if err := protojson.Unmarshal(resultJSON, balanceAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &balanceattributepb.UpdateBalanceAttributeResponse{
		Data: []*balanceattributepb.BalanceAttribute{balanceAttribute},
	}, nil
}

// DeleteBalanceAttribute deletes a balance attribute using common PostgreSQL operations
func (r *PostgresBalanceAttributeRepository) DeleteBalanceAttribute(ctx context.Context, req *balanceattributepb.DeleteBalanceAttributeRequest) (*balanceattributepb.DeleteBalanceAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance attribute ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete balance attribute: %w", err)
	}

	return &balanceattributepb.DeleteBalanceAttributeResponse{
		Success: true,
	}, nil
}

// ListBalanceAttributes lists balance attributes using common PostgreSQL operations
func (r *PostgresBalanceAttributeRepository) ListBalanceAttributes(ctx context.Context, req *balanceattributepb.ListBalanceAttributesRequest) (*balanceattributepb.ListBalanceAttributesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list balance attributes: %w", err)
	}

	var balanceAttributes []*balanceattributepb.BalanceAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		balanceAttribute := &balanceattributepb.BalanceAttribute{}
		if err := protojson.Unmarshal(resultJSON, balanceAttribute); err != nil {
			continue
		}
		balanceAttributes = append(balanceAttributes, balanceAttribute)
	}

	return &balanceattributepb.ListBalanceAttributesResponse{
		Data: balanceAttributes,
	}, nil
}

// GetBalanceAttributeListPageData retrieves paginated balance attribute list data with CTE
func (r *PostgresBalanceAttributeRepository) GetBalanceAttributeListPageData(ctx context.Context, req *balanceattributepb.GetBalanceAttributeListPageDataRequest) (*balanceattributepb.GetBalanceAttributeListPageDataResponse, error) {
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

	query := `WITH enriched AS (SELECT id, balance_id, attribute_id, value, active, date_created, date_modified FROM balance_attribute WHERE active = true AND ($1::text IS NULL OR $1::text = '' OR value ILIKE $1)), counted AS (SELECT COUNT(*) as total FROM enriched) SELECT e.*, c.total FROM enriched e, counted c ORDER BY ` + sortField + ` ` + sortOrder + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var balanceAttributes []*balanceattributepb.BalanceAttribute
	var totalCount int64
	for rows.Next() {
		var id, balanceId, attributeId, attributeValue string
		var active bool
		var dateCreated, dateModified time.Time
		var total int64
		if err := rows.Scan(&id, &balanceId, &attributeId, &attributeValue, &active, &dateCreated, &dateModified, &total); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		totalCount = total

		rawData := map[string]interface{}{
			"id":          id,
			"balanceId":   balanceId,
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
		balanceAttribute := &balanceattributepb.BalanceAttribute{}
		if err := protojson.Unmarshal(dataJSON, balanceAttribute); err == nil {
			balanceAttributes = append(balanceAttributes, balanceAttribute)
		}
	}
	totalPages := int32((totalCount + int64(limit) - 1) / int64(limit))
	return &balanceattributepb.GetBalanceAttributeListPageDataResponse{BalanceAttributeList: balanceAttributes, Pagination: &commonpb.PaginationResponse{TotalItems: int32(totalCount), CurrentPage: &page, TotalPages: &totalPages, HasNext: page < totalPages, HasPrev: page > 1}, Success: true}, nil
}

// GetBalanceAttributeItemPageData retrieves balance attribute item page data
func (r *PostgresBalanceAttributeRepository) GetBalanceAttributeItemPageData(ctx context.Context, req *balanceattributepb.GetBalanceAttributeItemPageDataRequest) (*balanceattributepb.GetBalanceAttributeItemPageDataResponse, error) {
	if req == nil || req.BalanceAttributeId == "" {
		return nil, fmt.Errorf("balance attribute ID required")
	}
	query := `SELECT id, balance_id, attribute_id, value, active, date_created, date_modified FROM balance_attribute WHERE id = $1 AND active = true`
	row := r.db.QueryRowContext(ctx, query, req.BalanceAttributeId)
	var id, balanceId, attributeId, attributeValue string
	var active bool
	var dateCreated, dateModified time.Time
	if err := row.Scan(&id, &balanceId, &attributeId, &attributeValue, &active, &dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("balance attribute not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	rawData := map[string]interface{}{
		"id":          id,
		"balanceId":   balanceId,
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
	balanceAttribute := &balanceattributepb.BalanceAttribute{}
	if err := protojson.Unmarshal(dataJSON, balanceAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	return &balanceattributepb.GetBalanceAttributeItemPageDataResponse{BalanceAttribute: balanceAttribute, Success: true}, nil
}

// NewBalanceAttributeRepository creates a new PostgreSQL balance_attribute repository (old-style constructor)
func NewBalanceAttributeRepository(db *sql.DB, tableName string) balanceattributepb.BalanceAttributeDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresBalanceAttributeRepository(dbOps, tableName)
}
