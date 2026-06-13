//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	clientworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_workspace_user"
	"google.golang.org/protobuf/encoding/protojson"
)

// clientWorkspaceUserSortableSQLCols is the sort-column whitelist that
// core.BuildOrderBy validates GetClientWorkspaceUserListPageData requests against
// (A2 fail-closed guard).
var clientWorkspaceUserSortableSQLCols = []string{
	"is_owner",
	"date_created",
	"date_modified",
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ClientWorkspaceUser, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres client_workspace_user repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresClientWorkspaceUserRepository(dbOps, tableName), nil
	})
}

// PostgresClientWorkspaceUserRepository implements client_workspace_user CRUD
// operations using PostgreSQL. The junction models one workspace_user assigned to
// a client's account team. The single-owner-per-client invariant and the
// downstream composite-FK target are enforced in the use case layer (the DB
// partial unique is the backstop).
type PostgresClientWorkspaceUserRepository struct {
	clientworkspaceuserpb.UnimplementedClientWorkspaceUserDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresClientWorkspaceUserRepository creates a new PostgreSQL client workspace user repository
func NewPostgresClientWorkspaceUserRepository(dbOps interfaces.DatabaseOperation, tableName string) clientworkspaceuserpb.ClientWorkspaceUserDomainServiceServer {
	if tableName == "" {
		tableName = "client_workspace_user"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresClientWorkspaceUserRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateClientWorkspaceUser creates a new client workspace user using common PostgreSQL operations
func (r *PostgresClientWorkspaceUserRepository) CreateClientWorkspaceUser(ctx context.Context, req *clientworkspaceuserpb.CreateClientWorkspaceUserRequest) (*clientworkspaceuserpb.CreateClientWorkspaceUserResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("client workspace user data is required")
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
		return nil, fmt.Errorf("failed to create client workspace user: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	clientWorkspaceUser := &clientworkspaceuserpb.ClientWorkspaceUser{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, clientWorkspaceUser); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &clientworkspaceuserpb.CreateClientWorkspaceUserResponse{
		Data:    []*clientworkspaceuserpb.ClientWorkspaceUser{clientWorkspaceUser},
		Success: true,
	}, nil
}

// ReadClientWorkspaceUser retrieves a client workspace user using common PostgreSQL operations
func (r *PostgresClientWorkspaceUserRepository) ReadClientWorkspaceUser(ctx context.Context, req *clientworkspaceuserpb.ReadClientWorkspaceUserRequest) (*clientworkspaceuserpb.ReadClientWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client workspace user ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read client workspace user: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	clientWorkspaceUser := &clientworkspaceuserpb.ClientWorkspaceUser{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, clientWorkspaceUser); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &clientworkspaceuserpb.ReadClientWorkspaceUserResponse{
		Data:    []*clientworkspaceuserpb.ClientWorkspaceUser{clientWorkspaceUser},
		Success: true,
	}, nil
}

// UpdateClientWorkspaceUser updates a client workspace user using common PostgreSQL operations
func (r *PostgresClientWorkspaceUserRepository) UpdateClientWorkspaceUser(ctx context.Context, req *clientworkspaceuserpb.UpdateClientWorkspaceUserRequest) (*clientworkspaceuserpb.UpdateClientWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client workspace user ID is required")
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
		return nil, fmt.Errorf("failed to update client workspace user: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	clientWorkspaceUser := &clientworkspaceuserpb.ClientWorkspaceUser{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, clientWorkspaceUser); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &clientworkspaceuserpb.UpdateClientWorkspaceUserResponse{
		Data:    []*clientworkspaceuserpb.ClientWorkspaceUser{clientWorkspaceUser},
		Success: true,
	}, nil
}

// DeleteClientWorkspaceUser soft-deletes a client workspace user (sets active=false).
func (r *PostgresClientWorkspaceUserRepository) DeleteClientWorkspaceUser(ctx context.Context, req *clientworkspaceuserpb.DeleteClientWorkspaceUserRequest) (*clientworkspaceuserpb.DeleteClientWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client workspace user ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete client workspace user: %w", err)
	}

	return &clientworkspaceuserpb.DeleteClientWorkspaceUserResponse{
		Success: true,
	}, nil
}

// ListClientWorkspaceUsers lists client workspace users using common PostgreSQL
// operations. Supports filters by client_id and workspace_user_id via the
// request's common.FilterRequest.
func (r *PostgresClientWorkspaceUserRepository) ListClientWorkspaceUsers(ctx context.Context, req *clientworkspaceuserpb.ListClientWorkspaceUsersRequest) (*clientworkspaceuserpb.ListClientWorkspaceUsersResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list client workspace users: %w", err)
	}

	var clientWorkspaceUsers []*clientworkspaceuserpb.ClientWorkspaceUser
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		clientWorkspaceUser := &clientworkspaceuserpb.ClientWorkspaceUser{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, clientWorkspaceUser); err != nil {
			continue
		}
		clientWorkspaceUsers = append(clientWorkspaceUsers, clientWorkspaceUser)
	}

	return &clientworkspaceuserpb.ListClientWorkspaceUsersResponse{
		Data:    clientWorkspaceUsers,
		Success: true,
	}, nil
}

