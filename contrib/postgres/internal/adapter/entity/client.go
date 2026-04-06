package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/consumer"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	clientcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_category"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Client, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres client repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresClientRepository(dbOps, tableName), nil
	})
}

// PostgresClientRepository implements client CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_client_user_id ON client(user_id) - Foreign key relationship to user table
//   - CREATE INDEX idx_client_active ON client(active) - Filter active records
//   - CREATE INDEX idx_client_date_created ON client(date_created DESC) - Default sorting
//   - CREATE INDEX idx_client_internal_id ON client(internal_id) - Search field
//   - CREATE INDEX idx_user_first_name ON "user"(first_name) - Search performance on joined table
//   - CREATE INDEX idx_user_last_name ON "user"(last_name) - Search performance on joined table
//   - CREATE INDEX idx_user_email_address ON "user"(email_address) - Search performance on joined table
//
// TODO: Add comprehensive tests for GetClientListPageData:
//   - Test with no search query (list all active clients)
//   - Test with search query matching user first_name
//   - Test with search query matching user last_name
//   - Test with search query matching user email_address
//   - Test with search query matching client internal_id
//   - Test pagination (page 1, page 2, page size variations)
//   - Test sorting (by different fields, ASC and DESC)
//   - Test with no matching results
//   - Test with inactive clients (should be filtered out)
//   - Test with null user_id (LEFT JOIN behavior)
//   - Test with inactive user (should be filtered out via JOIN condition)
//
// TODO: Add comprehensive tests for GetClientItemPageData:
//   - Test with valid client ID (with associated user)
//   - Test with valid client ID (without associated user - null user_id)
//   - Test with non-existent client ID
//   - Test with inactive client (should return not found)
//   - Test with client having inactive user (user fields should be null)
//   - Test timestamp parsing for date_created and date_modified
type PostgresClientRepository struct {
	clientpb.UnimplementedClientDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresClientRepository creates a new PostgreSQL client repository
func NewPostgresClientRepository(dbOps interfaces.DatabaseOperation, tableName string) clientpb.ClientDomainServiceServer {
	if tableName == "" {
		tableName = "client" // default fallback
	}

	return &PostgresClientRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateClient creates a new client using common PostgreSQL operations
func (r *PostgresClientRepository) CreateClient(ctx context.Context, req *clientpb.CreateClientRequest) (*clientpb.CreateClientResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("client data is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	client := &clientpb.Client{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, client); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &clientpb.CreateClientResponse{
		Data: []*clientpb.Client{client},
	}, nil
}

// ReadClient retrieves a client with joined user data using a custom SQL query
func (r *PostgresClientRepository) ReadClient(ctx context.Context, req *clientpb.ReadClientRequest) (*clientpb.ReadClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	// Custom query that JOINs with user table to populate nested User field
	query := `
		SELECT
			c.id,
			c.user_id,
			c.active,
			c.internal_id,
			c.date_created,
			c.date_modified,
			c.name,
			c.street_address,
			c.city,
			c.province,
			c.postal_code,
			c.notes,
			c.category_id,
			c.payment_term_id,
			u.id as user_id_value,
			u.first_name as user_first_name,
			u.last_name as user_last_name,
			u.email_address as user_email_address,
			u.mobile_number as user_phone_number
		FROM client c
		LEFT JOIN "user" u ON c.user_id = u.id
		WHERE c.id = $1 AND c.active = true
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.Data.Id)

	var (
		id               string
		userId           string
		active           bool
		internalId       *string
		dateCreated      time.Time
		dateModified     time.Time
		name             *string
		streetAddress    *string
		city             *string
		province         *string
		postalCode       *string
		notes            *string
		categoryId       *string
		paymentTermID    *string
		userIdValue      *string
		userFirstName    *string
		userLastName     *string
		userEmailAddress *string
		userPhoneNumber  *string
	)

	err := row.Scan(
		&id,
		&userId,
		&active,
		&internalId,
		&dateCreated,
		&dateModified,
		&name,
		&streetAddress,
		&city,
		&province,
		&postalCode,
		&notes,
		&categoryId,
		&paymentTermID,
		&userIdValue,
		&userFirstName,
		&userLastName,
		&userEmailAddress,
		&userPhoneNumber,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("client with ID '%s' not found", req.Data.Id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read client: %w", err)
	}

	client := &clientpb.Client{
		Id:     id,
		UserId: userId,
		Active: active,
	}

	if internalId != nil {
		client.InternalId = *internalId
	}

	// CRM fields
	client.Name = name
	client.StreetAddress = streetAddress
	client.City = city
	client.Province = province
	client.PostalCode = postalCode
	client.Notes = notes
	client.CategoryId = categoryId
	if paymentTermID != nil {
		client.PaymentTermId = paymentTermID
	}

	// Populate joined user data
	if userIdValue != nil {
		client.User = &userpb.User{Id: deref(userIdValue)}
		client.User.FirstName = deref(userFirstName)
		client.User.LastName = deref(userLastName)
		client.User.EmailAddress = deref(userEmailAddress)
		client.User.MobileNumber = deref(userPhoneNumber)
	}

	// Parse timestamps
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		client.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		client.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		client.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		client.DateModifiedString = &dmStr
	}

	return &clientpb.ReadClientResponse{
		Data: []*clientpb.Client{client},
	}, nil
}

// UpdateClient updates a client using common PostgreSQL operations
func (r *PostgresClientRepository) UpdateClient(ctx context.Context, req *clientpb.UpdateClientRequest) (*clientpb.UpdateClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update client: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	client := &clientpb.Client{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, client); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &clientpb.UpdateClientResponse{
		Data: []*clientpb.Client{client},
	}, nil
}

// DeleteClient deletes a client using common PostgreSQL operations
func (r *PostgresClientRepository) DeleteClient(ctx context.Context, req *clientpb.DeleteClientRequest) (*clientpb.DeleteClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete client: %w", err)
	}

	return &clientpb.DeleteClientResponse{
		Success: true,
	}, nil
}

// ListClients lists clients using common PostgreSQL operations
func (r *PostgresClientRepository) ListClients(ctx context.Context, req *clientpb.ListClientsRequest) (*clientpb.ListClientsResponse, error) {
	// Pass through filters from the request (e.g. user_id equality for FindOrCreateClient)
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}

	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var clients []*clientpb.Client
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			// Log error and continue with next item
			continue
		}

		client := &clientpb.Client{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, client); err != nil {
			// Log error and continue with next item
			continue
		}
		clients = append(clients, client)
	}

	return &clientpb.ListClientsResponse{
		Data: clients,
	}, nil
}

// Example implementation for Create (commented out until database schema is defined):
/*
func (r *DBClientRepository) Create(ctx context.Context, req *clientpb.CreateClientRequest) (*clientpb.CreateClientResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("client data is required")
	}

	query := `
		INSERT INTO clients (user_id, active, internal_id, date_created, date_modified)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id, date_created, date_modified
	`

	var id string
	var dateCreated, dateModified time.Time

	err := r.db.QueryRowContext(ctx, query,
		req.Data.UserId,
		req.Data.Active,
		req.Data.InternalId
	).Scan(&id, &dateCreated, &dateModified)

	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	client := &clientpb.Client{
		Id:                 id,
		UserId:             req.Data.UserId,
		DateCreated:        dateCreated.Unix(),
		DateCreatedString:  dateCreated.Format(time.RFC3339),
		DateModified:       dateModified.Unix(),
		DateModifiedString: dateModified.Format(time.RFC3339),
		Active:             req.Data.Active,
		InternalId:         req.Data.InternalId,
	}

	return &clientpb.CreateClientResponse{
		Data: []*clientpb.Client{client},
	}, nil
}
*/

// clientSortAllowlist maps external sort field names to safe SQL column references.
var clientSortAllowlist = map[string]string{
	"date_created":    "c.date_created",
	"date_modified":   "c.date_modified",
	"u.first_name":    "u.first_name",
	"u.last_name":     "u.last_name",
	"u.email_address": "u.email_address",
	"u.mobile_number": "u.mobile_number",
}

// GetClientListPageData retrieves clients with advanced filtering, sorting, searching, and pagination using CTE
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresClientRepository) GetClientListPageData(
	ctx context.Context,
	req *clientpb.GetClientListPageDataRequest,
) (*clientpb.GetClientListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get client list page data request is required")
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy)
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	// Default pagination values
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

	// Allowlist-validated sort
	sortCol := "c.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		f := req.Sort.Fields[0]
		if col, ok := clientSortAllowlist[f.Field]; ok {
			sortCol = col
		}
		if f.Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// Build filter/search WHERE clauses ($1 is reserved for workspace_id, start at $2)
	searchFields := []string{"u.first_name", "u.last_name", "u.email_address", "c.internal_id"}
	filterClauses, filterArgs, nextIdx := postgresCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 2)

	whereSQL := "WHERE c.workspace_id = $1"
	if len(filterClauses) > 0 {
		whereSQL += " AND " + strings.Join(filterClauses, " AND ")
	}

	// LIMIT/OFFSET are the next two params after filter args
	limitIdx := nextIdx
	offsetIdx := nextIdx + 1
	// workspace_id is $1; filter args follow; then limit/offset
	queryArgs := []any{workspaceID}
	queryArgs = append(queryArgs, filterArgs...)
	queryArgs = append(queryArgs, limit, offset)

	// CTE Query - Single round-trip with COUNT(*) OVER() window function
	// Performance Notes:
	// - INDEX RECOMMENDATION: Create index on client.workspace_id (multi-tenancy filter)
	// - INDEX RECOMMENDATION: Create index on client.user_id (foreign key)
	// - INDEX RECOMMENDATION: Create index on user.first_name, user.last_name, user.email_address for search performance
	// - INDEX RECOMMENDATION: Create index on client.active for filtering active records
	// - INDEX RECOMMENDATION: Create index on client.date_created for default sorting
	// - INDEX RECOMMENDATION: Create index on client.internal_id for search
	query := fmt.Sprintf(`
		SELECT
			c.id,
			c.user_id,
			c.active,
			c.internal_id,
			c.date_created,
			c.date_modified,
			c.name,
			c.street_address,
			c.city,
			c.province,
			c.postal_code,
			c.notes,
			c.payment_term_id,
			pt.name AS payment_term_name,
			u.id as user_id_value,
			u.first_name as user_first_name,
			u.last_name as user_last_name,
			u.email_address as user_email_address,
			u.mobile_number as user_phone_number,
			(
				SELECT json_agg(json_build_object('id', cc.category_id, 'name', cat.name))
				FROM client_category cc
				JOIN category cat ON cc.category_id = cat.id
				WHERE cc.client_id = c.id AND cc.active = true
			) AS categories_json,
			COUNT(*) OVER() AS total_count
		FROM client c
		LEFT JOIN "user" u ON c.user_id = u.id
		LEFT JOIN payment_term pt ON c.payment_term_id = pt.id
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereSQL, sortCol, sortOrder, limitIdx, offsetIdx)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query client list page data: %w", err)
	}
	defer rows.Close()

	var clients []*clientpb.Client
	var totalCount int64

	for rows.Next() {
		var (
			id           string
			userId       string
			active       bool
			internalId   *string
			dateCreated  time.Time
			dateModified time.Time
			// CRM fields
			name            *string
			streetAddress   *string
			city            *string
			province        *string
			postalCode      *string
			notes           *string
			paymentTermID   *string
			paymentTermName *string
			// User fields
			userIdValue      *string
			userFirstName    *string
			userLastName     *string
			userEmailAddress *string
			userPhoneNumber  *string
			categoriesJSON   *string
			total            int64
		)

		err := rows.Scan(
			&id,
			&userId,
			&active,
			&internalId,
			&dateCreated,
			&dateModified,
			&name,
			&streetAddress,
			&city,
			&province,
			&postalCode,
			&notes,
			&paymentTermID,
			&paymentTermName,
			&userIdValue,
			&userFirstName,
			&userLastName,
			&userEmailAddress,
			&userPhoneNumber,
			&categoriesJSON,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan client row: %w", err)
		}

		totalCount = total

		client := &clientpb.Client{
			Id:     id,
			UserId: userId,
			Active: active,
		}

		// Handle nullable fields
		if internalId != nil {
			client.InternalId = *internalId
		}

		// CRM fields
		client.Name = name
		client.StreetAddress = streetAddress
		client.City = city
		client.Province = province
		client.PostalCode = postalCode
		client.Notes = notes
		if paymentTermID != nil {
			client.PaymentTermId = paymentTermID
		}
		if paymentTermName != nil && *paymentTermName != "" {
			client.PaymentTerm = &paymenttermpb.PaymentTerm{Name: *paymentTermName}
		}

		// Populate categories from aggregated JSON
		if categoriesJSON != nil && *categoriesJSON != "" {
			var raw []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}
			if err := json.Unmarshal([]byte(*categoriesJSON), &raw); err == nil {
				for _, r := range raw {
					cat := &clientcategorypb.ClientCategory{
						CategoryId: r.ID,
						Category: &commonpb.Category{
							Name: r.Name,
						},
					}
					client.Categories = append(client.Categories, cat)
				}
			}
		}

		// Populate joined user data
		if userIdValue != nil {
			client.User = &userpb.User{Id: deref(userIdValue)}
			client.User.FirstName = deref(userFirstName)
			client.User.LastName = deref(userLastName)
			client.User.EmailAddress = deref(userEmailAddress)
			client.User.MobileNumber = deref(userPhoneNumber)
		}

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			client.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			client.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			client.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			client.DateModifiedString = &dmStr
		}

		clients = append(clients, client)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating client rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &clientpb.GetClientListPageDataResponse{
		ClientList: clients,
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

// GetClientItemPageData retrieves a single client with enhanced item page data using CTE
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresClientRepository) GetClientItemPageData(
	ctx context.Context,
	req *clientpb.GetClientItemPageDataRequest,
) (*clientpb.GetClientItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get client item page data request is required")
	}
	if req.ClientId == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy)
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	// CTE Query - Single round-trip with enriched user data and CRM fields
	query := `
		WITH enriched AS (
			SELECT
				c.id,
				c.user_id,
				c.active,
				c.internal_id,
				c.date_created,
				c.date_modified,
				-- CRM fields
				c.name,
				c.street_address,
				c.city,
				c.province,
				c.postal_code,
				c.notes,
				c.category_id,
				c.payment_term_id,
				-- User fields (1:1 relationship)
				u.id as user_id_value,
				u.first_name as user_first_name,
				u.last_name as user_last_name,
				u.email_address as user_email_address,
				u.mobile_number as user_phone_number
			FROM client c
			LEFT JOIN "user" u ON c.user_id = u.id
			WHERE c.id = $1 AND c.workspace_id = $2
		)
		SELECT * FROM enriched LIMIT 1;
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.ClientId, workspaceID)

	var (
		id           string
		userId       string
		active       bool
		internalId   *string
		dateCreated  time.Time
		dateModified time.Time
		// CRM fields
		name          *string
		streetAddress *string
		city          *string
		province      *string
		postalCode    *string
		notes         *string
		categoryId    *string
		paymentTermID *string
		// User fields
		userIdValue      *string
		userFirstName    *string
		userLastName     *string
		userEmailAddress *string
		userPhoneNumber  *string
	)

	err := row.Scan(
		&id,
		&userId,
		&active,
		&internalId,
		&dateCreated,
		&dateModified,
		&name,
		&streetAddress,
		&city,
		&province,
		&postalCode,
		&notes,
		&categoryId,
		&paymentTermID,
		&userIdValue,
		&userFirstName,
		&userLastName,
		&userEmailAddress,
		&userPhoneNumber,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("client with ID '%s' not found", req.ClientId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query client item page data: %w", err)
	}

	client := &clientpb.Client{
		Id:     id,
		UserId: userId,
		Active: active,
	}

	// Handle nullable fields
	if internalId != nil {
		client.InternalId = *internalId
	}

	// CRM fields
	client.Name = name
	client.StreetAddress = streetAddress
	client.City = city
	client.Province = province
	client.PostalCode = postalCode
	client.Notes = notes
	client.CategoryId = categoryId
	if paymentTermID != nil {
		client.PaymentTermId = paymentTermID
	}

	// Populate joined user data
	if userIdValue != nil {
		client.User = &userpb.User{Id: deref(userIdValue)}
		client.User.FirstName = deref(userFirstName)
		client.User.LastName = deref(userLastName)
		client.User.EmailAddress = deref(userEmailAddress)
		client.User.MobileNumber = deref(userPhoneNumber)
	}

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		client.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		client.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		client.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		client.DateModifiedString = &dmStr
	}

	// Load categories (tags) for this client via separate query
	categories, err := r.loadClientCategories(ctx, id)
	if err == nil && len(categories) > 0 {
		client.Categories = categories
	}

	return &clientpb.GetClientItemPageDataResponse{
		Client:  client,
		Success: true,
	}, nil
}

// loadClientCategories loads the category tags for a client via JOIN through client_category to category
func (r *PostgresClientRepository) loadClientCategories(ctx context.Context, clientId string) ([]*clientcategorypb.ClientCategory, error) {
	query := `
		SELECT
			cc.id,
			cc.client_id,
			cc.category_id,
			cat.name,
			cat.description
		FROM client_category cc
		INNER JOIN category cat ON cc.category_id = cat.id
		WHERE cc.client_id = $1 AND cc.active = true AND cat.active = true
		ORDER BY cat.name ASC
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, clientId)
	if err != nil {
		return nil, fmt.Errorf("failed to load client categories: %w", err)
	}
	defer rows.Close()

	var categories []*clientcategorypb.ClientCategory
	for rows.Next() {
		var (
			ccId       string
			ccClientId string
			ccCatId    string
			catName    string
			catDesc    *string
		)
		if err := rows.Scan(&ccId, &ccClientId, &ccCatId, &catName, &catDesc); err != nil {
			return nil, fmt.Errorf("failed to scan client category row: %w", err)
		}

		cat := &commonpb.Category{
			Id:   ccCatId,
			Name: catName,
		}
		if catDesc != nil {
			cat.Description = *catDesc
		}

		categories = append(categories, &clientcategorypb.ClientCategory{
			Id:         ccId,
			ClientId:   ccClientId,
			CategoryId: ccCatId,
			Category:   cat,
			Active:     true,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating client category rows: %w", err)
	}

	return categories, nil
}

// deref safely dereferences a *string, returning "" if nil.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// SearchClientsByName searches clients by company name or user first/last name using ILIKE
func (r *PostgresClientRepository) SearchClientsByName(ctx context.Context, req *clientpb.SearchClientsByNameRequest) (*clientpb.SearchClientsByNameResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("search clients by name request is required")
	}

	limit := int32(20)
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
	}

	query := `
		SELECT
			c.id,
			COALESCE(
				NULLIF(c.name, ''),
				NULLIF(TRIM(CONCAT(u.first_name, ' ', u.last_name)), ''),
				c.id
			) AS label
		FROM client c
		LEFT JOIN "user" u ON c.user_id = u.id
		WHERE c.active = true
			AND ($1::text = '' OR
				c.name ILIKE $1 OR
				u.first_name ILIKE $1 OR
				u.last_name ILIKE $1)
		ORDER BY label ASC
		LIMIT $2
	`

	pattern := ""
	if req.Query != "" {
		pattern = "%" + req.Query + "%"
	}

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search clients by name: %w", err)
	}
	defer rows.Close()

	var results []*clientpb.SearchClientResult
	for rows.Next() {
		var id, label string
		if err := rows.Scan(&id, &label); err != nil {
			return nil, fmt.Errorf("failed to scan search client row: %w", err)
		}
		results = append(results, &clientpb.SearchClientResult{
			Id:    id,
			Label: label,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search client rows: %w", err)
	}

	return &clientpb.SearchClientsByNameResponse{
		Results: results,
		Success: true,
	}, nil
}

// NewClientRepository creates a new PostgreSQL client repository (old-style constructor)
func NewClientRepository(db *sql.DB, tableName string) clientpb.ClientDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresClientRepository(dbOps, tableName)
}
