//go:build postgresql

package subscription

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
	subscriptionworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_workspace_user"
	"google.golang.org/protobuf/encoding/protojson"
)

// subscriptionWorkspaceUserSortableSQLCols is the sort-column whitelist that
// core.BuildOrderBy validates GetSubscriptionWorkspaceUserListPageData requests
// against (A2 fail-closed guard).
var subscriptionWorkspaceUserSortableSQLCols = []string{
	"is_owner",
	"date_created",
	"date_modified",
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SubscriptionWorkspaceUser, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres subscription_workspace_user repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSubscriptionWorkspaceUserRepository(dbOps, tableName), nil
	})
}

// PostgresSubscriptionWorkspaceUserRepository implements subscription_workspace_user
// CRUD operations using PostgreSQL. The junction models one workspace_user
// servicing a specific subscription. The composite-FK
// (client_id, workspace_user_id) -> client_workspace_user precondition and the
// client_id stamping from subscription.client_id are enforced in the use case
// layer; the DB composite FK is the backstop.
type PostgresSubscriptionWorkspaceUserRepository struct {
	subscriptionworkspaceuserpb.UnimplementedSubscriptionWorkspaceUserDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresSubscriptionWorkspaceUserRepository creates a new PostgreSQL subscription workspace user repository
func NewPostgresSubscriptionWorkspaceUserRepository(dbOps interfaces.DatabaseOperation, tableName string) subscriptionworkspaceuserpb.SubscriptionWorkspaceUserDomainServiceServer {
	if tableName == "" {
		tableName = "subscription_workspace_user"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresSubscriptionWorkspaceUserRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateSubscriptionWorkspaceUser creates a new subscription workspace user using common PostgreSQL operations
func (r *PostgresSubscriptionWorkspaceUserRepository) CreateSubscriptionWorkspaceUser(ctx context.Context, req *subscriptionworkspaceuserpb.CreateSubscriptionWorkspaceUserRequest) (*subscriptionworkspaceuserpb.CreateSubscriptionWorkspaceUserResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("subscription workspace user data is required")
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
		return nil, fmt.Errorf("failed to create subscription workspace user: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	subscriptionWorkspaceUser := &subscriptionworkspaceuserpb.SubscriptionWorkspaceUser{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, subscriptionWorkspaceUser); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &subscriptionworkspaceuserpb.CreateSubscriptionWorkspaceUserResponse{
		Data:    []*subscriptionworkspaceuserpb.SubscriptionWorkspaceUser{subscriptionWorkspaceUser},
		Success: true,
	}, nil
}

// ReadSubscriptionWorkspaceUser retrieves a subscription workspace user using common PostgreSQL operations
func (r *PostgresSubscriptionWorkspaceUserRepository) ReadSubscriptionWorkspaceUser(ctx context.Context, req *subscriptionworkspaceuserpb.ReadSubscriptionWorkspaceUserRequest) (*subscriptionworkspaceuserpb.ReadSubscriptionWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription workspace user ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription workspace user: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	subscriptionWorkspaceUser := &subscriptionworkspaceuserpb.SubscriptionWorkspaceUser{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, subscriptionWorkspaceUser); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &subscriptionworkspaceuserpb.ReadSubscriptionWorkspaceUserResponse{
		Data:    []*subscriptionworkspaceuserpb.SubscriptionWorkspaceUser{subscriptionWorkspaceUser},
		Success: true,
	}, nil
}

// UpdateSubscriptionWorkspaceUser updates a subscription workspace user using common PostgreSQL operations
func (r *PostgresSubscriptionWorkspaceUserRepository) UpdateSubscriptionWorkspaceUser(ctx context.Context, req *subscriptionworkspaceuserpb.UpdateSubscriptionWorkspaceUserRequest) (*subscriptionworkspaceuserpb.UpdateSubscriptionWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription workspace user ID is required")
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
		return nil, fmt.Errorf("failed to update subscription workspace user: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	subscriptionWorkspaceUser := &subscriptionworkspaceuserpb.SubscriptionWorkspaceUser{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, subscriptionWorkspaceUser); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &subscriptionworkspaceuserpb.UpdateSubscriptionWorkspaceUserResponse{
		Data:    []*subscriptionworkspaceuserpb.SubscriptionWorkspaceUser{subscriptionWorkspaceUser},
		Success: true,
	}, nil
}

// DeleteSubscriptionWorkspaceUser soft-deletes a subscription workspace user (sets active=false).
func (r *PostgresSubscriptionWorkspaceUserRepository) DeleteSubscriptionWorkspaceUser(ctx context.Context, req *subscriptionworkspaceuserpb.DeleteSubscriptionWorkspaceUserRequest) (*subscriptionworkspaceuserpb.DeleteSubscriptionWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription workspace user ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete subscription workspace user: %w", err)
	}

	return &subscriptionworkspaceuserpb.DeleteSubscriptionWorkspaceUserResponse{
		Success: true,
	}, nil
}

// ListSubscriptionWorkspaceUsers lists subscription workspace users using common
// PostgreSQL operations. Supports filters by subscription_id and by
// workspace_user_id (the "what do I service" query) via the request's
// common.FilterRequest.
func (r *PostgresSubscriptionWorkspaceUserRepository) ListSubscriptionWorkspaceUsers(ctx context.Context, req *subscriptionworkspaceuserpb.ListSubscriptionWorkspaceUsersRequest) (*subscriptionworkspaceuserpb.ListSubscriptionWorkspaceUsersResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscription workspace users: %w", err)
	}

	var subscriptionWorkspaceUsers []*subscriptionworkspaceuserpb.SubscriptionWorkspaceUser
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		subscriptionWorkspaceUser := &subscriptionworkspaceuserpb.SubscriptionWorkspaceUser{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, subscriptionWorkspaceUser); err != nil {
			continue
		}
		subscriptionWorkspaceUsers = append(subscriptionWorkspaceUsers, subscriptionWorkspaceUser)
	}

	return &subscriptionworkspaceuserpb.ListSubscriptionWorkspaceUsersResponse{
		Data:    subscriptionWorkspaceUsers,
		Success: true,
	}, nil
}

