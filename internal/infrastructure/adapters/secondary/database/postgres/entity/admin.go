//go:build postgresql

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
	adminpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/admin"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "admin", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres admin repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresAdminRepository(dbOps, tableName), nil
	})
}

// PostgresAdminRepository implements admin CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_admin_user_id ON admin(user_id) - Foreign key relationship to user table
//   - CREATE INDEX idx_admin_active ON admin(active) - Filter active records
//   - CREATE INDEX idx_admin_date_created ON admin(date_created DESC) - Default sorting
//   - CREATE INDEX idx_user_first_name ON "user"(first_name) - Search performance on joined table
//   - CREATE INDEX idx_user_last_name ON "user"(last_name) - Search performance on joined table
//   - CREATE INDEX idx_user_email_address ON "user"(email_address) - Search performance on joined table
//
// TODO: Add comprehensive tests for GetAdminListPageData:
//   - Test with no search query (list all active admins)
//   - Test with search query matching user first_name
//   - Test with search query matching user last_name
//   - Test with search query matching user email_address
//   - Test pagination (page 1, page 2, page size variations)
//   - Test sorting (by different fields, ASC and DESC)
//   - Test with no matching results
//   - Test with inactive admins (should be filtered out)
//   - Test with null user_id (LEFT JOIN behavior)
//   - Test with inactive user (should be filtered out via JOIN condition)
//
// TODO: Add comprehensive tests for GetAdminItemPageData:
//   - Test with valid admin ID (with associated user)
//   - Test with valid admin ID (without associated user - null user_id)
//   - Test with non-existent admin ID
//   - Test with inactive admin (should return not found)
//   - Test with admin having inactive user (user fields should be null)
//   - Test timestamp parsing for date_created and date_modified
//
type PostgresAdminRepository struct {
	adminpb.UnimplementedAdminDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresAdminRepository creates a new PostgreSQL admin repository
func NewPostgresAdminRepository(dbOps interfaces.DatabaseOperation, tableName string) adminpb.AdminDomainServiceServer {
	if tableName == "" {
		tableName = "admin" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresAdminRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateAdmin creates a new admin using common PostgreSQL operations
func (r *PostgresAdminRepository) CreateAdmin(ctx context.Context, req *adminpb.CreateAdminRequest) (*adminpb.CreateAdminResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("admin data is required")
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
		return nil, fmt.Errorf("failed to create admin: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	admin := &adminpb.Admin{}
	if err := protojson.Unmarshal(resultJSON, admin); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &adminpb.CreateAdminResponse{
		Data: []*adminpb.Admin{admin},
	}, nil
}

// ReadAdmin retrieves an admin using common PostgreSQL operations
func (r *PostgresAdminRepository) ReadAdmin(ctx context.Context, req *adminpb.ReadAdminRequest) (*adminpb.ReadAdminResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("admin ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read admin: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	admin := &adminpb.Admin{}
	if err := protojson.Unmarshal(resultJSON, admin); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &adminpb.ReadAdminResponse{
		Data: []*adminpb.Admin{admin},
	}, nil
}

// UpdateAdmin updates an admin using common PostgreSQL operations
func (r *PostgresAdminRepository) UpdateAdmin(ctx context.Context, req *adminpb.UpdateAdminRequest) (*adminpb.UpdateAdminResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("admin ID is required")
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
		return nil, fmt.Errorf("failed to update admin: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	admin := &adminpb.Admin{}
	if err := protojson.Unmarshal(resultJSON, admin); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &adminpb.UpdateAdminResponse{
		Data: []*adminpb.Admin{admin},
	}, nil
}

// DeleteAdmin deletes an admin using common PostgreSQL operations
func (r *PostgresAdminRepository) DeleteAdmin(ctx context.Context, req *adminpb.DeleteAdminRequest) (*adminpb.DeleteAdminResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("admin ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete admin: %w", err)
	}

	return &adminpb.DeleteAdminResponse{
		Success: true,
	}, nil
}

// ListAdmins lists admins using common PostgreSQL operations
func (r *PostgresAdminRepository) ListAdmins(ctx context.Context, req *adminpb.ListAdminsRequest) (*adminpb.ListAdminsResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list admins: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var admins []*adminpb.Admin
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		admin := &adminpb.Admin{}
		if err := protojson.Unmarshal(resultJSON, admin); err != nil {
			// Log error and continue with next item
			continue
		}
		admins = append(admins, admin)
	}

	return &adminpb.ListAdminsResponse{
		Data: admins,
	}, nil
}

// GetAdminListPageData retrieves admins with advanced filtering, sorting, searching, and pagination using CTE
func (r *PostgresAdminRepository) GetAdminListPageData(
	ctx context.Context,
	req *adminpb.GetAdminListPageDataRequest,
) (*adminpb.GetAdminListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get admin list page data request is required")
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
		if req.Sort.Fields[0].Direction == 1 { // ASC = 1
			sortOrder = "ASC"
		}
	}

	// CTE Query - Single round-trip with enriched user data
	// Performance Notes:
	// - INDEX RECOMMENDATION: Create index on admin.user_id (foreign key)
	// - INDEX RECOMMENDATION: Create index on user.first_name, user.last_name, user.email_address for search performance
	// - INDEX RECOMMENDATION: Create index on admin.active for filtering active records
	// - INDEX RECOMMENDATION: Create index on admin.date_created for default sorting
	query := `
		WITH enriched AS (
			SELECT
				a.id,
				a.user_id,
				a.active,
				a.date_created,
				a.date_modified,
				-- User fields (1:1 relationship)
				u.id as user_id_value,
				u.first_name as user_first_name,
				u.last_name as user_last_name,
				u.email_address as user_email_address,
				u.date_created as user_date_created,
				u.date_modified as user_date_modified,
				u.active as user_active
			FROM admin a
			LEFT JOIN "user" u ON a.user_id = u.id AND u.active = true
			WHERE a.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
				   u.first_name ILIKE $1 OR
				   u.last_name ILIKE $1 OR
				   u.email_address ILIKE $1)
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
		return nil, fmt.Errorf("failed to query admin list page data: %w", err)
	}
	defer rows.Close()

	var admins []*adminpb.Admin
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			userId             string
			active             bool
			dateCreated        time.Time
			dateModified       time.Time
			// User fields
			userIdValue              *string
			userFirstName            *string
			userLastName             *string
			userEmailAddress         *string
			userDateCreated          time.Time
			userDateModified         time.Time
			userActive               *bool
			total                    int64
		)

		err := rows.Scan(
			&id,
			&userId,
			&active,
			&dateCreated,
			&dateModified,
			&userIdValue,
			&userFirstName,
			&userLastName,
			&userEmailAddress,
			&userDateCreated,
			&userDateModified,
			&userActive,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan admin row: %w", err)
		}

		totalCount = total

		admin := &adminpb.Admin{
			Id:     id,
			UserId: userId,
			Active: active,
		}

		// Handle nullable timestamp fields for admin

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		admin.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		admin.DateCreatedString = &dcStr
	}
		if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		admin.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		admin.DateModifiedString = &dmStr
	}

		// Populate user data if available
		if userIdValue != nil {
			user := &userpb.User{
				Id:     *userIdValue,
				Active: userActive != nil && *userActive,
			}

			if userFirstName != nil {
				user.FirstName = *userFirstName
			}
			if userLastName != nil {
				user.LastName = *userLastName
			}
			if userEmailAddress != nil {
				user.EmailAddress = *userEmailAddress
			}

			// Parse user timestamps
			if !userDateCreated.IsZero() {
			ts := userDateCreated.UnixMilli()
			user.DateCreated = &ts
			udcStr := userDateCreated.Format(time.RFC3339)
			user.DateCreatedString = &udcStr
		}
			if !userDateModified.IsZero() {
			ts := userDateModified.UnixMilli()
			user.DateModified = &ts
			udmStr := userDateModified.Format(time.RFC3339)
			user.DateModifiedString = &udmStr
		}

			admin.User = user
		}

		admins = append(admins, admin)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating admin rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &adminpb.GetAdminListPageDataResponse{
		AdminList: admins,
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

// GetAdminItemPageData retrieves a single admin with enhanced item page data using CTE
func (r *PostgresAdminRepository) GetAdminItemPageData(
	ctx context.Context,
	req *adminpb.GetAdminItemPageDataRequest,
) (*adminpb.GetAdminItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get admin item page data request is required")
	}
	if req.AdminId == "" {
		return nil, fmt.Errorf("admin ID is required")
	}

	// CTE Query - Single round-trip with enriched user data
	query := `
		WITH enriched AS (
			SELECT
				a.id,
				a.user_id,
				a.active,
				a.date_created,
				a.date_modified,
				-- User fields (1:1 relationship)
				u.id as user_id_value,
				u.first_name as user_first_name,
				u.last_name as user_last_name,
				u.email_address as user_email_address,
				u.date_created as user_date_created,
				u.date_modified as user_date_modified,
				u.active as user_active
			FROM admin a
			LEFT JOIN "user" u ON a.user_id = u.id AND u.active = true
			WHERE a.id = $1 AND a.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.AdminId)

	var (
		id                 string
		userId             string
		active             bool
		dateCreated        time.Time
		dateModified       time.Time
		// User fields
		userIdValue            *string
		userFirstName          *string
		userLastName           *string
		userEmailAddress       *string
		userDateCreated        time.Time
		userDateModified       time.Time
		userActive             *bool
	)

	err := row.Scan(
		&id,
		&userId,
		&active,
		&dateCreated,
		&dateModified,
		&userIdValue,
		&userFirstName,
		&userLastName,
		&userEmailAddress,
		&userDateCreated,
		&userDateModified,
		&userActive,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("admin with ID '%s' not found", req.AdminId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query admin item page data: %w", err)
	}

	admin := &adminpb.Admin{
		Id:     id,
		UserId: userId,
		Active: active,
	}

	// Handle nullable timestamp fields for admin

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		admin.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		admin.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		admin.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		admin.DateModifiedString = &dmStr
	}

	// Populate user data if available
	if userIdValue != nil {
		user := &userpb.User{
			Id:     *userIdValue,
			Active: userActive != nil && *userActive,
		}

		if userFirstName != nil {
			user.FirstName = *userFirstName
		}
		if userLastName != nil {
			user.LastName = *userLastName
		}
		if userEmailAddress != nil {
			user.EmailAddress = *userEmailAddress
		}

		// Parse user timestamps
		if !userDateCreated.IsZero() {
			ts := userDateCreated.UnixMilli()
			user.DateCreated = &ts
			udcStr := userDateCreated.Format(time.RFC3339)
			user.DateCreatedString = &udcStr
		}
		if !userDateModified.IsZero() {
			ts := userDateModified.UnixMilli()
			user.DateModified = &ts
			udmStr := userDateModified.Format(time.RFC3339)
			user.DateModifiedString = &udmStr
		}

		admin.User = user
	}

	return &adminpb.GetAdminItemPageDataResponse{
		Admin:   admin,
		Success: true,
	}, nil
}


// NewAdminRepository creates a new PostgreSQL admin repository (old-style constructor)
func NewAdminRepository(db *sql.DB, tableName string) adminpb.AdminDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresAdminRepository(dbOps, tableName)
}
