//go:build mysql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/consumer"
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	subscriptionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_attribute"
	"google.golang.org/protobuf/encoding/protojson"
)

// subscriptionAttributeSortableSQLCols is the sort-column whitelist for
// GetSubscriptionAttributeListPageData (A2 fail-closed guard).
var subscriptionAttributeSortableSQLCols = []string{
	"value",
	"date_created",
	"date_modified",
}

// MySQLSubscriptionAttributeRepository implements subscription_attribute CRUD operations using MySQL 8.0+.
type MySQLSubscriptionAttributeRepository struct {
	subscriptionattributepb.UnimplementedSubscriptionAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.SubscriptionAttribute, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql subscription_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLSubscriptionAttributeRepository(dbOps, tableName), nil
	})
}

// NewMySQLSubscriptionAttributeRepository creates a new MySQL subscription attribute repository.
func NewMySQLSubscriptionAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) subscriptionattributepb.SubscriptionAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "subscription_attribute"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &MySQLSubscriptionAttributeRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateSubscriptionAttribute creates a new subscription attribute using common MySQL operations.
func (r *MySQLSubscriptionAttributeRepository) CreateSubscriptionAttribute(ctx context.Context, req *subscriptionattributepb.CreateSubscriptionAttributeRequest) (*subscriptionattributepb.CreateSubscriptionAttributeResponse, error) {
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
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, subscriptionAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &subscriptionattributepb.CreateSubscriptionAttributeResponse{
		Data: []*subscriptionattributepb.SubscriptionAttribute{subscriptionAttribute},
	}, nil
}

// ReadSubscriptionAttribute retrieves a subscription attribute using common MySQL operations.
func (r *MySQLSubscriptionAttributeRepository) ReadSubscriptionAttribute(ctx context.Context, req *subscriptionattributepb.ReadSubscriptionAttributeRequest) (*subscriptionattributepb.ReadSubscriptionAttributeResponse, error) {
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
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, subscriptionAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &subscriptionattributepb.ReadSubscriptionAttributeResponse{
		Data: []*subscriptionattributepb.SubscriptionAttribute{subscriptionAttribute},
	}, nil
}

// UpdateSubscriptionAttribute updates a subscription attribute using common MySQL operations.
func (r *MySQLSubscriptionAttributeRepository) UpdateSubscriptionAttribute(ctx context.Context, req *subscriptionattributepb.UpdateSubscriptionAttributeRequest) (*subscriptionattributepb.UpdateSubscriptionAttributeResponse, error) {
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
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, subscriptionAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &subscriptionattributepb.UpdateSubscriptionAttributeResponse{
		Data: []*subscriptionattributepb.SubscriptionAttribute{subscriptionAttribute},
	}, nil
}

// DeleteSubscriptionAttribute deletes a subscription attribute using common MySQL operations.
func (r *MySQLSubscriptionAttributeRepository) DeleteSubscriptionAttribute(ctx context.Context, req *subscriptionattributepb.DeleteSubscriptionAttributeRequest) (*subscriptionattributepb.DeleteSubscriptionAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription attribute ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete subscription attribute: %w", err)
	}

	return &subscriptionattributepb.DeleteSubscriptionAttributeResponse{
		Success: true,
	}, nil
}

// ListSubscriptionAttributes lists subscription attributes using common MySQL operations.
func (r *MySQLSubscriptionAttributeRepository) ListSubscriptionAttributes(ctx context.Context, req *subscriptionattributepb.ListSubscriptionAttributesRequest) (*subscriptionattributepb.ListSubscriptionAttributesResponse, error) {
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
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, subscriptionAttribute); err != nil {
			continue
		}
		subscriptionAttributes = append(subscriptionAttributes, subscriptionAttribute)
	}

	return &subscriptionattributepb.ListSubscriptionAttributesResponse{
		Data: subscriptionAttributes,
	}, nil
}

