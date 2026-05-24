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
	espynactx "github.com/erniealice/espyna-golang/shared/context"
	principaltypepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/principal_type"
	sessionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/session"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Session, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres session repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSessionRepository(dbOps, tableName), nil
	})
}

// PostgresSessionRepository implements session CRUD operations using PostgreSQL.
//
// In addition to the SessionDomainServiceServer interface (Read/Create/Update/
// Delete/List), this adapter exposes SwitchPrincipal — the transactional
// session-rotation primitive that owns the in-tx coordination across the
// session table, the binding tables (workspace_user / *_portal_grant /
// delegate / delegate_*), and the audit_trail.audit_entry row insert.
// SwitchPrincipal was migrated FROM
// apps/service-admin/internal/composition/principal_switch.go (which violated
// the no-direct-sql-rule per docs/wiki/articles/no-direct-sql-rule.md §"Never
// in") INTO this adapter (where direct SQL is the legitimate hexagonal layer).
// See docs/plan/20260524-principal-switch-typed-stack/ Phase 2.
type PostgresSessionRepository struct {
	sessionpb.UnimplementedSessionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresSessionRepository creates a new PostgreSQL session repository.
//
// The repository captures a direct *sql.DB handle (via the dbOps' GetDB shim)
// because SwitchPrincipal owns its own transaction lifecycle (BeginTx →
// lockTargetBinding → session UPDATE/INSERT → audit insert → Commit). The
// CRUD methods continue to flow through dbOps so they pick up the workspace-
// id injection and transaction-aware executor behavior.
func NewPostgresSessionRepository(dbOps interfaces.DatabaseOperation, tableName string) sessionpb.SessionDomainServiceServer {
	if tableName == "" {
		tableName = "session"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresSessionRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateSession creates a new session row.
func (r *PostgresSessionRepository) CreateSession(ctx context.Context, req *sessionpb.CreateSessionRequest) (*sessionpb.CreateSessionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("session data is required")
	}

	jsonData, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	session := &sessionpb.Session{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &sessionpb.CreateSessionResponse{
		Data:    []*sessionpb.Session{session},
		Success: true,
	}, nil
}

// ReadSession retrieves a session. It supports lookup by id (primary key) or
// by token (unique index) — whichever field is populated on req.Data is used.
func (r *PostgresSessionRepository) ReadSession(ctx context.Context, req *sessionpb.ReadSessionRequest) (*sessionpb.ReadSessionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("session lookup criteria is required")
	}

	var (
		query string
		arg   string
	)

	switch {
	case req.Data.Token != "":
		// Token lookup is the auth-bootstrap path — workspace_id may not yet be
		// in ctx, so the empty-fallback ($2='' bypasses) keeps middleware working.
		query = `
			SELECT
				id, user_id, token,
				workspace_user_id, workspace_id,
				expires_at, active,
				date_created, date_modified,
				principal_type, principal_id,
				acting_as_client_id, acting_as_supplier_id, acting_as_workspace_id
			FROM ` + r.tableName + `
			WHERE token = $1 AND active = true
			  AND ($2::text = '' OR workspace_id = $2::text)
		`
		arg = req.Data.Token
	case req.Data.Id != "":
		query = `
			SELECT
				id, user_id, token,
				workspace_user_id, workspace_id,
				expires_at, active,
				date_created, date_modified,
				principal_type, principal_id,
				acting_as_client_id, acting_as_supplier_id, acting_as_workspace_id
			FROM ` + r.tableName + `
			WHERE id = $1 AND active = true
			  AND ($2::text = '' OR workspace_id = $2::text)
		`
		arg = req.Data.Id
	default:
		return nil, fmt.Errorf("session id or token is required")
	}

	wsID := espynactx.ExtractWorkspaceIDFromContext(ctx)
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, arg, wsID)

	session, err := scanSession(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read session: %w", err)
	}

	return &sessionpb.ReadSessionResponse{
		Data:    []*sessionpb.Session{session},
		Success: true,
	}, nil
}

// UpdateSession updates an existing session row.
func (r *PostgresSessionRepository) UpdateSession(ctx context.Context, req *sessionpb.UpdateSessionRequest) (*sessionpb.UpdateSessionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("session ID is required")
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
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	session := &sessionpb.Session{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &sessionpb.UpdateSessionResponse{
		Data:    []*sessionpb.Session{session},
		Success: true,
	}, nil
}

// DeleteSession soft-deletes a session (sets active = false).
func (r *PostgresSessionRepository) DeleteSession(ctx context.Context, req *sessionpb.DeleteSessionRequest) (*sessionpb.DeleteSessionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("session ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete session: %w", err)
	}

	return &sessionpb.DeleteSessionResponse{Success: true}, nil
}

// ListSessions returns sessions matching the optional filters.
func (r *PostgresSessionRepository) ListSessions(ctx context.Context, req *sessionpb.ListSessionsRequest) (*sessionpb.ListSessionsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	var sessions []*sessionpb.Session
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}

		session := &sessionpb.Session{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, session); err != nil {
			continue
		}
		sessions = append(sessions, session)
	}

	return &sessionpb.ListSessionsResponse{
		Data:    sessions,
		Success: true,
	}, nil
}

// scanSession reads a single session row from a QueryRowContext result.
// date_created and date_modified are stored as BIGINT unix-ms in the DB.
func scanSession(row *sql.Row) (*sessionpb.Session, error) {
	var (
		id                  string
		userID              string
		token               string
		workspaceUserID     *string
		workspaceID         *string
		expiresAt           int64
		active              bool
		dateCreated         *int64
		dateModified        *int64
		principalType       *int32
		principalID         *string
		actingAsClientID    *string
		actingAsSupplierID  *string
		actingAsWorkspaceID *string
	)

	err := row.Scan(
		&id,
		&userID,
		&token,
		&workspaceUserID,
		&workspaceID,
		&expiresAt,
		&active,
		&dateCreated,
		&dateModified,
		&principalType,
		&principalID,
		&actingAsClientID,
		&actingAsSupplierID,
		&actingAsWorkspaceID,
	)
	if err != nil {
		return nil, err
	}

	s := &sessionpb.Session{
		Id:                  id,
		UserId:              userID,
		Token:               token,
		WorkspaceUserId:     workspaceUserID,
		WorkspaceId:         workspaceID,
		ExpiresAt:           expiresAt,
		Active:              active,
		DateCreated:         dateCreated,
		DateModified:        dateModified,
		PrincipalId:         principalID,
		ActingAsClientId:    actingAsClientID,
		ActingAsSupplierId:  actingAsSupplierID,
		ActingAsWorkspaceId: actingAsWorkspaceID,
	}

	// Map the principal_type integer to the proto enum.
	if principalType != nil {
		pt := principaltypepb.PrincipalType(*principalType)
		s.PrincipalType = &pt
	}

	// Derive the human-readable string fields from the unix-ms timestamps.
	if dateCreated != nil && *dateCreated > 0 {
		t := time.UnixMilli(*dateCreated).UTC()
		str := t.Format(time.RFC3339)
		s.DateCreatedString = &str
	}
	if dateModified != nil && *dateModified > 0 {
		t := time.UnixMilli(*dateModified).UTC()
		str := t.Format(time.RFC3339)
		s.DateModifiedString = &str
	}

	return s, nil
}
