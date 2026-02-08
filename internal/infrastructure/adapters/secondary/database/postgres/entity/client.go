//go:build postgres

package entity

import (
	"time"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "client", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres client repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
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
//
type PostgresClientRepository struct {
	clientpb.UnimplementedClientDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresClientRepository creates a new PostgreSQL client repository
func NewPostgresClientRepository(dbOps interfaces.DatabaseOperation, tableName string) clientpb.ClientDomainServiceServer {
	if tableName == "" {
		tableName = "client" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresClientRepository{
		dbOps:     dbOps,
		db:        db,
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
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	client := &clientpb.Client{}
	if err := protojson.Unmarshal(resultJSON, client); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &clientpb.CreateClientResponse{
		Data: []*clientpb.Client{client},
	}, nil
}

// ReadClient retrieves a client using common PostgreSQL operations
func (r *PostgresClientRepository) ReadClient(ctx context.Context, req *clientpb.ReadClientRequest) (*clientpb.ReadClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read client: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	client := &clientpb.Client{}
	if err := protojson.Unmarshal(resultJSON, client); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
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
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	client := &clientpb.Client{}
	if err := protojson.Unmarshal(resultJSON, client); err != nil {
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
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var clients []*clientpb.Client
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		client := &clientpb.Client{}
		if err := protojson.Unmarshal(resultJSON, client); err != nil {
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

// GetClientListPageData retrieves clients with advanced filtering, sorting, searching, and pagination using CTE
func (r *PostgresClientRepository) GetClientListPageData(
	ctx context.Context,
	req *clientpb.GetClientListPageDataRequest,
) (*clientpb.GetClientListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get client list page data request is required")
	}

	// Build search condition
	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	// Default pagination values
	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		// Handle offset pagination
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	// Default sort
	sortField := "date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// CTE Query - Single round-trip with enriched user data
	// Performance Notes:
	// - INDEX RECOMMENDATION: Create index on client.user_id (foreign key)
	// - INDEX RECOMMENDATION: Create index on user.first_name, user.last_name, user.email_address for search performance
	// - INDEX RECOMMENDATION: Create index on client.active for filtering active records
	// - INDEX RECOMMENDATION: Create index on client.date_created for default sorting
	// - INDEX RECOMMENDATION: Create index on client.internal_id for search
	query := `
		WITH enriched AS (
			SELECT
				c.id,
				c.user_id,
				c.active,
				c.internal_id,
				c.date_created,
				c.date_modified,
				-- User fields (1:1 relationship) - NO JSONB in domain model, direct fields
				u.id as user_id_value,
				u.first_name as user_first_name,
				u.last_name as user_last_name,
				u.email_address as user_email_address,
				u.mobile_number as user_phone_number
			FROM client c
			LEFT JOIN "user" u ON c.user_id = u.id AND u.active = true
			WHERE c.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
				   u.first_name ILIKE $1 OR
				   u.last_name ILIKE $1 OR
				   u.email_address ILIKE $1 OR
				   c.internal_id ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query client list page data: %w", err)
	}
	defer rows.Close()

	var clients []*clientpb.Client
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			userId             string
			active             bool
			internalId         *string
			dateCreated        time.Time
			dateModified       time.Time
			// User fields
			userIdValue       *string
			userFirstName     *string
			userLastName      *string
			userEmailAddress  *string
			userPhoneNumber   *string
			total             int64
		)

		err := rows.Scan(
			&id,
			&userId,
			&active,
			&internalId,
			&dateCreated,
			&dateModified,
			&userIdValue,
			&userFirstName,
			&userLastName,
			&userEmailAddress,
			&userPhoneNumber,
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

	// CTE Query - Single round-trip with enriched user data
	query := `
		WITH enriched AS (
			SELECT
				c.id,
				c.user_id,
				c.active,
				c.internal_id,
				c.date_created,
				c.date_modified,
				-- User fields (1:1 relationship) - NO JSONB in domain model, direct fields
				u.id as user_id_value,
				u.first_name as user_first_name,
				u.last_name as user_last_name,
				u.email_address as user_email_address,
				u.mobile_number as user_phone_number
			FROM client c
			LEFT JOIN "user" u ON c.user_id = u.id AND u.active = true
			WHERE c.id = $1 AND c.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.ClientId)

	var (
		id                 string
		userId             string
		active             bool
		internalId         *string
		dateCreated        time.Time
		dateModified       time.Time
		// User fields
		userIdValue       *string
		userFirstName     *string
		userLastName      *string
		userEmailAddress  *string
		userPhoneNumber   *string
	)

	err := row.Scan(
		&id,
		&userId,
		&active,
		&internalId,
		&dateCreated,
		&dateModified,
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

	return &clientpb.GetClientItemPageDataResponse{
		Client:  client,
		Success: true,
	}, nil
}


// deref safely dereferences a *string, returning "" if nil.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// NewClientRepository creates a new PostgreSQL client repository (old-style constructor)
func NewClientRepository(db *sql.DB, tableName string) clientpb.ClientDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresClientRepository(dbOps, tableName)
}