// GetSubscriptionAttributeListPageData retrieves paginated subscription attribute list data.
//
// Dialect translation from postgres gold standard:
//   - $N → ? (MySQL positional placeholders)
//   - ILIKE → LIKE (MySQL ci collation)
//   - active = true → active = 1
//   - COUNT(*) OVER () replaces counted-CTE + cross join (MySQL 8.0+ window function)
//   - WHERE workspace_id = ? added for multi-tenancy (via subscription FK join)
//   - mysqlCore.BuildOrderBy used for safe sort interpolation
func (r *MySQLSubscriptionAttributeRepository) GetSubscriptionAttributeListPageData(ctx context.Context, req *subscriptionattributepb.GetSubscriptionAttributeListPageDataRequest) (*subscriptionattributepb.GetSubscriptionAttributeListPageDataResponse, error) {
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
	orderBy, err := mysqlCore.BuildOrderBy(subscriptionAttributeSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, fmt.Errorf("invalid sort for subscription attribute list: %w", err)
	}

	// A1: subscription_attribute has no workspace_id column of its own; tenancy is
	// inherited through its subscription FK, so the predicate scopes on the joined
	// subscription's workspace_id. The explicit sa.* column list keeps the scan
	// unaffected by the join. Empty wsID = service-to-service call → no scoping.
	// Dialect: COUNT(*) OVER () replaces counted-CTE + CROSS JOIN (MySQL 8.0+),
	// $N → ?, ILIKE → LIKE, active = true → active = 1,
	// WHERE workspace_id added (postgres gold was missing this — added here per brief).
	wsID := consumer.GetWorkspaceIDFromContext(ctx)
	query := `WITH enriched AS (
		SELECT sa.id, sa.subscription_id, sa.attribute_id, sa.value, sa.active, sa.date_created, sa.date_modified
		FROM subscription_attribute sa
		LEFT JOIN subscription s ON sa.subscription_id = s.id
		WHERE sa.active = 1
		  AND (? = '' OR s.workspace_id = ?)
		  AND (? IS NULL OR ? = '' OR sa.value LIKE ?)
	) SELECT e.*, COUNT(*) OVER () AS total FROM enriched e ` + orderBy + ` LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, wsID, wsID, searchPattern, searchPattern, searchPattern, limit, offset)
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
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(dataJSON, subscriptionAttribute); err == nil {
			subscriptionAttributes = append(subscriptionAttributes, subscriptionAttribute)
		}
	}
	totalPages := int32((totalCount + int64(limit) - 1) / int64(limit))
	return &subscriptionattributepb.GetSubscriptionAttributeListPageDataResponse{
		SubscriptionAttributeList: subscriptionAttributes,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     page < totalPages,
			HasPrev:     page > 1,
		},
		Success: true,
	}, nil
}

// GetSubscriptionAttributeItemPageData retrieves subscription attribute item page data.
//
// Dialect: $1 → ?, active = true → active = 1.
func (r *MySQLSubscriptionAttributeRepository) GetSubscriptionAttributeItemPageData(ctx context.Context, req *subscriptionattributepb.GetSubscriptionAttributeItemPageDataRequest) (*subscriptionattributepb.GetSubscriptionAttributeItemPageDataResponse, error) {
	if req == nil || req.SubscriptionAttributeId == "" {
		return nil, fmt.Errorf("subscription attribute ID required")
	}
	query := `SELECT id, subscription_id, attribute_id, value, active, date_created, date_modified FROM subscription_attribute WHERE id = ? AND active = 1`
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
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(dataJSON, subscriptionAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	return &subscriptionattributepb.GetSubscriptionAttributeItemPageDataResponse{SubscriptionAttribute: subscriptionAttribute, Success: true}, nil
}

// NewSubscriptionAttributeRepository creates a new MySQL subscription_attribute repository (old-style constructor).
func NewSubscriptionAttributeRepository(db *sql.DB, tableName string) subscriptionattributepb.SubscriptionAttributeDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLSubscriptionAttributeRepository(dbOps, tableName)
}