// GetSubscriptionWorkspaceUserListPageData retrieves paginated subscription workspace user list data.
// Tenancy scopes on the junction's own workspace_id column.
func (r *PostgresSubscriptionWorkspaceUserRepository) GetSubscriptionWorkspaceUserListPageData(ctx context.Context, req *subscriptionworkspaceuserpb.GetSubscriptionWorkspaceUserListPageDataRequest) (*subscriptionworkspaceuserpb.GetSubscriptionWorkspaceUserListPageDataResponse, error) {
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
	orderBy, err := postgresCore.BuildOrderBy(subscriptionWorkspaceUserSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, fmt.Errorf("invalid sort for subscription workspace user list: %w", err)
	}

	wsID := identity.Must(ctx).WorkspaceID
	query := `SELECT id, subscription_id, client_id, workspace_user_id, is_owner, active, date_created, date_modified
		FROM subscription_workspace_user
		WHERE active = true
			AND ($4::text = '' OR workspace_id = $4::text)
			AND ($1::text IS NULL OR $1::text = '' OR subscription_id ILIKE $1 OR workspace_user_id ILIKE $1) ` + orderBy + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset, wsID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var subscriptionWorkspaceUsers []*subscriptionworkspaceuserpb.SubscriptionWorkspaceUser
	for rows.Next() {
		swu, scanErr := scanSubscriptionWorkspaceUserRow(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("scan failed: %w", scanErr)
		}
		subscriptionWorkspaceUsers = append(subscriptionWorkspaceUsers, swu)
	}
	return &subscriptionworkspaceuserpb.GetSubscriptionWorkspaceUserListPageDataResponse{SubscriptionWorkspaceUserList: subscriptionWorkspaceUsers, Success: true}, nil
}

