//go:build mysql

// Dialect translation from postgres gold standard:
//   - $1,$2,... → ? (MySQL positional placeholders)
//   - "ident"   → `ident` (backtick quoting)
//   - ILIKE     → LIKE (ci collation)
//   - active = true → active = 1
//   - LIMIT $2 OFFSET $3 → LIMIT ? OFFSET ?
//   - COUNT(*) OVER () stays (MySQL 8.0+ window function)
package revenue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	revenueattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.RevenueAttribute, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql revenue_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLRevenueAttributeRepository(dbOps, tableName), nil
	})
}

// MySQLRevenueAttributeRepository implements revenue_attribute CRUD using MySQL 8.0+.
type MySQLRevenueAttributeRepository struct {
	revenueattributepb.UnimplementedRevenueAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLRevenueAttributeRepository creates a new MySQL revenue attribute repository.
func NewMySQLRevenueAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) revenueattributepb.RevenueAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "revenue_attribute"
	}
	return &MySQLRevenueAttributeRepository{
		dbOps:     dbOps,
		db:        getDB(dbOps),
		tableName: tableName,
	}
}

// CreateRevenueAttribute creates a new revenue attribute.
func (r *MySQLRevenueAttributeRepository) CreateRevenueAttribute(ctx context.Context, req *revenueattributepb.CreateRevenueAttributeRequest) (*revenueattributepb.CreateRevenueAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("revenue attribute data is required")
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
		return nil, fmt.Errorf("failed to create revenue attribute: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	attr := &revenueattributepb.RevenueAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenueattributepb.CreateRevenueAttributeResponse{
		Data: []*revenueattributepb.RevenueAttribute{attr},
	}, nil
}

// ReadRevenueAttribute retrieves a revenue attribute by ID.
func (r *MySQLRevenueAttributeRepository) ReadRevenueAttribute(ctx context.Context, req *revenueattributepb.ReadRevenueAttributeRequest) (*revenueattributepb.ReadRevenueAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue attribute ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read revenue attribute: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	attr := &revenueattributepb.RevenueAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenueattributepb.ReadRevenueAttributeResponse{
		Data: []*revenueattributepb.RevenueAttribute{attr},
	}, nil
}

// UpdateRevenueAttribute updates a revenue attribute.
func (r *MySQLRevenueAttributeRepository) UpdateRevenueAttribute(ctx context.Context, req *revenueattributepb.UpdateRevenueAttributeRequest) (*revenueattributepb.UpdateRevenueAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue attribute ID is required")
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
		return nil, fmt.Errorf("failed to update revenue attribute: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	attr := &revenueattributepb.RevenueAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenueattributepb.UpdateRevenueAttributeResponse{
		Data: []*revenueattributepb.RevenueAttribute{attr},
	}, nil
}

// DeleteRevenueAttribute soft-deletes a revenue attribute.
func (r *MySQLRevenueAttributeRepository) DeleteRevenueAttribute(ctx context.Context, req *revenueattributepb.DeleteRevenueAttributeRequest) (*revenueattributepb.DeleteRevenueAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue attribute ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete revenue attribute: %w", err)
	}

	return &revenueattributepb.DeleteRevenueAttributeResponse{Success: true}, nil
}

// ListRevenueAttributes lists revenue attributes with optional filters.
func (r *MySQLRevenueAttributeRepository) ListRevenueAttributes(ctx context.Context, req *revenueattributepb.ListRevenueAttributesRequest) (*revenueattributepb.ListRevenueAttributesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list revenue attributes: %w", err)
	}

	var attrs []*revenueattributepb.RevenueAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal revenue_attribute row: %v", err)
			continue
		}
		attr := &revenueattributepb.RevenueAttribute{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attr); err != nil {
			log.Printf("WARN: protojson unmarshal revenue_attribute: %v", err)
			continue
		}
		attrs = append(attrs, attr)
	}

	return &revenueattributepb.ListRevenueAttributesResponse{Data: attrs}, nil
}

