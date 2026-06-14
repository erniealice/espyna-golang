//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	principaltypepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/principal_type"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

// PrincipalResolverAdapter handles principal resolution queries: enumerating
// a user's active bindings across grant tables (workspace_user,
// client_portal_grant, supplier_portal_grant, delegate_client,
// delegate_supplier).
//
// Migrated FROM apps/service-admin/internal/infrastructure/input/http/
// principal_loader.go (1,062 lines, 12 raw SQL queries) per the no-direct-sql
// rule (docs/wiki/articles/no-direct-sql-rule.md). The SQL is VERBATIM from
// the original — same joins, same predicates, same column aliases.
//
// This adapter also handles session principal lookup (reading principal_type,
// principal_id, acting_as_* from the session table by token), migrated FROM
// apps/service-admin/internal/composition/session_principal.go.
type PrincipalResolverAdapter struct {
	db *sql.DB
}

// NewPrincipalResolverAdapter creates a new adapter from a *sql.DB.
func NewPrincipalResolverAdapter(db *sql.DB) *PrincipalResolverAdapter {
	return &PrincipalResolverAdapter{db: db}
}

// ResolvePrincipals returns every active principal binding the given user has
// across all workspaces and all five grant tables. Returns an empty (non-nil)
// slice when no bindings exist.
//
// Query plan (per principal_loader.go §Resolve):
//  1. workspace_user (+ workspace_user_role) → OPERATOR_OWNER / OPERATOR_STAFF
//  2. client_portal_grant                    → CLIENT
//  3. supplier_portal_grant                  → SUPPLIER
//  4. delegate → delegate_client             → CLIENT_DELEGATE
//  5. delegate → delegate_supplier           → SUPPLIER_DELEGATE
func (a *PrincipalResolverAdapter) ResolvePrincipals(
	ctx context.Context,
	req *authpb.ResolvePrincipalsRequest,
) (*authpb.ResolvePrincipalsResponse, error) {
	if req == nil || strings.TrimSpace(req.GetUserId()) == "" {
		return &authpb.ResolvePrincipalsResponse{}, nil
	}
	userID := req.GetUserId()
	db := a.db

	out := make([]*authpb.Principal, 0, 4)

	// ─── 1. WorkspaceUser → Operator-Owner / Operator-Staff ─────────────────
	{
		const q = `
			SELECT
				wu.id,
				wu.workspace_id,
				COALESCE(w.name, '') AS workspace_name,
				COALESCE(BOOL_OR(wur.role_id = 'role-admin'), false) AS is_owner
			FROM workspace_user wu
			LEFT JOIN workspace w
				ON w.id = wu.workspace_id AND w.active = true
			LEFT JOIN workspace_user_role wur
				ON wur.workspace_user_id = wu.id AND wur.active = true
			WHERE wu.user_id = $1
				AND wu.active = true
			GROUP BY wu.id, wu.workspace_id, w.name
			ORDER BY workspace_name ASC, wu.id ASC
		`
		rows, err := db.QueryContext(ctx, q, userID)
		if err != nil {
			return nil, fmt.Errorf("principal_resolver: workspace_user query: %w", err)
		}
		for rows.Next() {
			var (
				wuID, wsID, wsName sql.NullString
				isOwner            sql.NullBool
			)
			if err := rows.Scan(&wuID, &wsID, &wsName, &isOwner); err != nil {
				rows.Close()
				return nil, fmt.Errorf("principal_resolver: workspace_user scan: %w", err)
			}
			ptype := principaltypepb.PrincipalType_PRINCIPAL_TYPE_OPERATOR_STAFF
			suffix := "Staff"
			if isOwner.Bool {
				ptype = principaltypepb.PrincipalType_PRINCIPAL_TYPE_OPERATOR_OWNER
				suffix = "Owner"
			}
			name := strings.TrimSpace(wsName.String)
			if name == "" {
				name = "Workspace"
			}
			out = append(out, &authpb.Principal{
				Type:        ptype,
				PrincipalId: wuID.String,
				WorkspaceId: wsID.String,
				DisplayName: fmt.Sprintf("%s · %s", name, suffix),
			})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("principal_resolver: workspace_user rows: %w", err)
		}
		rows.Close()
	}

	// ─── 2. ClientPortalGrant → CLIENT ──────────────────────────────────────
	{
		const q = `
			SELECT
				cpg.id,
				cpg.workspace_id,
				COALESCE(c.name, '') AS client_name
			FROM client_portal_grant cpg
			LEFT JOIN client c ON c.id = cpg.client_id AND c.active = true
			WHERE cpg.user_id = $1
				AND cpg.active = true
			ORDER BY client_name ASC, cpg.id ASC
		`
		rows, err := db.QueryContext(ctx, q, userID)
		if err != nil {
			return nil, fmt.Errorf("principal_resolver: client_portal_grant query: %w", err)
		}
		for rows.Next() {
			var grantID, wsID, name sql.NullString
			if err := rows.Scan(&grantID, &wsID, &name); err != nil {
				rows.Close()
				return nil, fmt.Errorf("principal_resolver: client_portal_grant scan: %w", err)
			}
			display := strings.TrimSpace(name.String)
			if display == "" {
				display = "Client"
			}
			out = append(out, &authpb.Principal{
				Type:        principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT,
				PrincipalId: grantID.String,
				WorkspaceId: wsID.String,
				DisplayName: fmt.Sprintf("%s · Client", display),
			})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("principal_resolver: client_portal_grant rows: %w", err)
		}
		rows.Close()
	}

	// ─── 3. SupplierPortalGrant → SUPPLIER ──────────────────────────────────
	{
		const q = `
			SELECT
				spg.id,
				spg.workspace_id,
				COALESCE(s.name, '') AS supplier_name
			FROM supplier_portal_grant spg
			LEFT JOIN supplier s ON s.id = spg.supplier_id AND s.active = true
			WHERE spg.user_id = $1
				AND spg.active = true
			ORDER BY supplier_name ASC, spg.id ASC
		`
		rows, err := db.QueryContext(ctx, q, userID)
		if err != nil {
			return nil, fmt.Errorf("principal_resolver: supplier_portal_grant query: %w", err)
		}
		for rows.Next() {
			var grantID, wsID, name sql.NullString
			if err := rows.Scan(&grantID, &wsID, &name); err != nil {
				rows.Close()
				return nil, fmt.Errorf("principal_resolver: supplier_portal_grant scan: %w", err)
			}
			display := strings.TrimSpace(name.String)
			if display == "" {
				display = "Supplier"
			}
			out = append(out, &authpb.Principal{
				Type:        principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER,
				PrincipalId: grantID.String,
				WorkspaceId: wsID.String,
				DisplayName: fmt.Sprintf("%s · Supplier", display),
			})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("principal_resolver: supplier_portal_grant rows: %w", err)
		}
		rows.Close()
	}

	// ─── 4. Delegate → DelegateClient → CLIENT_DELEGATE ─────────────────────
	if cdPrincipals, err := a.resolveDelegatePrincipals(
		ctx, db, userID, principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE,
	); err != nil {
		return nil, err
	} else {
		out = append(out, cdPrincipals...)
	}

	// ─── 5. Delegate → DelegateSupplier → SUPPLIER_DELEGATE ─────────────────
	if sdPrincipals, err := a.resolveDelegatePrincipals(
		ctx, db, userID, principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE,
	); err != nil {
		return nil, err
	} else {
		out = append(out, sdPrincipals...)
	}

	log.Printf("[principal_resolver] resolved %d principal(s) for user %s", len(out), userID)
	return &authpb.ResolvePrincipalsResponse{Principals: out}, nil
}

