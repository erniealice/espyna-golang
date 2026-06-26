//go:build mysql

// Dialect translation from postgres gold standard:
//   - $1,$2,... → ? (MySQL positional placeholders, args in same left-to-right order)
//   - "ident"   → `ident` (backtick quoting for reserved words)
//   - ILIKE     → LIKE (MySQL ci collation handles case-insensitivity)
//   - active = true → active = 1 (MySQL TINYINT(1) booleans)
//   - LIMIT $N OFFSET $N → LIMIT ? OFFSET ? (trailing positional args)
//   - COUNT(*) OVER () stays — MySQL 8.0+ supports window functions
//   - core.BuildOrderBy → mysqlCore.BuildOrderBy (backtick quoting)
//
// CRITICAL: workspace_id isolation enforced on every raw-SQL query.
// Centavos (total_amount, unit_price, etc.) are never scaled in SQL.
package revenue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/shared/identity"
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
)

// revenueSortableSQLCols lists SQL column names safe for ORDER BY interpolation.
var revenueSortableSQLCols = []string{
	"reference_number",
	"total_amount",
	"status",
	"date_created",
	"date_modified",
	"client_name",
	"revenue_date_string",
}

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.Revenue, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql revenue repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLRevenueRepository(dbOps, tableName), nil
	})
}

// MySQLRevenueRepository implements revenue CRUD operations using MySQL 8.0+.
type MySQLRevenueRepository struct {
	revenuepb.UnimplementedRevenueDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLRevenueRepository creates a new MySQL revenue repository.
func NewMySQLRevenueRepository(dbOps interfaces.DatabaseOperation, tableName string) revenuepb.RevenueDomainServiceServer {
	if tableName == "" {
		tableName = "revenue"
	}
	return &MySQLRevenueRepository{
		dbOps:     dbOps,
		db:        getDB(dbOps),
		tableName: tableName,
	}
}

// CreateRevenue creates a new revenue record.
func (r *MySQLRevenueRepository) CreateRevenue(ctx context.Context, req *revenuepb.CreateRevenueRequest) (*revenuepb.CreateRevenueResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("revenue data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "revenueDate")
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")
	convertMillisToTime(data, "dueDate")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create revenue: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	revenue := &revenuepb.Revenue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, revenue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenuepb.CreateRevenueResponse{
		Success: true,
		Data:    []*revenuepb.Revenue{revenue},
	}, nil
}

// ReadRevenue retrieves a revenue record by ID.
func (r *MySQLRevenueRepository) ReadRevenue(ctx context.Context, req *revenuepb.ReadRevenueRequest) (*revenuepb.ReadRevenueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read revenue: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	revenue := &revenuepb.Revenue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, revenue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenuepb.ReadRevenueResponse{
		Success: true,
		Data:    []*revenuepb.Revenue{revenue},
	}, nil
}

// UpdateRevenue updates a revenue record.
func (r *MySQLRevenueRepository) UpdateRevenue(ctx context.Context, req *revenuepb.UpdateRevenueRequest) (*revenuepb.UpdateRevenueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "revenueDate")
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")
	convertMillisToTime(data, "dueDate")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update revenue: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	revenue := &revenuepb.Revenue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, revenue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenuepb.UpdateRevenueResponse{
		Success: true,
		Data:    []*revenuepb.Revenue{revenue},
	}, nil
}

// DeleteRevenue soft-deletes a revenue record.
func (r *MySQLRevenueRepository) DeleteRevenue(ctx context.Context, req *revenuepb.DeleteRevenueRequest) (*revenuepb.DeleteRevenueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete revenue: %w", err)
	}

	return &revenuepb.DeleteRevenueResponse{Success: true}, nil
}

// ListRevenues lists revenue records with optional filters.
func (r *MySQLRevenueRepository) ListRevenues(ctx context.Context, req *revenuepb.ListRevenuesRequest) (*revenuepb.ListRevenuesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list revenues: %w", err)
	}

	var revenues []*revenuepb.Revenue
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal revenue row: %v", err)
			continue
		}
		revenue := &revenuepb.Revenue{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, revenue); err != nil {
			log.Printf("WARN: protojson unmarshal revenue: %v", err)
			continue
		}
		revenues = append(revenues, revenue)
	}

	return &revenuepb.ListRevenuesResponse{
		Success: true,
		Data:    revenues,
	}, nil
}