// GetSubscriptionWorkspaceUserItemPageData retrieves subscription workspace user item page data
func (r *PostgresSubscriptionWorkspaceUserRepository) GetSubscriptionWorkspaceUserItemPageData(ctx context.Context, req *subscriptionworkspaceuserpb.GetSubscriptionWorkspaceUserItemPageDataRequest) (*subscriptionworkspaceuserpb.GetSubscriptionWorkspaceUserItemPageDataResponse, error) {
	if req == nil || req.SubscriptionWorkspaceUserId == "" {
		return nil, fmt.Errorf("subscription workspace user ID required")
	}
	// Tenancy: scope on the junction's own workspace_id (IDOR defense — mirror the
	// list query). Empty wsID = service-to-service call → no scoping.
	wsID := identity.Must(ctx).WorkspaceID
	query := `SELECT id, subscription_id, client_id, workspace_user_id, is_owner, active, date_created, date_modified
		FROM subscription_workspace_user WHERE id = $1 AND active = true AND ($2::text = '' OR workspace_id = $2::text)`
	row := r.db.QueryRowContext(ctx, query, req.SubscriptionWorkspaceUserId, wsID)
	swu, err := scanSubscriptionWorkspaceUserRow(row.Scan)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("subscription workspace user not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return &subscriptionworkspaceuserpb.GetSubscriptionWorkspaceUserItemPageDataResponse{SubscriptionWorkspaceUser: swu, Success: true}, nil
}

// scanSubscriptionWorkspaceUserRow scans one subscription_workspace_user row
// (column order must match the SELECT lists above) into a proto message.
func scanSubscriptionWorkspaceUserRow(scan func(dest ...any) error) (*subscriptionworkspaceuserpb.SubscriptionWorkspaceUser, error) {
	var id, subscriptionId, clientId, workspaceUserId string
	var isOwner, active bool
	var dateCreated, dateModified time.Time
	if err := scan(&id, &subscriptionId, &clientId, &workspaceUserId, &isOwner, &active, &dateCreated, &dateModified); err != nil {
		return nil, err
	}
	swu := &subscriptionworkspaceuserpb.SubscriptionWorkspaceUser{
		Id:              id,
		SubscriptionId:  subscriptionId,
		ClientId:        clientId,
		WorkspaceUserId: workspaceUserId,
		IsOwner:         isOwner,
		Active:          active,
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		swu.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		swu.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		swu.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		swu.DateModifiedString = &dmStr
	}
	return swu, nil
}

// IsActiveServicer reports whether the acting principal (principalID = the login
// user_id, the same identity authcheck authorizes on) is an ACTIVE servicer of
// subscriptionID. It satisfies servicing.ServicingMembershipReader — the
// Q-SERVICING F-GATE PROJECT-scope port (CR-5). The join through workspace_user
// maps user_id → workspace_user_id; the optional workspace_id filter (from
// context) scopes the read to the request's tenant. Fail-closed: empty inputs
// return (false, nil); a query error returns (false, err), which CanServiceRow
// treats as deny.
func (r *PostgresSubscriptionWorkspaceUserRepository) IsActiveServicer(ctx context.Context, principalID string, subscriptionID string) (bool, error) {
	if r.db == nil || principalID == "" || subscriptionID == "" {
		return false, nil
	}
	wsID := identity.Must(ctx).WorkspaceID
	const q = `SELECT EXISTS (
		SELECT 1
		FROM subscription_workspace_user swu
		JOIN workspace_user wu ON wu.id = swu.workspace_user_id
		WHERE swu.subscription_id = $1
			AND swu.active = true
			AND wu.user_id = $2
			AND wu.active = true
			AND ($3::text = '' OR swu.workspace_id = $3::text))`
	var ok bool
	if err := r.db.QueryRowContext(ctx, q, subscriptionID, principalID, wsID).Scan(&ok); err != nil {
		return false, fmt.Errorf("servicer membership check failed: %w", err)
	}
	return ok, nil
}

// NewSubscriptionWorkspaceUserRepository creates a new PostgreSQL subscription_workspace_user repository (old-style constructor)
func NewSubscriptionWorkspaceUserRepository(db *sql.DB, tableName string) subscriptionworkspaceuserpb.SubscriptionWorkspaceUserDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresSubscriptionWorkspaceUserRepository(dbOps, tableName)
}
