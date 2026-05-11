//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.User, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres user repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresUserRepository(dbOps, tableName), nil
	})
}

// PostgresUserRepository implements user CRUD operations using PostgreSQL
type PostgresUserRepository struct {
	userpb.UnimplementedUserDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresUserRepository creates a new PostgreSQL user repository
func NewPostgresUserRepository(dbOps interfaces.DatabaseOperation, tableName string) userpb.UserDomainServiceServer {
	if tableName == "" {
		tableName = "user" // default fallback
	}

	return &PostgresUserRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateUser creates a new user using common PostgreSQL operations
func (r *PostgresUserRepository) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("user data is required")
	}

	// Emit unpopulated fields so false booleans are preserved instead of
	// disappearing from the JSON payload and becoming NULL on insert/update.
	jsonData, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(req.Data)
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
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, user); err != nil {
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
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &userpb.ReadUserResponse{
		Data:    []*userpb.User{user},
		Success: true,
	}, nil
}

// UpdateUser updates a user using common PostgreSQL operations
func (r *PostgresUserRepository) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// Emit unpopulated fields so false booleans are preserved instead of
	// disappearing from the JSON payload and becoming NULL on insert/update.
	jsonData, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(req.Data)
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
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, user); err != nil {
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

var userSortableSQLCols = []string{
	"id", "active", "first_name", "last_name", "email_address",
	"mobile_number", "timezone", "date_created", "date_modified",
}

var userSortSpec = espynahttp.SortSpec{AllowedCols: userSortableSQLCols}

// ListUsers lists users using common PostgreSQL operations
func (r *PostgresUserRepository) ListUsers(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	if err := espynahttp.ValidateSortColumns(userSortSpec, req.GetSort(), "user"); err != nil {
		return nil, err
	}

	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filters
		params.Search = req.Search
		params.Sort = req.Sort
		params.Pagination = req.Pagination
	}

	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
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
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, user); err != nil {
			// Log error and continue with next item
			continue
		}
		users = append(users, user)
	}

	return &userpb.ListUsersResponse{
		Data: users,
	}, nil
}

// userSortAllowlist maps external sort field names to safe SQL column references.
var userSortAllowlist = map[string]string{
	"first_name":    "first_name",
	"last_name":     "last_name",
	"email_address": "email_address",
	"date_created":  "date_created",
	"date_modified": "date_modified",
}

func (r *PostgresUserRepository) GetUserListPageData(ctx context.Context, req *userpb.GetUserListPageDataRequest) (*userpb.GetUserListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}

	// Default pagination values
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

	// Allowlist-validated sort
	sortCol := "date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		f := req.Sort.Fields[0]
		if col, ok := userSortAllowlist[f.Field]; ok {
			sortCol = col
		}
		if f.Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// Build filter/search WHERE clauses starting at $1
	searchFields := []string{"first_name", "last_name", "email_address"}
	filterClauses, filterArgs, nextIdx := postgresCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 1)

	whereSQL := ""
	if len(filterClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(filterClauses, " AND ")
	}

	limitIdx := nextIdx
	offsetIdx := nextIdx + 1
	filterArgs = append(filterArgs, limit, offset)

	query := fmt.Sprintf(`
		SELECT
			id, first_name, last_name, email_address, active, date_created, date_modified, timezone,
			COUNT(*) OVER() AS total_count
		FROM "user"
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereSQL, sortCol, sortOrder, limitIdx, offsetIdx)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, filterArgs...)
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
		var timezone sql.NullString
		var total int64
		if err := rows.Scan(&id, &firstName, &lastName, &emailAddress, &active, &dateCreated, &dateModified, &timezone, &total); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		totalCount = total
		user := &userpb.User{Id: id, FirstName: firstName, LastName: lastName, EmailAddress: emailAddress, Active: active}
		if timezone.Valid {
			tz := timezone.String
			user.Timezone = &tz
		}
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	return &userpb.GetUserListPageDataResponse{
		UserList: users,
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

func (r *PostgresUserRepository) GetUserItemPageData(ctx context.Context, req *userpb.GetUserItemPageDataRequest) (*userpb.GetUserItemPageDataResponse, error) {
	if req == nil || req.UserId == "" {
		return nil, fmt.Errorf("user ID required")
	}
	query := `SELECT id, first_name, last_name, email_address, active, date_created, date_modified, timezone FROM "user" WHERE id = $1`
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.UserId)
	var id, firstName, lastName, emailAddress string
	var active bool
	var dateCreated, dateModified time.Time
	var timezone sql.NullString
	if err := row.Scan(&id, &firstName, &lastName, &emailAddress, &active, &dateCreated, &dateModified, &timezone); err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	user := &userpb.User{Id: id, FirstName: firstName, LastName: lastName, EmailAddress: emailAddress, Active: active}
	if timezone.Valid {
		tz := timezone.String
		user.Timezone = &tz
	}
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
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresUserRepository(dbOps, tableName)
}