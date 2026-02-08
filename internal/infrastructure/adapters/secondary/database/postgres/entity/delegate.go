//go:build postgres

package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	delegatepb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate"
	delegateclientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate_client"
	userpb "leapfor.xyz/esqyma/golang/v1/domain/entity/user"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "delegate", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres delegate repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresDelegateRepository(dbOps, tableName), nil
	})
}

// PostgresDelegateRepository implements delegate CRUD operations using PostgreSQL
type PostgresDelegateRepository struct {
	delegatepb.UnimplementedDelegateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresDelegateRepository creates a new PostgreSQL delegate repository
func NewPostgresDelegateRepository(dbOps interfaces.DatabaseOperation, tableName string) delegatepb.DelegateDomainServiceServer {
	if tableName == "" {
		tableName = "delegate" // default fallback
	}
	return &PostgresDelegateRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateDelegate creates a new delegate using common PostgreSQL operations
func (r *PostgresDelegateRepository) CreateDelegate(ctx context.Context, req *delegatepb.CreateDelegateRequest) (*delegatepb.CreateDelegateResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("delegate data is required")
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
		return nil, fmt.Errorf("failed to create delegate: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	delegate := &delegatepb.Delegate{}
	if err := protojson.Unmarshal(resultJSON, delegate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &delegatepb.CreateDelegateResponse{
		Data: []*delegatepb.Delegate{delegate},
	}, nil
}

// ReadDelegate retrieves a delegate using common PostgreSQL operations
func (r *PostgresDelegateRepository) ReadDelegate(ctx context.Context, req *delegatepb.ReadDelegateRequest) (*delegatepb.ReadDelegateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read delegate: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	delegate := &delegatepb.Delegate{}
	if err := protojson.Unmarshal(resultJSON, delegate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &delegatepb.ReadDelegateResponse{
		Data: []*delegatepb.Delegate{delegate},
	}, nil
}

// UpdateDelegate updates a delegate using common PostgreSQL operations
func (r *PostgresDelegateRepository) UpdateDelegate(ctx context.Context, req *delegatepb.UpdateDelegateRequest) (*delegatepb.UpdateDelegateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate ID is required")
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
		return nil, fmt.Errorf("failed to update delegate: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	delegate := &delegatepb.Delegate{}
	if err := protojson.Unmarshal(resultJSON, delegate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &delegatepb.UpdateDelegateResponse{
		Data: []*delegatepb.Delegate{delegate},
	}, nil
}

// DeleteDelegate deletes a delegate using common PostgreSQL operations
func (r *PostgresDelegateRepository) DeleteDelegate(ctx context.Context, req *delegatepb.DeleteDelegateRequest) (*delegatepb.DeleteDelegateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete delegate: %w", err)
	}

	return &delegatepb.DeleteDelegateResponse{
		Success: true,
	}, nil
}

// ListDelegates lists delegates using common PostgreSQL operations
func (r *PostgresDelegateRepository) ListDelegates(ctx context.Context, req *delegatepb.ListDelegatesRequest) (*delegatepb.ListDelegatesResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list delegates: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var delegates []*delegatepb.Delegate
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		delegate := &delegatepb.Delegate{}
		if err := protojson.Unmarshal(resultJSON, delegate); err != nil {
			// Log error and continue with next item
			continue
		}
		delegates = append(delegates, delegate)
	}

	return &delegatepb.ListDelegatesResponse{
		Data: delegates,
	}, nil
}

// GetDelegateListPageData retrieves a paginated, filtered, sorted, and searchable list of delegates with user and client relationships
// This method uses CTEs (Common Table Expressions) to optimize query performance by loading all data in a single query
// TODO: Add unit tests for GetDelegateListPageData
func (r *PostgresDelegateRepository) GetDelegateListPageData(ctx context.Context, req *delegatepb.GetDelegateListPageDataRequest) (*delegatepb.GetDelegateListPageDataResponse, error) {
	// Extract pagination parameters with defaults
	limit := int32(20)
	page := int32(1)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
		if limit > 100 {
			limit = 100 // Cap at 100 items per page
		}
		if req.Pagination.GetOffset() != nil {
			page = req.Pagination.GetOffset().Page
			if page < 1 {
				page = 1
			}
		}
	}
	offset := (page - 1) * limit

	// Extract search query
	searchQuery := ""
	if req.Search != nil && req.Search.Query != "" {
		searchQuery = "%" + req.Search.Query + "%"
	}

	// Extract sort parameters with defaults
	sortField := "date_created"
	sortDirection := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == 1 { // DESC enum value
			sortDirection = "DESC"
		} else {
			sortDirection = "ASC"
		}
	}

	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_delegate_active ON delegate(active) WHERE active = true;
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_delegate_user_id ON delegate(user_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_delegate_date_created ON delegate(date_created DESC);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_delegate_client_delegate_id ON delegate_client(delegate_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_delegate_client_client_id ON delegate_client(client_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_delegate_client_active ON delegate_client(active) WHERE active = true;
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_user_active ON "user"(active) WHERE active = true;
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_user_first_name_trgm ON "user" USING gin(first_name gin_trgm_ops);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_user_last_name_trgm ON "user" USING gin(last_name gin_trgm_ops);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_user_email_trgm ON "user" USING gin(email_address gin_trgm_ops);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_client_active ON client(active) WHERE active = true;

	// Build the CTE query following the translation plan pattern
	query := `
		WITH
		-- CTE 1: Aggregate delegate_client relationships with client and user details
		delegate_clients_agg AS (
			SELECT
				dc.delegate_id,
				array_agg(
					DISTINCT jsonb_build_object(
						'id', dc.id,
						'delegate_id', dc.delegate_id,
						'client_id', dc.client_id,
						'date_created', dc.date_created,
						'date_modified', dc.date_modified,
						'active', dc.active,
						'client', jsonb_build_object(
							'id', c.id,
							'user_id', c.user_id,
							'date_created', c.date_created,
							'date_modified', c.date_modified,
							'active', c.active,
							'user', CASE
								WHEN cu.id IS NOT NULL THEN jsonb_build_object(
									'id', cu.id,
									'first_name', cu.first_name,
									'last_name', cu.last_name,
									'email_address', cu.email_address,
									'date_created', cu.date_created,
									'date_modified', cu.date_modified,
									'active', cu.active
								)
								ELSE NULL
							END
						)
					) ORDER BY c.id ASC
				) FILTER (WHERE c.id IS NOT NULL) as delegate_clients
			FROM delegate_client dc
			INNER JOIN client c ON dc.client_id = c.id
			LEFT JOIN "user" cu ON c.user_id = cu.id
			WHERE dc.active = true AND c.active = true
			GROUP BY dc.delegate_id
		),

		-- CTE 2: Apply search filter
		search_filtered AS (
			SELECT d.*
			FROM delegate d
			LEFT JOIN "user" u ON d.user_id = u.id
			WHERE d.active = true
				AND ($1::text = '' OR
					u.first_name ILIKE $1 OR
					u.last_name ILIKE $1 OR
					u.email_address ILIKE $1)
		),

		-- CTE 3: Join with user and delegate_clients and prepare for sorting
		enriched AS (
			SELECT
				sf.id,
				sf.user_id,
				sf.active,
				sf.date_created,
				sf.date_modified,
				CASE
					WHEN u.id IS NOT NULL THEN jsonb_build_object(
						'id', u.id,
						'first_name', u.first_name,
						'last_name', u.last_name,
						'email_address', u.email_address,
						'date_created', u.date_created,
						'date_modified', u.date_modified,
						'active', u.active
					)
					ELSE NULL
				END as user
				COALESCE(dca.delegate_clients, ARRAY[]::jsonb[]) as delegate_clients
			FROM search_filtered sf
			LEFT JOIN "user" u ON sf.user_id = u.id
			LEFT JOIN delegate_clients_agg dca ON sf.id = dca.delegate_id
		),

		-- CTE 4: Apply sorting
		sorted AS (
			SELECT * FROM enriched
			ORDER BY
				CASE WHEN $4 = 'user_id' AND $5 = 'ASC' THEN user_id END ASC,
				CASE WHEN $4 = 'user_id' AND $5 = 'DESC' THEN user_id END DESC,
				CASE WHEN ($4 = 'date_created' OR $4 = '') AND $5 = 'DESC' THEN date_created END DESC,
				CASE WHEN $4 = 'date_created' AND $5 = 'ASC' THEN date_created END ASC
		),

		-- CTE 5: Calculate total count for pagination
		total_count AS (
			SELECT count(*) as total FROM sorted
		)

		-- Final SELECT with pagination
		SELECT
			s.id,
			s.user_id,
			s.active,
			s.date_created,
			s.date_modified,
			s.user,
			s.delegate_clients,
			tc.total as _total_count
		FROM sorted s
		CROSS JOIN total_count tc
		LIMIT $2 OFFSET $3
	`

	// Get DB connection from dbOps interface
	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	// Execute query
	rows, err := db.GetDB().QueryContext(ctx, query,
		searchQuery,   // $1
		limit,         // $2
		offset,        // $3
		sortField,     // $4
		sortDirection, // $5
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetDelegateListPageData query: %w", err)
	}
	defer rows.Close()

	var delegates []*delegatepb.Delegate
	var totalCount int32

	for rows.Next() {
		var (
			id                  string
			userId              string
			active              bool
			dateCreated         sql.NullInt64
			dateCreatedString   sql.NullString
			dateModified        sql.NullInt64
			dateModifiedString  sql.NullString
			userJSON            []byte
			delegateClientsJSON []byte
			rowTotalCount       int32
		)

		err := rows.Scan(
			&id,
			&userId,
			&active,
			&dateCreated,
			&dateModified,
			&userJSON,
			&delegateClientsJSON,
			&rowTotalCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan delegate row: %w", err)
		}

		totalCount = rowTotalCount

		// Build delegate message
		delegate := &delegatepb.Delegate{
			Id:     id,
			UserId: userId,
			Active: active,
		}

		if dateCreated.Valid {
			delegate.DateCreated = &dateCreated.Int64
		}
		if dateCreatedString.Valid {
			delegate.DateCreatedString = &dateCreatedString.String
		}
		if dateModified.Valid {
			delegate.DateModified = &dateModified.Int64
		}
		if dateModifiedString.Valid {
			delegate.DateModifiedString = &dateModifiedString.String
		}

		// Parse user JSON
		if len(userJSON) > 0 && string(userJSON) != "null" {
			var userData map[string]any
			if err := json.Unmarshal(userJSON, &userData); err == nil {
				userDataJSON, _ := json.Marshal(userData)
				var user userpb.User
				if err := protojson.Unmarshal(userDataJSON, &user); err == nil {
					delegate.User = &user
				}
			}
		}

		// Parse delegate_clients JSON array
		if len(delegateClientsJSON) > 0 {
			var delegateClients []map[string]any
			if err := json.Unmarshal(delegateClientsJSON, &delegateClients); err == nil {
				// Convert to protobuf DelegateClient messages
				for _, dcData := range delegateClients {
					dcJSON, _ := json.Marshal(dcData)
					var delegateClient delegateclientpb.DelegateClient
					if err := protojson.Unmarshal(dcJSON, &delegateClient); err == nil {
						delegate.DelegateClients = append(delegate.DelegateClients, &delegateClient)
					}
				}
			}
		}

		delegates = append(delegates, delegate)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating delegate rows: %w", err)
	}

	// Build pagination response
	totalPages := (totalCount + limit - 1) / limit
	hasNext := page < totalPages
	hasPrev := page > 1

	paginationResponse := &commonpb.PaginationResponse{
		TotalItems:  totalCount,
		CurrentPage: &page,
		TotalPages:  &totalPages,
		HasNext:     hasNext,
		HasPrev:     hasPrev,
	}

	return &delegatepb.GetDelegateListPageDataResponse{
		Success:      true,
		DelegateList: delegates,
		Pagination:   paginationResponse,
	}, nil
}

// GetDelegateItemPageData retrieves a single delegate with all related user and client data expanded
// This method uses CTEs (Common Table Expressions) to load all related data in a single query
// TODO: Add unit tests for GetDelegateItemPageData
func (r *PostgresDelegateRepository) GetDelegateItemPageData(ctx context.Context, req *delegatepb.GetDelegateItemPageDataRequest) (*delegatepb.GetDelegateItemPageDataResponse, error) {
	if req.DelegateId == "" {
		return nil, fmt.Errorf("delegate ID is required")
	}

	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_delegate_id ON delegate(id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_delegate_user_id ON delegate(user_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_delegate_client_delegate_id ON delegate_client(delegate_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_delegate_client_client_id ON delegate_client(client_id);

	// Build CTE query to fetch delegate with all related data
	query := `
		WITH
		-- CTE 1: Aggregate delegate_client relationships with client and user details
		delegate_clients_agg AS (
			SELECT
				dc.delegate_id,
				array_agg(
					DISTINCT jsonb_build_object(
						'id', dc.id,
						'delegate_id', dc.delegate_id,
						'client_id', dc.client_id,
						'date_created', dc.date_created,
						'date_modified', dc.date_modified,
						'active', dc.active,
						'client', jsonb_build_object(
							'id', c.id,
							'user_id', c.user_id,
							'date_created', c.date_created,
							'date_modified', c.date_modified,
							'active', c.active,
							'user', CASE
								WHEN cu.id IS NOT NULL THEN jsonb_build_object(
									'id', cu.id,
									'first_name', cu.first_name,
									'last_name', cu.last_name,
									'email_address', cu.email_address,
									'date_created', cu.date_created,
									'date_modified', cu.date_modified,
									'active', cu.active
								)
								ELSE NULL
							END
						)
					) ORDER BY c.id ASC
				) FILTER (WHERE c.id IS NOT NULL) as delegate_clients
			FROM delegate_client dc
			INNER JOIN client c ON dc.client_id = c.id
			LEFT JOIN "user" cu ON c.user_id = cu.id
			WHERE dc.delegate_id = $1 AND dc.active = true AND c.active = true
			GROUP BY dc.delegate_id
		)

		-- Final SELECT with all related data
		SELECT
			d.id,
			d.user_id,
			d.active,
			d.date_created,
			d.date_modified,
			CASE
				WHEN u.id IS NOT NULL THEN jsonb_build_object(
					'id', u.id,
					'first_name', u.first_name,
					'last_name', u.last_name,
					'email_address', u.email_address,
					'date_created', u.date_created,
					'date_modified', u.date_modified,
					'active', u.active
				)
				ELSE NULL
			END as user
			COALESCE(dca.delegate_clients, ARRAY[]::jsonb[]) as delegate_clients
		FROM delegate d
		LEFT JOIN "user" u ON d.user_id = u.id
		LEFT JOIN delegate_clients_agg dca ON d.id = dca.delegate_id
		WHERE d.id = $1 AND d.active = true
	`

	// Get DB connection from dbOps interface
	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	// Execute query
	var (
		id                  string
		userId              string
		active              bool
		dateCreated         sql.NullInt64
		dateCreatedString   sql.NullString
		dateModified        sql.NullInt64
		dateModifiedString  sql.NullString
		userJSON            []byte
		delegateClientsJSON []byte
	)

	err := db.GetDB().QueryRowContext(ctx, query, req.DelegateId).Scan(
		&id,
		&userId,
		&active,
		&dateCreated,
		&dateModified,
		&userJSON,
		&delegateClientsJSON,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("delegate not found with ID: %s", req.DelegateId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetDelegateItemPageData query: %w", err)
	}

	// Build delegate message
	delegate := &delegatepb.Delegate{
		Id:     id,
		UserId: userId,
		Active: active,
	}

	if dateCreated.Valid {
		delegate.DateCreated = &dateCreated.Int64
	}
	if dateCreatedString.Valid {
		delegate.DateCreatedString = &dateCreatedString.String
	}
	if dateModified.Valid {
		delegate.DateModified = &dateModified.Int64
	}
	if dateModifiedString.Valid {
		delegate.DateModifiedString = &dateModifiedString.String
	}

	// Parse user JSON
	if len(userJSON) > 0 && string(userJSON) != "null" {
		var userData map[string]any
		if err := json.Unmarshal(userJSON, &userData); err == nil {
			userDataJSON, _ := json.Marshal(userData)
			var user userpb.User
			if err := protojson.Unmarshal(userDataJSON, &user); err == nil {
				delegate.User = &user
			}
		}
	}

	// Parse delegate_clients JSON array
	if len(delegateClientsJSON) > 0 {
		var delegateClients []map[string]any
		if err := json.Unmarshal(delegateClientsJSON, &delegateClients); err == nil {
			// Convert to protobuf DelegateClient messages
			for _, dcData := range delegateClients {
				dcJSON, _ := json.Marshal(dcData)
				var delegateClient delegateclientpb.DelegateClient
				if err := protojson.Unmarshal(dcJSON, &delegateClient); err == nil {
					delegate.DelegateClients = append(delegate.DelegateClients, &delegateClient)
				}
			}
		}
	}

	return &delegatepb.GetDelegateItemPageDataResponse{
		Success:  true,
		Delegate: delegate,
	}, nil
}

// NewDelegateRepository creates a new PostgreSQL delegate repository (old-style constructor)
func NewDelegateRepository(db *sql.DB, tableName string) delegatepb.DelegateDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresDelegateRepository(dbOps, tableName)
}
