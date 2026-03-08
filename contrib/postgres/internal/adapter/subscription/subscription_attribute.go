
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
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	subscriptionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "subscription_attribute", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres subscription_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresSubscriptionAttributeRepository(dbOps, tableName), nil
	})
}

// PostgresSubscriptionAttributeRepository implements subscription attribute CRUD operations using PostgreSQL
type PostgresSubscriptionAttributeRepository struct {
	subscriptionattributepb.UnimplementedSubscriptionAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresSubscriptionAttributeRepository creates a new PostgreSQL subscription attribute repository
func NewPostgresSubscriptionAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) subscriptionattributepb.SubscriptionAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "subscription_attribute"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresSubscriptionAttributeRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateSubscriptionAttribute creates a new subscription attribute using common PostgreSQL operations
func (r *PostgresSubscriptionAttributeRepository) CreateSubscriptionAttribute(ctx context.Context, req *subscriptionattributepb.CreateSubscriptionAttributeRequest) (*subscriptionattributepb.CreateSubscriptionAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("subscription attribute data is required")
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
		return nil, fmt.Errorf("failed to create subscription attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	subscriptionAttribute := &subscriptionattributepb.SubscriptionAttribute{}
	if err := protojson.Unmarshal(resultJSON, subscriptionAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &subscriptionattributepb.CreateSubscriptionAttributeResponse{
		Data: []*subscriptionattributepb.SubscriptionAttribute{subscriptionAttribute},
	}, nil
}

// ReadSubscriptionAttribute retrieves a subscription attribute using common PostgreSQL operations
func (r *PostgresSubscriptionAttributeRepository) ReadSubscriptionAttribute(ctx context.Context, req *subscriptionattributepb.ReadSubscriptionAttributeRequest) (*subscriptionattributepb.ReadSubscriptionAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription attribute ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	subscriptionAttribute := &subscriptionattributepb.SubscriptionAttribute{}
	if err := protojson.Unmarshal(resultJSON, subscriptionAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &subscriptionattributepb.ReadSubscriptionAttributeResponse{
		Data: []*subscriptionattributepb.SubscriptionAttribute{subscriptionAttribute},
	}, nil
}

// UpdateSubscriptionAttribute updates a subscription attribute using common PostgreSQL operations
func (r *PostgresSubscriptionAttributeRepository) UpdateSubscriptionAttribute(ctx context.Context, req *subscriptionattributepb.UpdateSubscriptionAttributeRequest) (*subscriptionattributepb.UpdateSubscriptionAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription attribute ID is required")
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
		return nil, fmt.Errorf("failed to update subscription attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	subscriptionAttribute := &subscriptionattributepb.SubscriptionAttribute{}
	if err := protojson.Unmarshal(resultJSON, subscriptionAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &subscriptionattributepb.UpdateSubscriptionAttributeResponse{
		Data: []*subscriptionattributepb.SubscriptionAttribute{subscriptionAttribute},
	}, nil
}

// DeleteSubscriptionAttribute deletes a subscription attribute using common PostgreSQL operations
func (r *PostgresSubscriptionAttributeRepository) DeleteSubscriptionAttribute(ctx context.Context, req *subscriptionattributepb.DeleteSubscriptionAttributeRequest) (*subscriptionattributepb.DeleteSubscriptionAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription attribute ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete subscription attribute: %w", err)
	}

	return &subscriptionattributepb.DeleteSubscriptionAttributeResponse{
		Success: true,
	}, nil
}

// ListSubscriptionAttributes lists subscription attributes using common PostgreSQL operations
func (r *PostgresSubscriptionAttributeRepository) ListSubscriptionAttributes(ctx context.Context, req *subscriptionattributepb.ListSubscriptionAttributesRequest) (*subscriptionattributepb.ListSubscriptionAttributesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscription attributes: %w", err)
	}

	var subscriptionAttributes []*subscriptionattributepb.SubscriptionAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		subscriptionAttribute := &subscriptionattributepb.SubscriptionAttribute{}
		if err := protojson.Unmarshal(resultJSON, subscriptionAttribute); err != nil {
			continue
		}
		subscriptionAttributes = append(subscriptionAttributes, subscriptionAttribute)
	}

	return &subscriptionattributepb.ListSubscriptionAttributesResponse{
		Data: subscriptionAttributes,
	}, nil
}

// GetSubscriptionAttributeListPageData retrieves paginated subscription attribute list data with CTE
func (r *PostgresSubscriptionAttributeRepository) GetSubscriptionAttributeListPageData(ctx context.Context, req *subscriptionattributepb.GetSubscriptionAttributeListPageDataRequest) (*subscriptionattributepb.GetSubscriptionAttributeListPageDataResponse, error) {
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

	query := `WITH enriched AS (SELECT id, subscription_id, attribute_id, value, active, date_created, date_modified FROM subscription_attribute WHERE active = true AND ($1::text IS NULL OR $1::text = '' OR value ILIKE $1)), counted AS (SELECT COUNT(*) as total FROM enriched) SELECT e.*, c.total FROM enriched e, counted c ORDER BY ` + sortField + ` ` + sortOrder + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var subscriptionAttributes []*subscriptionattributepb.SubscriptionAttribute
	var totalCount int64
	for rows.Next() {
		var id, subscriptionId, attributeId, attributeValue string
		var active bool
		var dateCreated, dateModified time.Time
		var total int64
		if err := rows.Scan(&id, &subscriptionId, &attributeId, &attributeValue, &active, &dateCreated, &dateModified, &total); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		totalCount = total

		rawData := map[string]interface{}{
			"id":             id,
			"subscriptionId": subscriptionId,
			"attributeId":    attributeId,
			"value":          attributeValue,
			"active":         active,
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
		subscriptionAttribute := &subscriptionattributepb.SubscriptionAttribute{}
		if err := protojson.Unmarshal(dataJSON, subscriptionAttribute); err == nil {
			subscriptionAttributes = append(subscriptionAttributes, subscriptionAttribute)
		}
	}
	totalPages := int32((totalCount + int64(limit) - 1) / int64(limit))
	return &subscriptionattributepb.GetSubscriptionAttributeListPageDataResponse{SubscriptionAttributeList: subscriptionAttributes, Pagination: &commonpb.PaginationResponse{TotalItems: int32(totalCount), CurrentPage: &page, TotalPages: &totalPages, HasNext: page < totalPages, HasPrev: page > 1}, Success: true}, nil
}

// GetSubscriptionAttributeItemPageData retrieves subscription attribute item page data
func (r *PostgresSubscriptionAttributeRepository) GetSubscriptionAttributeItemPageData(ctx context.Context, req *subscriptionattributepb.GetSubscriptionAttributeItemPageDataRequest) (*subscriptionattributepb.GetSubscriptionAttributeItemPageDataResponse, error) {
	if req == nil || req.SubscriptionAttributeId == "" {
		return nil, fmt.Errorf("subscription attribute ID required")
	}
	query := `SELECT id, subscription_id, attribute_id, value, active, date_created, date_modified FROM subscription_attribute WHERE id = $1 AND active = true`
	row := r.db.QueryRowContext(ctx, query, req.SubscriptionAttributeId)
	var id, subscriptionId, attributeId, attributeValue string
	var active bool
	var dateCreated, dateModified time.Time
	if err := row.Scan(&id, &subscriptionId, &attributeId, &attributeValue, &active, &dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("subscription attribute not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	rawData := map[string]interface{}{
		"id":             id,
		"subscriptionId": subscriptionId,
		"attributeId":    attributeId,
		"value":          attributeValue,
		"active":         active,
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
	subscriptionAttribute := &subscriptionattributepb.SubscriptionAttribute{}
	if err := protojson.Unmarshal(dataJSON, subscriptionAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	return &subscriptionattributepb.GetSubscriptionAttributeItemPageDataResponse{SubscriptionAttribute: subscriptionAttribute, Success: true}, nil
}

// NewSubscriptionAttributeRepository creates a new PostgreSQL subscription_attribute repository (old-style constructor)
func NewSubscriptionAttributeRepository(db *sql.DB, tableName string) subscriptionattributepb.SubscriptionAttributeDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresSubscriptionAttributeRepository(dbOps, tableName)
}