// GetRevenueAttributeListPageData retrieves revenue attributes with pagination,
// sorting, and search using a counted CTE.
//
// Dialect: $1/$2/$3 → ?; ILIKE → LIKE; active = true → active = 1;
// COUNT(*) OVER() stays (MySQL 8.0+).
func (r *MySQLRevenueAttributeRepository) GetRevenueAttributeListPageData(
	ctx context.Context,
	req *revenueattributepb.GetRevenueAttributeListPageDataRequest,
) (*revenueattributepb.GetRevenueAttributeListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get revenue attribute list page data request is required")
	}

	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	sortField := "ra.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// Dialect: $1::text IS NULL OR ... ILIKE $1 →
	// (? = '' OR ra.value LIKE ? OR rv.name LIKE ?)
	// Pass searchPattern three times: empty-check + two LIKE args.
	query := `
		WITH enriched AS (
			SELECT
				ra.id,
				ra.revenue_id,
				ra.attribute_id,
				ra.value,
				ra.date_created,
				ra.date_modified,
				ra.active,
				COALESCE(rv.name, '') as revenue_name
			FROM revenue_attribute ra
			LEFT JOIN revenue rv ON ra.revenue_id = rv.id AND rv.active = 1
			WHERE ra.active = 1
			  AND (? = '' OR ra.value LIKE ? OR rv.name LIKE ?)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT ? OFFSET ?
	`

	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	rows, err := r.db.QueryContext(ctx, query, searchPattern, searchPattern, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue attribute list page data: %w", err)
	}
	defer rows.Close()

	var attrs []*revenueattributepb.RevenueAttribute
	var totalCount int64

	for rows.Next() {
		var (
			id           string
			revenueID    string
			attributeID  string
			value        string
			dateCreated  time.Time
			dateModified time.Time
			active       bool
			revenueName  string
			total        int64
		)

		if err := rows.Scan(
			&id,
			&revenueID,
			&attributeID,
			&value,
			&dateCreated,
			&dateModified,
			&active,
			&revenueName,
			&total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan revenue attribute row: %w", err)
		}

		totalCount = total

		attr := &revenueattributepb.RevenueAttribute{
			Id:          id,
			RevenueId:   revenueID,
			AttributeId: attributeID,
			Value:       value,
			Active:      active,
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			attr.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			attr.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			attr.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			attr.DateModifiedString = &dmStr
		}

		attrs = append(attrs, attr)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating revenue attribute rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &revenueattributepb.GetRevenueAttributeListPageDataResponse{
		RevenueAttributeList: attrs,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
		Success: true,
	}, nil
}

// GetRevenueAttributeItemPageData retrieves a single revenue attribute with enriched data.
//
// Dialect: $1 → ?; active = true → active = 1.
func (r *MySQLRevenueAttributeRepository) GetRevenueAttributeItemPageData(
	ctx context.Context,
	req *revenueattributepb.GetRevenueAttributeItemPageDataRequest,
) (*revenueattributepb.GetRevenueAttributeItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get revenue attribute item page data request is required")
	}
	if req.RevenueAttributeId == "" {
		return nil, fmt.Errorf("revenue attribute ID is required")
	}

	query := `
		WITH enriched AS (
			SELECT
				ra.id,
				ra.revenue_id,
				ra.attribute_id,
				ra.value,
				ra.date_created,
				ra.date_modified,
				ra.active,
				COALESCE(rv.name, '') as revenue_name
			FROM revenue_attribute ra
			LEFT JOIN revenue rv ON ra.revenue_id = rv.id AND rv.active = 1
			WHERE ra.id = ? AND ra.active = 1
		)
		SELECT * FROM enriched LIMIT 1
	`

	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	row := r.db.QueryRowContext(ctx, query, req.RevenueAttributeId)

	var (
		id           string
		revenueID    string
		attributeID  string
		value        string
		dateCreated  time.Time
		dateModified time.Time
		active       bool
		revenueName  string
	)

	err := row.Scan(
		&id,
		&revenueID,
		&attributeID,
		&value,
		&dateCreated,
		&dateModified,
		&active,
		&revenueName,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("revenue attribute with ID '%s' not found", req.RevenueAttributeId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue attribute item page data: %w", err)
	}

	attr := &revenueattributepb.RevenueAttribute{
		Id:          id,
		RevenueId:   revenueID,
		AttributeId: attributeID,
		Value:       value,
		Active:      active,
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		attr.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		attr.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		attr.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		attr.DateModifiedString = &dmStr
	}

	return &revenueattributepb.GetRevenueAttributeItemPageDataResponse{
		RevenueAttribute: attr,
		Success:          true,
	}, nil
}

// NewRevenueAttributeRepository creates a new MySQL revenue attribute repository (old-style constructor).
func NewRevenueAttributeRepository(db *sql.DB, tableName string) revenueattributepb.RevenueAttributeDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLRevenueAttributeRepository(dbOps, tableName)
}