// GetClientWorkspaceUserListPageData retrieves paginated client workspace user list data.
// Tenancy scopes on the junction's own workspace_id column.
func (r *PostgresClientWorkspaceUserRepository) GetClientWorkspaceUserListPageData(ctx context.Context, req *clientworkspaceuserpb.GetClientWorkspaceUserListPageDataRequest) (*clientworkspaceuserpb.GetClientWorkspaceUserListPageDataResponse, error) {
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
	orderBy, err := postgresCore.BuildOrderBy(clientWorkspaceUserSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, fmt.Errorf("invalid sort for client workspace user list: %w", err)
	}

	wsID := identity.Must(ctx).WorkspaceID
	query := `SELECT id, client_id, workspace_user_id, is_owner, active, date_created, date_modified
		FROM client_workspace_user
		WHERE active = true
			AND ($4::text = '' OR workspace_id = $4::text)
			AND ($1::text IS NULL OR $1::text = '' OR client_id ILIKE $1 OR workspace_user_id ILIKE $1) ` + orderBy + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset, wsID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var clientWorkspaceUsers []*clientworkspaceuserpb.ClientWorkspaceUser
	for rows.Next() {
		cwu, scanErr := scanClientWorkspaceUserRow(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("scan failed: %w", scanErr)
		}
		clientWorkspaceUsers = append(clientWorkspaceUsers, cwu)
	}
	return &clientworkspaceuserpb.GetClientWorkspaceUserListPageDataResponse{ClientWorkspaceUserList: clientWorkspaceUsers, Success: true}, nil
}

// GetClientWorkspaceUserItemPageData retrieves client workspace user item page data
func (r *PostgresClientWorkspaceUserRepository) GetClientWorkspaceUserItemPageData(ctx context.Context, req *clientworkspaceuserpb.GetClientWorkspaceUserItemPageDataRequest) (*clientworkspaceuserpb.GetClientWorkspaceUserItemPageDataResponse, error) {
	if req == nil || req.ClientWorkspaceUserId == "" {
		return nil, fmt.Errorf("client workspace user ID required")
	}
	// Tenancy: scope on the junction's own workspace_id (IDOR defense — mirror the
	// list query). Empty wsID = service-to-service call → no scoping.
	wsID := identity.Must(ctx).WorkspaceID
	query := `SELECT id, client_id, workspace_user_id, is_owner, active, date_created, date_modified
		FROM client_workspace_user WHERE id = $1 AND active = true AND ($2::text = '' OR workspace_id = $2::text)`
	row := r.db.QueryRowContext(ctx, query, req.ClientWorkspaceUserId, wsID)
	cwu, err := scanClientWorkspaceUserRow(row.Scan)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("client workspace user not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return &clientworkspaceuserpb.GetClientWorkspaceUserItemPageDataResponse{ClientWorkspaceUser: cwu, Success: true}, nil
}

// scanClientWorkspaceUserRow scans one client_workspace_user row (column order
// must match the SELECT lists above) into a proto message.
func scanClientWorkspaceUserRow(scan func(dest ...any) error) (*clientworkspaceuserpb.ClientWorkspaceUser, error) {
	var id, clientId, workspaceUserId string
	var isOwner, active bool
	var dateCreated, dateModified time.Time
	if err := scan(&id, &clientId, &workspaceUserId, &isOwner, &active, &dateCreated, &dateModified); err != nil {
		return nil, err
	}
	cwu := &clientworkspaceuserpb.ClientWorkspaceUser{
		Id:              id,
		ClientId:        clientId,
		WorkspaceUserId: workspaceUserId,
		IsOwner:         isOwner,
		Active:          active,
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		cwu.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		cwu.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		cwu.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		cwu.DateModifiedString = &dmStr
	}
	return cwu, nil
}

// IsActiveAccountTeamMember reports whether the acting principal (principalID =
// the login user_id, the same identity authcheck authorizes on) is an ACTIVE
// account-team member of clientID. It satisfies servicing.AccountTeamMembershipReader
// — the Q-SERVICING F-GATE ACCOUNT-scope port (CR-5). The join through
// workspace_user maps user_id → workspace_user_id; the optional workspace_id
// filter (from context) scopes the read to the request's tenant. Fail-closed:
// empty inputs return (false, nil); a query error returns (false, err), which
// CanServiceRow treats as deny.
func (r *PostgresClientWorkspaceUserRepository) IsActiveAccountTeamMember(ctx context.Context, principalID string, clientID string) (bool, error) {
	if r.db == nil || principalID == "" || clientID == "" {
		return false, nil
	}
	wsID := identity.Must(ctx).WorkspaceID
	const q = `SELECT EXISTS (
		SELECT 1
		FROM client_workspace_user cwu
		JOIN workspace_user wu ON wu.id = cwu.workspace_user_id
		WHERE cwu.client_id = $1
			AND cwu.active = true
			AND wu.user_id = $2
			AND wu.active = true
			AND ($3::text = '' OR cwu.workspace_id = $3::text))`
	var ok bool
	if err := r.db.QueryRowContext(ctx, q, clientID, principalID, wsID).Scan(&ok); err != nil {
		return false, fmt.Errorf("account-team membership check failed: %w", err)
	}
	return ok, nil
}

// NewClientWorkspaceUserRepository creates a new PostgreSQL client_workspace_user repository (old-style constructor)
func NewClientWorkspaceUserRepository(db *sql.DB, tableName string) clientworkspaceuserpb.ClientWorkspaceUserDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresClientWorkspaceUserRepository(dbOps, tableName)
}
