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
	staffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "staff", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres staff repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresStaffRepository(dbOps, tableName), nil
	})
}

// PostgresStaffRepository implements staff CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_staff_user_id ON staff(user_id) - Foreign key relationship to user table
//   - CREATE INDEX idx_staff_active ON staff(active) - Filter active records
//   - CREATE INDEX idx_staff_date_created ON staff(date_created DESC) - Default sorting
//   - CREATE INDEX idx_user_first_name ON "user"(first_name) - Search performance on joined table
//   - CREATE INDEX idx_user_last_name ON "user"(last_name) - Search performance on joined table
//   - CREATE INDEX idx_user_email_address ON "user"(email_address) - Search performance on joined table
type PostgresStaffRepository struct {
	staffpb.UnimplementedStaffDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresStaffRepository creates a new PostgreSQL staff repository
func NewPostgresStaffRepository(dbOps interfaces.DatabaseOperation, tableName string) staffpb.StaffDomainServiceServer {
	if tableName == "" {
		tableName = "staff" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresStaffRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateStaff creates a new staff using common PostgreSQL operations
func (r *PostgresStaffRepository) CreateStaff(ctx context.Context, req *staffpb.CreateStaffRequest) (*staffpb.CreateStaffResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("staff data is required")
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
		return nil, fmt.Errorf("failed to create staff: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	staff := &staffpb.Staff{}
	if err := protojson.Unmarshal(resultJSON, staff); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &staffpb.CreateStaffResponse{
		Data: []*staffpb.Staff{staff},
	}, nil
}

// ReadStaff retrieves a staff using common PostgreSQL operations
func (r *PostgresStaffRepository) ReadStaff(ctx context.Context, req *staffpb.ReadStaffRequest) (*staffpb.ReadStaffResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("staff ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read staff: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	staff := &staffpb.Staff{}
	if err := protojson.Unmarshal(resultJSON, staff); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &staffpb.ReadStaffResponse{
		Data: []*staffpb.Staff{staff},
	}, nil
}

// UpdateStaff updates a staff using common PostgreSQL operations
func (r *PostgresStaffRepository) UpdateStaff(ctx context.Context, req *staffpb.UpdateStaffRequest) (*staffpb.UpdateStaffResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("staff ID is required")
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
		return nil, fmt.Errorf("failed to update staff: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	staff := &staffpb.Staff{}
	if err := protojson.Unmarshal(resultJSON, staff); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &staffpb.UpdateStaffResponse{
		Data: []*staffpb.Staff{staff},
	}, nil
}

// DeleteStaff deletes a staff using common PostgreSQL operations
func (r *PostgresStaffRepository) DeleteStaff(ctx context.Context, req *staffpb.DeleteStaffRequest) (*staffpb.DeleteStaffResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("staff ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete staff: %w", err)
	}

	return &staffpb.DeleteStaffResponse{
		Success: true,
	}, nil
}

// ListStaffs lists staffs using common PostgreSQL operations
func (r *PostgresStaffRepository) ListStaffs(ctx context.Context, req *staffpb.ListStaffsRequest) (*staffpb.ListStaffsResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list staffs: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var staffs []*staffpb.Staff
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		staff := &staffpb.Staff{}
		if err := protojson.Unmarshal(resultJSON, staff); err != nil {
			// Log error and continue with next item
			continue
		}
		staffs = append(staffs, staff)
	}

	return &staffpb.ListStaffsResponse{
		Data: staffs,
	}, nil
}

// GetStaffListPageData retrieves staffs with advanced filtering, sorting, searching, and pagination using CTE
func (r *PostgresStaffRepository) GetStaffListPageData(
	ctx context.Context,
	req *staffpb.GetStaffListPageDataRequest,
) (*staffpb.GetStaffListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get staff list page data request is required")
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
	query := `
		WITH enriched AS (
			SELECT
				s.id,
				s.user_id,
				s.active,
				s.date_created,
				s.date_modified,
				-- User fields (1:1 relationship)
				u.id as user_id_value,
				u.first_name as user_first_name,
				u.last_name as user_last_name,
				u.email_address as user_email_address,
				u.date_created as user_date_created,
				u.date_modified as user_date_modified,
				u.active as user_active
			FROM staff s
			LEFT JOIN "user" u ON s.user_id = u.id AND u.active = true
			WHERE s.active = true
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
		return nil, fmt.Errorf("failed to query staff list page data: %w", err)
	}
	defer rows.Close()

	var staffs []*staffpb.Staff
	var totalCount int64

	for rows.Next() {
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
			total                  int64
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
			return nil, fmt.Errorf("failed to scan staff row: %w", err)
		}

		totalCount = total

		staff := &staffpb.Staff{
			Id:     id,
			UserId: userId,
			Active: active,
		}

		// Handle nullable timestamp fields for staff

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		staff.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		staff.DateCreatedString = &dcStr
	}
		if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		staff.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		staff.DateModifiedString = &dmStr
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

			staff.User = user
		}

		staffs = append(staffs, staff)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating staff rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &staffpb.GetStaffListPageDataResponse{
		StaffList: staffs,
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

// GetStaffItemPageData retrieves a single staff with enhanced item page data using CTE
func (r *PostgresStaffRepository) GetStaffItemPageData(
	ctx context.Context,
	req *staffpb.GetStaffItemPageDataRequest,
) (*staffpb.GetStaffItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get staff item page data request is required")
	}
	if req.StaffId == "" {
		return nil, fmt.Errorf("staff ID is required")
	}

	// CTE Query - Single round-trip with enriched user data
	query := `
		WITH enriched AS (
			SELECT
				s.id,
				s.user_id,
				s.active,
				s.date_created,
				s.date_modified,
				-- User fields (1:1 relationship)
				u.id as user_id_value,
				u.first_name as user_first_name,
				u.last_name as user_last_name,
				u.email_address as user_email_address,
				u.date_created as user_date_created,
				u.date_modified as user_date_modified,
				u.active as user_active
			FROM staff s
			LEFT JOIN "user" u ON s.user_id = u.id AND u.active = true
			WHERE s.id = $1 AND s.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.StaffId)

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
		return nil, fmt.Errorf("staff with ID '%s' not found", req.StaffId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query staff item page data: %w", err)
	}

	staff := &staffpb.Staff{
		Id:     id,
		UserId: userId,
		Active: active,
	}

	// Handle nullable timestamp fields for staff

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		staff.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		staff.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		staff.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		staff.DateModifiedString = &dmStr
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

		staff.User = user
	}

	return &staffpb.GetStaffItemPageDataResponse{
		Staff:   staff,
		Success: true,
	}, nil
}


// NewStaffRepository creates a new PostgreSQL staff repository (old-style constructor)
func NewStaffRepository(db *sql.DB, tableName string) staffpb.StaffDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresStaffRepository(dbOps, tableName)
}
