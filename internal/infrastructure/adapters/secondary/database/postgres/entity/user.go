//go:build postgres

package entity

import (
	"context"
	"database/sql"
	"time"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	userpb "leapfor.xyz/esqyma/golang/v1/domain/entity/user"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "user", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres user repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresUserRepository(dbOps, tableName), nil
	})
}

// PostgresUserRepository implements user CRUD operations using PostgreSQL
type PostgresUserRepository struct {
	userpb.UnimplementedUserDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresUserRepository creates a new PostgreSQL user repository
func NewPostgresUserRepository(dbOps interfaces.DatabaseOperation, tableName string) userpb.UserDomainServiceServer {
	if tableName == "" {
		tableName = "user" // default fallback
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresUserRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateUser creates a new user using common PostgreSQL operations
func (r *PostgresUserRepository) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("user data is required")
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
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	user := &userpb.User{}
	if err := protojson.Unmarshal(resultJSON, user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &userpb.CreateUserResponse{
		Data: []*userpb.User{user},
	}, nil
}

// ReadUser retrieves a user using common PostgreSQL operations
func (r *PostgresUserRepository) ReadUser(ctx context.Context, req *userpb.ReadUserRequest) (*userpb.ReadUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read user: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	user := &userpb.User{}
	if err := protojson.Unmarshal(resultJSON, user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &userpb.ReadUserResponse{
		Data: []*userpb.User{user},
	}, nil
}

// UpdateUser updates a user using common PostgreSQL operations
func (r *PostgresUserRepository) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("user ID is required")
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
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	user := &userpb.User{}
	if err := protojson.Unmarshal(resultJSON, user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &userpb.UpdateUserResponse{
		Data: []*userpb.User{user},
	}, nil
}

// DeleteUser deletes a user using common PostgreSQL operations
func (r *PostgresUserRepository) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete user: %w", err)
	}

	return &userpb.DeleteUserResponse{
		Success: true,
	}, nil
}

// ListUsers lists users using common PostgreSQL operations
func (r *PostgresUserRepository) ListUsers(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var users []*userpb.User
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		user := &userpb.User{}
		if err := protojson.Unmarshal(resultJSON, user); err != nil {
			// Log error and continue with next item
			continue
		}
		users = append(users, user)
	}

	return &userpb.ListUsersResponse{
		Data: users,
	}, nil
}

func (r *PostgresUserRepository) GetUserListPageData(ctx context.Context, req *userpb.GetUserListPageDataRequest) (*userpb.GetUserListPageDataResponse, error) {
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
	query := `WITH enriched AS (SELECT id, first_name, last_name, email_address, active, date_created, date_modified FROM "user" WHERE active = true AND ($1::text IS NULL OR $1::text = '' OR first_name ILIKE $1 OR last_name ILIKE $1 OR email_address ILIKE $1)), counted AS (SELECT COUNT(*) as total FROM enriched) SELECT e.*, c.total FROM enriched e, counted c ORDER BY ` + sortField + ` ` + sortOrder + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var users []*userpb.User
	var totalCount int64
	for rows.Next() {
		var id, firstName, lastName, emailAddress string
		var active bool
		var dateCreated, dateModified time.Time
		var total int64
		if err := rows.Scan(&id, &firstName, &lastName, &emailAddress, &active, &dateCreated, &dateModified, &total); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		totalCount = total
		user := &userpb.User{Id: id, FirstName: firstName, LastName: lastName, EmailAddress: emailAddress, Active: active}
		if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		user.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		user.DateCreatedString = &dcStr
	}
		if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		user.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		user.DateModifiedString = &dmStr
	}
		users = append(users, user)
	}
	totalPages := int32((totalCount + int64(limit) - 1) / int64(limit))
	return &userpb.GetUserListPageDataResponse{UserList: users, Pagination: &commonpb.PaginationResponse{TotalItems: int32(totalCount), CurrentPage: &page, TotalPages: &totalPages, HasNext: page < totalPages, HasPrev: page > 1}, Success: true}, nil
}

func (r *PostgresUserRepository) GetUserItemPageData(ctx context.Context, req *userpb.GetUserItemPageDataRequest) (*userpb.GetUserItemPageDataResponse, error) {
	if req == nil || req.UserId == "" {
		return nil, fmt.Errorf("user ID required")
	}
	query := `SELECT id, first_name, last_name, email_address, active, date_created, date_modified FROM "user" WHERE id = $1 AND active = true`
	row := r.db.QueryRowContext(ctx, query, req.UserId)
	var id, firstName, lastName, emailAddress string
	var active bool
	var dateCreated, dateModified time.Time
	if err := row.Scan(&id, &firstName, &lastName, &emailAddress, &active, &dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	user := &userpb.User{Id: id, FirstName: firstName, LastName: lastName, EmailAddress: emailAddress, Active: active}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		user.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		user.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		user.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		user.DateModifiedString = &dmStr
	}
	return &userpb.GetUserItemPageDataResponse{User: user, Success: true}, nil
}


// NewUserRepository creates a new PostgreSQL user repository (old-style constructor)
func NewUserRepository(db *sql.DB, tableName string) userpb.UserDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresUserRepository(dbOps, tableName)
}