// resolveDelegatePrincipals enumerates the user's active delegate principals
// of a given kind (CLIENT_DELEGATE or SUPPLIER_DELEGATE) across ALL of the
// user's active delegate rows, emitting ONE Principal per delegate.id.
//
// SQL is VERBATIM from principal_loader.go's resolveDelegatePrincipals.
func (a *PrincipalResolverAdapter) resolveDelegatePrincipals(
	ctx context.Context,
	db *sql.DB,
	userID string,
	kind principaltypepb.PrincipalType,
) ([]*authpb.Principal, error) {
	var (
		q         string
		roleLabel string
	)
	switch kind {
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE:
		roleLabel = "Client Delegate"
		q = `
			SELECT
				d.id,
				dc.client_id,
				COALESCE(dc.workspace_id, c.workspace_id, '') AS workspace_id,
				COALESCE(c.name, '') AS display_name
			FROM delegate d
			JOIN delegate_client dc
				ON dc.delegate_id = d.id AND dc.active = true
			LEFT JOIN client c ON c.id = dc.client_id AND c.active = true
			WHERE d.user_id = $1
				AND d.active = true
			ORDER BY d.id, COALESCE(c.name, ''), dc.client_id
		`
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE:
		roleLabel = "Supplier Delegate"
		q = `
			SELECT
				d.id,
				ds.supplier_id,
				COALESCE(ds.workspace_id, s.workspace_id, '') AS workspace_id,
				COALESCE(s.name, '') AS display_name
			FROM delegate d
			JOIN delegate_supplier ds
				ON ds.delegate_id = d.id AND ds.active = true
			LEFT JOIN supplier s ON s.id = ds.supplier_id AND s.active = true
			WHERE d.user_id = $1
				AND d.active = true
			ORDER BY d.id, COALESCE(s.name, ''), ds.supplier_id
		`
	default:
		return nil, nil
	}

	rows, err := db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("principal_resolver: delegate (%s) query: %w", kind, err)
	}
	defer rows.Close()

	scanned := make([]delegateResolverRow, 0, 4)
	for rows.Next() {
		var dID, targetID, wsID, name sql.NullString
		if err := rows.Scan(&dID, &targetID, &wsID, &name); err != nil {
			return nil, fmt.Errorf("principal_resolver: delegate (%s) scan: %w", kind, err)
		}
		scanned = append(scanned, delegateResolverRow{
			DelegateID:  dID.String,
			TargetID:    targetID.String,
			WorkspaceID: strings.TrimSpace(wsID.String),
			DisplayName: strings.TrimSpace(name.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("principal_resolver: delegate (%s) rows: %w", kind, err)
	}
	return groupDelegateResolverRows(kind, roleLabel, scanned), nil
}

// delegateResolverRow is one scanned (delegate.id, target) row.
// Mirrors delegateTargetRow from principal_loader.go.
type delegateResolverRow struct {
	DelegateID  string
	TargetID    string
	WorkspaceID string
	DisplayName string
}

// groupDelegateResolverRows is the pure group-break that turns delegate join
// rows into one Principal per delegate.id. Mirrors groupDelegateRows from
// principal_loader.go but produces proto types.
func groupDelegateResolverRows(
	kind principaltypepb.PrincipalType,
	roleLabel string,
	rows []delegateResolverRow,
) []*authpb.Principal {
	fallbackName := "Client"
	if kind == principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE {
		fallbackName = "Supplier"
	}

	out := make([]*authpb.Principal, 0, 2)
	var (
		curDelegateID string
		curTargets    []*authpb.ActingAsTarget
	)
	flush := func() {
		if curDelegateID != "" && len(curTargets) > 0 {
			out = append(out, &authpb.Principal{
				Type:            kind,
				PrincipalId:     curDelegateID,
				WorkspaceId:     curTargets[0].WorkspaceId, // first target's workspace as default
				DisplayName:     formatResolverDelegateLabel(roleLabel, curTargets),
				ActingAsTargets: curTargets,
			})
		}
	}
	for _, row := range rows {
		if row.DelegateID != curDelegateID {
			flush()
			curDelegateID = row.DelegateID
			curTargets = nil
		}
		display := strings.TrimSpace(row.DisplayName)
		if display == "" {
			display = fallbackName
		}
		curTargets = append(curTargets, &authpb.ActingAsTarget{
			Id:          row.TargetID,
			WorkspaceId: strings.TrimSpace(row.WorkspaceID),
			DisplayName: display,
		})
	}
	flush() // emit the final delegate group
	return out
}

// formatResolverDelegateLabel mirrors formatDelegateLabel from principal_loader.go.
func formatResolverDelegateLabel(role string, targets []*authpb.ActingAsTarget) string {
	switch len(targets) {
	case 0:
		return role
	case 1:
		return fmt.Sprintf("%s · %s", targets[0].DisplayName, role)
	default:
		return fmt.Sprintf("%d targets · %s", len(targets), role)
	}
}

// EnumerateBindingsInWorkspace returns every active binding the user holds
// in the given workspace, across all five grant tables.
//
// SQL is VERBATIM from principal_loader.go's enumerateBindingsInWorkspace.
func (a *PrincipalResolverAdapter) EnumerateBindingsInWorkspace(
	ctx context.Context,
	req *authpb.EnumerateBindingsRequest,
) (*authpb.EnumerateBindingsResponse, error) {
	if req == nil {
		return &authpb.EnumerateBindingsResponse{}, nil
	}
	userID := strings.TrimSpace(req.GetUserId())
	workspaceID := strings.TrimSpace(req.GetWorkspaceId())
	if userID == "" || workspaceID == "" {
		return &authpb.EnumerateBindingsResponse{}, nil
	}
	db := a.db

	out := make([]*authpb.Principal, 0, 2)

	// 1. workspace_user (with role lookup for owner-vs-staff)
	{
		const q = `
			SELECT
				wu.id,
				COALESCE(w.name, '') AS workspace_name,
				COALESCE(BOOL_OR(wur.role_id = 'role-admin'), false) AS is_owner
			FROM workspace_user wu
			LEFT JOIN workspace w
				ON w.id = wu.workspace_id AND w.active = true
			LEFT JOIN workspace_user_role wur
				ON wur.workspace_user_id = wu.id AND wur.active = true
			WHERE wu.user_id = $1
				AND wu.workspace_id = $2
				AND wu.active = true
			GROUP BY wu.id, w.name
			ORDER BY wu.id
		`
		rows, err := db.QueryContext(ctx, q, userID, workspaceID)
		if err != nil {
			return nil, fmt.Errorf("principal_resolver: workspace_user lookup: %w", err)
		}
		for rows.Next() {
			var (
				wuID, wsName sql.NullString
				isOwner      sql.NullBool
			)
			if err := rows.Scan(&wuID, &wsName, &isOwner); err != nil {
				rows.Close()
				return nil, fmt.Errorf("principal_resolver: workspace_user scan: %w", err)
			}
			ptype := principaltypepb.PrincipalType_PRINCIPAL_TYPE_OPERATOR_STAFF
			suffix := "Staff"
			if isOwner.Bool {
				ptype = principaltypepb.PrincipalType_PRINCIPAL_TYPE_OPERATOR_OWNER
				suffix = "Owner"
			}
			name := strings.TrimSpace(wsName.String)
			if name == "" {
				name = "Workspace"
			}
			out = append(out, &authpb.Principal{
				Type:        ptype,
				PrincipalId: wuID.String,
				WorkspaceId: workspaceID,
				DisplayName: fmt.Sprintf("%s · %s", name, suffix),
			})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("principal_resolver: workspace_user rows: %w", err)
		}
		rows.Close()
	}

	// 2. client_portal_grant → CLIENT
	{
		const q = `
			SELECT cpg.id, COALESCE(c.name, '')
			FROM client_portal_grant cpg
			LEFT JOIN client c ON c.id = cpg.client_id AND c.active = true
			WHERE cpg.user_id = $1
				AND cpg.workspace_id = $2
				AND cpg.active = true
			ORDER BY COALESCE(c.name, ''), cpg.id
		`
		rows, err := db.QueryContext(ctx, q, userID, workspaceID)
		if err != nil {
			return nil, fmt.Errorf("principal_resolver: client_portal_grant lookup: %w", err)
		}
		for rows.Next() {
			var grantID, name sql.NullString
			if err := rows.Scan(&grantID, &name); err != nil {
				rows.Close()
				return nil, fmt.Errorf("principal_resolver: client_portal_grant scan: %w", err)
			}
			display := strings.TrimSpace(name.String)
			if display == "" {
				display = "Client"
			}
			out = append(out, &authpb.Principal{
				Type:        principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT,
				PrincipalId: grantID.String,
				WorkspaceId: workspaceID,
				DisplayName: fmt.Sprintf("%s · Client", display),
			})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("principal_resolver: client_portal_grant rows: %w", err)
		}
		rows.Close()
	}

	// 3. supplier_portal_grant → SUPPLIER
	{
		const q = `
			SELECT spg.id, COALESCE(s.name, '')
			FROM supplier_portal_grant spg
			LEFT JOIN supplier s ON s.id = spg.supplier_id AND s.active = true
			WHERE spg.user_id = $1
				AND spg.workspace_id = $2
				AND spg.active = true
			ORDER BY COALESCE(s.name, ''), spg.id
		`
		rows, err := db.QueryContext(ctx, q, userID, workspaceID)
		if err != nil {
			return nil, fmt.Errorf("principal_resolver: supplier_portal_grant lookup: %w", err)
		}
		for rows.Next() {
			var grantID, name sql.NullString
			if err := rows.Scan(&grantID, &name); err != nil {
				rows.Close()
				return nil, fmt.Errorf("principal_resolver: supplier_portal_grant scan: %w", err)
			}
			display := strings.TrimSpace(name.String)
			if display == "" {
				display = "Supplier"
			}
			out = append(out, &authpb.Principal{
				Type:        principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER,
				PrincipalId: grantID.String,
				WorkspaceId: workspaceID,
				DisplayName: fmt.Sprintf("%s · Supplier", display),
			})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("principal_resolver: supplier_portal_grant rows: %w", err)
		}
		rows.Close()
	}

	// 4. delegate_client → CLIENT_DELEGATE in this workspace
	{
		const q = `
			SELECT d.id, dc.client_id, COALESCE(c.name, '')
			FROM delegate d
			JOIN delegate_client dc
				ON dc.delegate_id = d.id AND dc.active = true
			LEFT JOIN client c ON c.id = dc.client_id AND c.active = true
			WHERE d.user_id = $1
				AND d.active = true
				AND COALESCE(dc.workspace_id, c.workspace_id) = $2
			ORDER BY d.id, COALESCE(c.name, ''), dc.client_id
		`
		rows, err := db.QueryContext(ctx, q, userID, workspaceID)
		if err != nil {
			return nil, fmt.Errorf("principal_resolver: delegate_client lookup: %w", err)
		}
		var (
			curDelegateID string
			curTargets    []*authpb.ActingAsTarget
		)
		flush := func() {
			if curDelegateID != "" && len(curTargets) > 0 {
				out = append(out, &authpb.Principal{
					Type:            principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE,
					PrincipalId:     curDelegateID,
					WorkspaceId:     workspaceID,
					DisplayName:     formatResolverDelegateLabel("Client Delegate", curTargets),
					ActingAsTargets: curTargets,
				})
			}
		}
		for rows.Next() {
			var dID, clientID, clientName sql.NullString
			if err := rows.Scan(&dID, &clientID, &clientName); err != nil {
				rows.Close()
				return nil, fmt.Errorf("principal_resolver: delegate_client scan: %w", err)
			}
			if dID.String != curDelegateID {
				flush()
				curDelegateID = dID.String
				curTargets = nil
			}
			display := strings.TrimSpace(clientName.String)
			if display == "" {
				display = "Client"
			}
			curTargets = append(curTargets, &authpb.ActingAsTarget{
				Id:          clientID.String,
				WorkspaceId: workspaceID,
				DisplayName: display,
			})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("principal_resolver: delegate_client rows: %w", err)
		}
		rows.Close()
		flush()
	}

	// 5. delegate_supplier → SUPPLIER_DELEGATE in this workspace
	{
		const q = `
			SELECT d.id, ds.supplier_id, COALESCE(s.name, '')
			FROM delegate d
			JOIN delegate_supplier ds
				ON ds.delegate_id = d.id AND ds.active = true
			LEFT JOIN supplier s ON s.id = ds.supplier_id AND s.active = true
			WHERE d.user_id = $1
				AND d.active = true
				AND COALESCE(ds.workspace_id, s.workspace_id) = $2
			ORDER BY d.id, COALESCE(s.name, ''), ds.supplier_id
		`
		rows, err := db.QueryContext(ctx, q, userID, workspaceID)
		if err != nil {
			return nil, fmt.Errorf("principal_resolver: delegate_supplier lookup: %w", err)
		}
		var (
			curDelegateID string
			curTargets    []*authpb.ActingAsTarget
		)
		flush := func() {
			if curDelegateID != "" && len(curTargets) > 0 {
				out = append(out, &authpb.Principal{
					Type:            principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE,
					PrincipalId:     curDelegateID,
					WorkspaceId:     workspaceID,
					DisplayName:     formatResolverDelegateLabel("Supplier Delegate", curTargets),
					ActingAsTargets: curTargets,
				})
			}
		}
		for rows.Next() {
			var dID, supplierID, supplierName sql.NullString
			if err := rows.Scan(&dID, &supplierID, &supplierName); err != nil {
				rows.Close()
				return nil, fmt.Errorf("principal_resolver: delegate_supplier scan: %w", err)
			}
			if dID.String != curDelegateID {
				flush()
				curDelegateID = dID.String
				curTargets = nil
			}
			display := strings.TrimSpace(supplierName.String)
			if display == "" {
				display = "Supplier"
			}
			curTargets = append(curTargets, &authpb.ActingAsTarget{
				Id:          supplierID.String,
				WorkspaceId: workspaceID,
				DisplayName: display,
			})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("principal_resolver: delegate_supplier rows: %w", err)
		}
		rows.Close()
		flush()
	}

	return &authpb.EnumerateBindingsResponse{Bindings: out}, nil
}

// LookupSessionPrincipal reads (principal_type, principal_id, acting_as_*)
// from the session row identified by token. Returns a zero response on any
// miss/error — callers treat that as the "no hint" sentinel.
//
// SQL is VERBATIM from composition/session_principal.go's
// lookupSessionPrincipalFull. Includes the direct-client acting-as derivation
// (inherited 20260601 Phase-4 gap).
func (a *PrincipalResolverAdapter) LookupSessionPrincipal(
	ctx context.Context,
	req *authpb.LookupSessionPrincipalRequest,
) (*authpb.LookupSessionPrincipalResponse, error) {
	if req == nil || req.GetToken() == "" {
		return &authpb.LookupSessionPrincipalResponse{
			Kind: principaltypepb.PrincipalType_PRINCIPAL_TYPE_UNSPECIFIED,
		}, nil
	}
	db := a.db
	token := req.GetToken()

	const q = `
		SELECT principal_type, principal_id, acting_as_client_id, acting_as_supplier_id
		FROM session
		WHERE token = $1
			AND active = true
		LIMIT 1
	`
	var (
		kindNull       sql.NullInt64
		idNull         sql.NullString
		actingClient   sql.NullString
		actingSupplier sql.NullString
	)
	err := db.QueryRowContext(ctx, q, token).
		Scan(&kindNull, &idNull, &actingClient, &actingSupplier)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("[principal_resolver] session lookup error: %v", err)
		}
		return &authpb.LookupSessionPrincipalResponse{
			Kind: principaltypepb.PrincipalType_PRINCIPAL_TYPE_UNSPECIFIED,
		}, nil
	}
	if !kindNull.Valid || !idNull.Valid {
		return &authpb.LookupSessionPrincipalResponse{
			Kind: principaltypepb.PrincipalType_PRINCIPAL_TYPE_UNSPECIFIED,
		}, nil
	}

	resp := &authpb.LookupSessionPrincipalResponse{
		Kind:               principaltypepb.PrincipalType(kindNull.Int64),
		PrincipalId:        idNull.String,
		ActingAsClientId:   actingClient.String,
		ActingAsSupplierId: actingSupplier.String,
	}

	// Direct-client acting-as derivation (inherited 20260601 Phase-4 gap).
	// SwitchPrincipal stamps acting_as_client_id only for CLIENT_DELEGATE, not
	// for a direct PRINCIPAL_TYPE_CLIENT — whose principal_id IS the
	// client_portal_grant.id — so a direct client's session row carries an empty
	// scope. Derive it read-only from the (active) grant.
	if resp.Kind == principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT &&
		resp.ActingAsClientId == "" && resp.PrincipalId != "" {
		var grantClientID sql.NullString
		const grantQ = `SELECT client_id FROM client_portal_grant WHERE id = $1 AND active = true LIMIT 1`
		if derr := db.QueryRowContext(ctx, grantQ, resp.PrincipalId).Scan(&grantClientID); derr != nil {
			if derr != sql.ErrNoRows {
				log.Printf("[principal_resolver] direct-client acting_as derivation error: %v", derr)
			}
		} else if grantClientID.Valid {
			resp.ActingAsClientId = grantClientID.String
		}
	}

	return resp, nil
}