// GetRevenueListPageData retrieves revenues with pagination, filtering, sorting,
// and search using a CTE. Joins client and location tables for enriched display.
//
// Dialect changes from postgres gold standard:
//   - $1,$2,... → ? (positional); args resequenced: [workspaceID, ...filterArgs, limit, offset]
//   - ILIKE → LIKE
//   - active = true → active = 1
//   - LIMIT $N OFFSET $N → LIMIT ? OFFSET ? (two trailing args)
//   - COUNT(*) OVER () stays (MySQL 8.0+ window function)
//   - mysqlCore.BuildOrderBy for sort (backtick-quoted column names)
//
// CRITICAL: Always filters by workspace_id for multi-tenancy.
func (r *MySQLRevenueRepository) GetRevenueListPageData(
	ctx context.Context,
	req *revenuepb.GetRevenueListPageDataRequest,
) (*revenuepb.GetRevenueListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get revenue list page data request is required")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

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

	// Sort — fail-closed against per-entity whitelist (A2 guard).
	// mysqlCore.BuildOrderBy uses backtick quoting.
	orderByClause, err := mysqlCore.BuildOrderBy(revenueSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}

	// Build filter/search WHERE clauses.
	// First arg (?) is workspace_id; filter builder starts at index 2 for arg count parity.
	searchFields := []string{"rv.reference_number", "c.name"}
	filterClauses, filterArgs, _ := mysqlCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 2)

	whereSQL := "WHERE rv.active = 1 AND rv.workspace_id = ?"
	if len(filterClauses) > 0 {
		whereSQL += " AND " + strings.Join(filterClauses, " AND ")
	}

	// Args: [workspaceID, ...filterArgs, limit, offset]
	queryArgs := []any{workspaceID}
	queryArgs = append(queryArgs, filterArgs...)
	queryArgs = append(queryArgs, limit, offset)

	// 20260517 advance-cash-events: expose advance_collection_id on list row.
	// Two-step CTE mirrors the postgres gold standard structure.
	// COUNT(*) OVER() is valid in MySQL 8.0+.
	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				rv.id,
				rv.date_created,
				rv.date_modified,
				rv.active,
				rv.name,
				rv.client_id,
				rv.revenue_date_string,
				rv.total_amount,
				rv.currency,
				rv.status,
				rv.reference_number,
				rv.notes,
				rv.revenue_category_id,
				rv.location_id,
				rv.payment_term_id,
				rv.due_date_string,
				rv.subscription_id,
				rv.advance_collection_id,
				COALESCE(c.name, '') as client_name,
				COALESCE(l.name, '') as location_name,
				COALESCE(pt.name, '') as payment_term_name,
				EXISTS(SELECT 1 FROM treasury_collection tc WHERE tc.revenue_id = rv.id) as has_collection,
				COUNT(*) OVER() AS total_count
			FROM %s rv
			LEFT JOIN client c ON rv.client_id = c.id AND c.active = 1
			LEFT JOIN location l ON rv.location_id = l.id AND l.active = 1
			LEFT JOIN payment_term pt ON rv.payment_term_id = pt.id
			%s
		)
		SELECT * FROM enriched
		%s
		LIMIT ? OFFSET ?
	`, r.tableName, whereSQL, orderByClause)

	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue list page data: %w", err)
	}
	defer rows.Close()

	var revenues []*revenuepb.Revenue
	var totalCount int64

	for rows.Next() {
		var (
			id                  string
			dateCreated         time.Time
			dateModified        time.Time
			active              bool
			name                string
			clientID            *string
			revenueDateString   *string
			totalAmount         int64
			currency            *string
			status              *string
			referenceNumber     *string
			notes               *string
			revenueCategoryID   *string
			locationID          *string
			paymentTermID       *string
			dueDateString       *string
			subscriptionID      *string
			advanceCollectionID *string
			clientName          string
			locationName        string
			paymentTermName     string
			hasCollection       bool
			total               int64
		)

		if err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&name,
			&clientID,
			&revenueDateString,
			&totalAmount,
			&currency,
			&status,
			&referenceNumber,
			&notes,
			&revenueCategoryID,
			&locationID,
			&paymentTermID,
			&dueDateString,
			&subscriptionID,
			&advanceCollectionID,
			&clientName,
			&locationName,
			&paymentTermName,
			&hasCollection,
			&total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan revenue row: %w", err)
		}

		totalCount = total

		revenue := &revenuepb.Revenue{
			Id:                id,
			Active:            active,
			Name:              name,
			TotalAmount:       totalAmount,
			ReferenceNumber:   referenceNumber,
			Notes:             notes,
			RevenueCategoryId: revenueCategoryID,
		}

		if clientID != nil {
			revenue.ClientId = *clientID
		}
		if locationID != nil {
			revenue.LocationId = *locationID
		}
		if currency != nil {
			revenue.Currency = *currency
		}
		if status != nil {
			revenue.Status = *status
		}
		if revenueDateString != nil {
			revenue.RevenueDate = revenueDateString
		}
		if paymentTermID != nil {
			revenue.PaymentTermId = paymentTermID
			if paymentTermName != "" {
				revenue.PaymentTerm = &paymenttermpb.PaymentTerm{
					Id:   *paymentTermID,
					Name: paymentTermName,
				}
			}
		}
		if dueDateString != nil {
			revenue.DueDate = dueDateString
		}
		if subscriptionID != nil {
			revenue.SubscriptionId = subscriptionID
		}
		if advanceCollectionID != nil {
			revenue.AdvanceCollectionId = advanceCollectionID
		}
		if hasCollection {
			hc := "has_collection"
			revenue.FulfillmentStatus = &hc
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			revenue.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			revenue.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			revenue.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			revenue.DateModifiedString = &dmStr
		}

		revenues = append(revenues, revenue)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating revenue rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &revenuepb.GetRevenueListPageDataResponse{
		RevenueList: revenues,
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

// GetRevenueItemPageData retrieves a single revenue with enriched data.
//
// Dialect changes: $1/$2 → ? (positional); active = true → active = 1.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
func (r *MySQLRevenueRepository) GetRevenueItemPageData(
	ctx context.Context,
	req *revenuepb.GetRevenueItemPageDataRequest,
) (*revenuepb.GetRevenueItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get revenue item page data request is required")
	}
	if req.RevenueId == "" {
		return nil, fmt.Errorf("revenue ID is required")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

	// 20260517 advance-cash-events: expose advance_collection_id.
	// Dialect: $1/$2 → ?; active = true → active = 1.
	query := `
		WITH enriched AS (
			SELECT
				rv.id,
				rv.date_created,
				rv.date_modified,
				rv.active,
				rv.name,
				rv.client_id,
				rv.revenue_date_string,
				rv.total_amount,
				rv.currency,
				rv.status,
				rv.reference_number,
				rv.notes,
				rv.revenue_category_id,
				rv.location_id,
				rv.payment_term_id,
				rv.due_date_string,
				rv.subscription_id,
				rv.advance_collection_id,
				COALESCE(c.name, '') as client_name,
				COALESCE(l.name, '') as location_name
			FROM revenue rv
			LEFT JOIN client c ON rv.client_id = c.id AND c.active = 1
			LEFT JOIN location l ON rv.location_id = l.id AND l.active = 1
			WHERE rv.id = ? AND rv.workspace_id = ? AND rv.active = 1
		)
		SELECT * FROM enriched LIMIT 1
	`

	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	// Arg order: revenueId (?), workspaceID (?) — same positional order as postgres.
	row := r.db.QueryRowContext(ctx, query, req.RevenueId, workspaceID)

	var (
		id                  string
		dateCreated         time.Time
		dateModified        time.Time
		active              bool
		name                string
		clientID            *string
		revenueDateString   *string
		totalAmount         int64
		currency            *string
		status              *string
		referenceNumber     *string
		notes               *string
		revenueCategoryID   *string
		locationID          *string
		paymentTermID       *string
		dueDateString       *string
		subscriptionID      *string
		advanceCollectionID *string
		clientName          string
		locationName        string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&name,
		&clientID,
		&revenueDateString,
		&totalAmount,
		&currency,
		&status,
		&referenceNumber,
		&notes,
		&revenueCategoryID,
		&locationID,
		&paymentTermID,
		&dueDateString,
		&subscriptionID,
		&advanceCollectionID,
		&clientName,
		&locationName,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("revenue with ID '%s' not found", req.RevenueId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue item page data: %w", err)
	}

	revenue := &revenuepb.Revenue{
		Id:                id,
		Active:            active,
		Name:              name,
		TotalAmount:       totalAmount,
		ReferenceNumber:   referenceNumber,
		Notes:             notes,
		RevenueCategoryId: revenueCategoryID,
	}

	if clientID != nil {
		revenue.ClientId = *clientID
	}
	if locationID != nil {
		revenue.LocationId = *locationID
	}
	if currency != nil {
		revenue.Currency = *currency
	}
	if status != nil {
		revenue.Status = *status
	}
	if revenueDateString != nil {
		revenue.RevenueDate = revenueDateString
	}
	if paymentTermID != nil {
		revenue.PaymentTermId = paymentTermID
	}
	if dueDateString != nil {
		revenue.DueDate = dueDateString
	}
	if subscriptionID != nil {
		revenue.SubscriptionId = subscriptionID
	}
	if advanceCollectionID != nil {
		revenue.AdvanceCollectionId = advanceCollectionID
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		revenue.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		revenue.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		revenue.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		revenue.DateModifiedString = &dmStr
	}

	return &revenuepb.GetRevenueItemPageDataResponse{
		Revenue: revenue,
		Success: true,
	}, nil
}

// NewRevenueRepository creates a new MySQL revenue repository (old-style constructor).
func NewRevenueRepository(db *sql.DB, tableName string) revenuepb.RevenueDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLRevenueRepository(dbOps, tableName)
}

// convertMillisToTime converts a millis-epoch value in a JSON map to time.Time.
// Protobuf int64 fields serialize to JSON strings via protojson (e.g. "1771886746000").
// MySQL timestamp columns need time.Time, not raw millis.
func convertMillisToTime(data map[string]any, jsonKey string) {
	v, ok := data[jsonKey]
	if !ok {
		return
	}
	switch val := v.(type) {
	case string:
		var millis int64
		if _, err := fmt.Sscanf(val, "%d", &millis); err == nil && millis > 1e12 {
			data[jsonKey] = time.UnixMilli(millis)
		}
	case float64:
		if val > 1e12 {
			data[jsonKey] = time.UnixMilli(int64(val))
		}
	}
}
