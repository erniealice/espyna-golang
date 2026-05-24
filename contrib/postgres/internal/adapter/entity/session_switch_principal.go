//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	principaltypepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/principal_type"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

// SwitchPrincipal implements the security-critical session-rotation
// transactional primitive. It was migrated FROM
// apps/service-admin/internal/composition/principal_switch.go (which violated
// the no-direct-sql-rule per docs/wiki/articles/no-direct-sql-rule.md §"Never
// in") INTO this adapter where direct SQL is the legitimate hexagonal layer.
// See docs/plan/20260524-principal-switch-typed-stack/ Phase 2.
//
// Decision tree (Q-WS-13 — workspace-boundary rule):
//
//	currentToken == "" or different workspace_id → ROTATE
//	  (insert new session row, mark old inactive, return new token)
//	same workspace_id, different principal_type / acting_as_* → IN-PLACE
//	  (update existing session row, return empty NewToken)
//	same workspace_id, same principal_type, same acting_as → NO-OP-LIKE
//	  (touches date_modified, returns empty NewToken)
//
// Transaction shape (red-team A-4):
//
//	BEGIN
//	  SELECT session ... WHERE token = $1 AND active = true FOR UPDATE
//	  SELECT <binding_table> ... WHERE user_id=$1 AND workspace_id=$2
//	         AND active = true FOR UPDATE      -- binding TOCTOU defense
//	  INSERT / UPDATE session row
//	  INSERT audit_entry (RequireAudit=true: failure rolls tx back)
//	COMMIT
//
// Per Q2-A lock, the audit insert lives INSIDE this transaction so the
// RequireAudit atomicity (red-team A-4) is preserved — an attacker DoS-ing
// the audit table cannot suppress evidence of a forced rotation.
//
// The method returns the RAW underlying error; the calling use case
// (packages/espyna-golang/internal/application/usecases/service/auth/
// switch_principal.go) is responsible for translating errors via the
// Translator port. Adapters never import the application layer
// (hexagonal-rules.md §1 principle 3).
func (r *PostgresSessionRepository) SwitchPrincipal(
	ctx context.Context,
	req *authpb.SwitchPrincipalRequest,
) (*authpb.SwitchPrincipalResponse, error) {
	if r.db == nil {
		return nil, errors.New("session adapter: SwitchPrincipal requires direct *sql.DB access (GetDB shim missing)")
	}
	if req == nil {
		return nil, errors.New("session adapter: SwitchPrincipal: nil request")
	}
	if req.GetUserId() == "" {
		return nil, errors.New("session adapter: SwitchPrincipal: user_id required")
	}
	tgt := req.GetTargetPrincipal()
	if tgt == nil {
		return nil, errors.New("session adapter: SwitchPrincipal: target_principal required")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("session adapter: SwitchPrincipal: begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// ─── Read current session state (if any) ────────────────────────────────
	// FOR UPDATE serializes concurrent rotations for the same session
	// (red-team A-4 / Q-WS-13 risk register). Without it two parallel
	// requests like /w/A/x and /w/B/x from the same cookie could both
	// observe the same pre-state and both write new "active" rows, leaving
	// orphan active sessions.
	var (
		curSessionID          sql.NullString
		curPrincipalType      sql.NullInt32
		curActingAsClientID   sql.NullString
		curActingAsSupplierID sql.NullString
		curWorkspaceID        sql.NullString
		curWorkspaceUserID    sql.NullString
	)
	if req.GetToken() != "" {
		err := tx.QueryRowContext(ctx, `
			SELECT id, principal_type, acting_as_client_id, acting_as_supplier_id,
			       workspace_id, workspace_user_id
			FROM `+r.tableName+`
			WHERE token = $1 AND active = true
			LIMIT 1
			FOR UPDATE
		`, req.GetToken()).Scan(
			&curSessionID,
			&curPrincipalType,
			&curActingAsClientID,
			&curActingAsSupplierID,
			&curWorkspaceID,
			&curWorkspaceUserID,
		)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("session adapter: SwitchPrincipal: read current session: %w", err)
		}
	}

	// ─── Resolve target session fields ──────────────────────────────────────
	newPrincipalType := int32(tgt.GetType())

	// workspace_id on the new session row is the workspace the principal
	// operates in. For a delegate, that's the workspace of the chosen
	// acting_as target.
	newWorkspaceID := tgt.GetWorkspaceId()
	newWorkspaceUserID := ""
	switch tgt.GetType() {
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_OPERATOR_OWNER,
		principaltypepb.PrincipalType_PRINCIPAL_TYPE_OPERATOR_STAFF:
		// principal_id IS the workspace_user.id for staff principals;
		// keep that mapping on workspace_user_id too for back-compat with
		// the existing session middleware / permission loader chain.
		newWorkspaceUserID = tgt.GetPrincipalId()
	}

	var (
		newActingAsClientID   string
		newActingAsSupplierID string
	)
	switch tgt.GetType() {
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE:
		if req.GetActingAsClientId() != "" {
			// A2-followup round-3 (2026-05-24): when an explicit
			// acting-as id is supplied (URL-driven /w/{ws}/as/{id} or
			// explicit form post), validate it against the binding's
			// ActingAsTargets slice when the slice is non-empty. The
			// workspace_path middleware now always rewrites the slice
			// to the URL-selected target before calling here (see
			// workspace_path.go step 8b), so a mismatch at this point
			// is either a wiring bug or an explicit-form caller
			// passing an id that's not in the resolved binding. Fail
			// closed rather than silently fall through.
			//
			// Authoritative grant validation still runs at the SQL
			// JOIN in lockTargetBinding below; this guard catches the
			// in-process drift before the tx writes anything.
			if len(tgt.GetActingAsTargets()) > 0 && !actingAsTargetIDsContain(tgt.GetActingAsTargets(), req.GetActingAsClientId()) {
				return nil, fmt.Errorf("session adapter: SwitchPrincipal: requested acting_as_client_id %q is not in the resolved binding's targets (delegate=%s, available=%s)",
					req.GetActingAsClientId(), tgt.GetPrincipalId(), formatActingAsTargetIDs(tgt.GetActingAsTargets()))
			}
			newActingAsClientID = req.GetActingAsClientId()
		} else if len(tgt.GetActingAsTargets()) == 1 {
			newActingAsClientID = tgt.GetActingAsTargets()[0].GetId()
		}
		// Override workspace_id from the chosen target (each client may
		// live in a different workspace).
		for _, t := range tgt.GetActingAsTargets() {
			if t.GetId() == newActingAsClientID && t.GetWorkspaceId() != "" {
				newWorkspaceID = t.GetWorkspaceId()
				break
			}
		}
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE:
		if req.GetActingAsSupplierId() != "" {
			// Symmetric to CLIENT_DELEGATE above — A2-followup round-3
			// fail-closed guard for the supplier-delegate path.
			if len(tgt.GetActingAsTargets()) > 0 && !actingAsTargetIDsContain(tgt.GetActingAsTargets(), req.GetActingAsSupplierId()) {
				return nil, fmt.Errorf("session adapter: SwitchPrincipal: requested acting_as_supplier_id %q is not in the resolved binding's targets (delegate=%s, available=%s)",
					req.GetActingAsSupplierId(), tgt.GetPrincipalId(), formatActingAsTargetIDs(tgt.GetActingAsTargets()))
			}
			newActingAsSupplierID = req.GetActingAsSupplierId()
		} else if len(tgt.GetActingAsTargets()) == 1 {
			newActingAsSupplierID = tgt.GetActingAsTargets()[0].GetId()
		}
		for _, t := range tgt.GetActingAsTargets() {
			if t.GetId() == newActingAsSupplierID && t.GetWorkspaceId() != "" {
				newWorkspaceID = t.GetWorkspaceId()
				break
			}
		}
	}

	// ─── Decide: rotate vs. in-place (Q-WS-13: workspace-boundary rule) ─────
	// Pre-Q-WS-13 we rotated on principal_type change. That conflated
	// principal_type with workspace boundary and missed the
	// two-workspace-same-principal-type case (Phase P3+P7 fix). Now: rotate
	// iff the workspace_id changes; otherwise in-place mutation.
	shouldRotate := true
	if curSessionID.Valid && curWorkspaceID.Valid && curWorkspaceID.String != "" &&
		curWorkspaceID.String == newWorkspaceID {
		shouldRotate = false // same workspace → in-place
	}
	// If we have no prior session row at all (login bootstrap / no cookie),
	// always rotate to mint the principal-scoped row.
	if !curSessionID.Valid {
		shouldRotate = true
	}

	// ─── Active-binding re-check inside tx (red-team A-4 / verify H-5) ──────
	// The middleware (or explicit-form handler) validated the binding
	// outside the tx; this re-check defends against an admin revoking the
	// binding between that outside check and our writes. The lock-row
	// shape varies by principal type — we issue ONE locked SELECT keyed on
	// the target type. Missing row aborts the rotation cleanly.
	//
	// Note: delegate principals lock the delegate_client / delegate_supplier
	// "acting-as" row when one is selected, since that's the actual grant
	// boundary that could be revoked. When no acting_as_* is set yet
	// (multi-target picker not yet chosen), we lock the parent delegate row.
	if curSessionID.Valid {
		// Skip the re-check at login-bootstrap (no prior session row): the
		// caller has just resolved principals from the same DB; locking
		// here adds no defense and would block any user with no binding
		// from logging in to a no-access page.
		if err := lockTargetBinding(ctx, tx, req.GetUserId(), tgt, newActingAsClientID, newActingAsSupplierID); err != nil {
			return nil, err
		}
	}

	var (
		newToken     string
		newSessionID string
	)

	if shouldRotate {
		newToken, err = generateOpaqueSwitchToken()
		if err != nil {
			return nil, fmt.Errorf("session adapter: SwitchPrincipal: gen token: %w", err)
		}
		newSessionID = uuid.New().String()
		expiresAt := time.Now().Add(7 * 24 * time.Hour).UnixMilli()
		nowMs := time.Now().UnixMilli()

		// Insert the new session row with the chosen principal context.
		// principal_type is written as the proto enum integer; NULLs for
		// optional FK columns are passed via NULLIF($n, '').
		_, err = tx.ExecContext(ctx, `
			INSERT INTO `+r.tableName+` (
				id, user_id, token,
				workspace_user_id, workspace_id,
				expires_at, active,
				date_created, date_modified,
				principal_type, principal_id,
				acting_as_client_id, acting_as_supplier_id, acting_as_workspace_id
			) VALUES (
				$1, $2, $3,
				NULLIF($4, ''), NULLIF($5, ''),
				$6, true,
				$7, $7,
				$8, NULLIF($9, ''),
				NULLIF($10, ''), NULLIF($11, ''), NULLIF($12, '')
			)
		`,
			newSessionID, req.GetUserId(), newToken,
			newWorkspaceUserID, newWorkspaceID,
			expiresAt,
			nowMs,
			newPrincipalType, tgt.GetPrincipalId(),
			newActingAsClientID, newActingAsSupplierID, newWorkspaceID,
		)
		if err != nil {
			return nil, fmt.Errorf("session adapter: SwitchPrincipal: insert new session: %w", err)
		}

		// Invalidate the prior session row so the old token 401s.
		if curSessionID.Valid && curSessionID.String != "" {
			_, err = tx.ExecContext(ctx, `
				UPDATE `+r.tableName+`
				SET active = false,
				    date_modified = $1
				WHERE id = $2
			`, nowMs, curSessionID.String)
			if err != nil {
				return nil, fmt.Errorf("session adapter: SwitchPrincipal: invalidate old session: %w", err)
			}
		}
	} else {
		// In-place: same workspace_id → principal_type / acting_as may move
		// without rotation. Returns no new token; the cookie keeps pointing
		// at the same row.
		//
		// A4 fix (WKR-P0-4, 2026-05-22): include principal_type in the SET
		// list. The Mutual co-op scenario (Carol picks her CLIENT role in
		// sunrise from WORKSPACE_USER) keeps the same workspace_id, so the
		// rotation path is skipped — but principal_id alone is not enough;
		// the (principal_id, principal_type) tuple must stay coherent or
		// the session middleware / permission loader read the wrong principal
		// kind. Pre-fix, principal_type was never updated on this path,
		// leaving the row in an incoherent state.
		newSessionID = curSessionID.String
		nowMs := time.Now().UnixMilli()
		_, err = tx.ExecContext(ctx, `
			UPDATE `+r.tableName+`
			SET workspace_id           = NULLIF($1, ''),
			    workspace_user_id      = NULLIF($2, ''),
			    principal_type         = $3,
			    principal_id           = NULLIF($4, ''),
			    acting_as_client_id    = NULLIF($5, ''),
			    acting_as_supplier_id  = NULLIF($6, ''),
			    acting_as_workspace_id = NULLIF($7, ''),
			    date_modified          = $8
			WHERE id = $9
		`,
			newWorkspaceID, newWorkspaceUserID,
			newPrincipalType, tgt.GetPrincipalId(),
			newActingAsClientID, newActingAsSupplierID, newWorkspaceID,
			nowMs,
			curSessionID.String,
		)
		if err != nil {
			return nil, fmt.Errorf("session adapter: SwitchPrincipal: in-place update: %w", err)
		}
	}

	// ─── Audit row ──────────────────────────────────────────────────────────
	// Forensic metadata (request_url / referer / sec_fetch_site / user_agent)
	// is folded into the `reason` text below until a follow-up migration
	// adds dedicated columns to audit_trail.audit_entry. Choice rationale:
	// adding columns requires a migration which is out of scope for this
	// rotation-primitive refactor; the structured-reason format is
	// machine-parseable (key:value space-delimited) and a future migration
	// can backfill columns by re-parsing existing rows.
	reason := fmt.Sprintf(
		"principal_type:%s→%s acting_as_client:%s→%s acting_as_supplier:%s→%s rotated:%t",
		coalesceInt32PrincipalTypeString(curPrincipalType), principalTypeAuditLabel(tgt.GetType()),
		coalesceSwitchNullString(curActingAsClientID), newActingAsClientID,
		coalesceSwitchNullString(curActingAsSupplierID), newActingAsSupplierID,
		shouldRotate,
	)
	if req.GetRequestUrl() != "" {
		reason += " url:" + sanitizeSwitchAuditField(req.GetRequestUrl())
	}
	if req.GetReferer() != "" {
		reason += " referer:" + sanitizeSwitchAuditField(req.GetReferer())
	}
	if req.GetSecFetchSite() != "" {
		reason += " sec_fetch_site:" + sanitizeSwitchAuditField(req.GetSecFetchSite())
	}
	if req.GetUserAgent() != "" {
		reason += " ua:" + sanitizeSwitchAuditField(req.GetUserAgent())
	}

	// Resolve the audit use_case discriminator.
	// When the caller leaves UseCase UNSPECIFIED we derive it from the
	// rotation decision + principal/acting-as change detection (A5 —
	// WKR-P1-4). Explicit caller-supplied UseCase wins (preserves intent
	// for callers that already tag their event correctly).
	useCase := req.GetUseCase()
	if useCase == authpb.SwitchUseCase_SWITCH_USE_CASE_UNSPECIFIED {
		principalTypeChanged := curPrincipalType.Valid && curPrincipalType.Int32 != newPrincipalType
		actingAsChanged :=
			coalesceNullStringOrSentinel(curActingAsClientID.String) != coalesceNullStringOrSentinel(newActingAsClientID) ||
				coalesceNullStringOrSentinel(curActingAsSupplierID.String) != coalesceNullStringOrSentinel(newActingAsSupplierID)
		useCase = deriveSwitchUseCaseEnum(req.GetUrlDriven(), shouldRotate, principalTypeChanged, actingAsChanged)
	}

	auditErr := writeSwitchAuditRow(ctx, tx, switchAuditRow{
		UserID:       req.GetUserId(),
		WorkspaceID:  newWorkspaceID,
		EntityID:     newSessionID,
		UseCaseLabel: switchUseCaseAuditLabel(useCase),
		Reason:       reason,
		RotatedToken: shouldRotate,
		RequireAudit: req.GetRequireAudit(),
	})
	if auditErr != nil {
		if req.GetRequireAudit() {
			// URL-driven rotation: audit failure must abort the tx
			// (red-team A-4 stealth-rotation defense).
			return nil, fmt.Errorf("session adapter: SwitchPrincipal: required audit failed: %w", auditErr)
		}
		// Explicit-form callers: log and continue (dev-mode best-effort).
		log.Printf("[session_switch_principal] audit write failed (non-fatal, RequireAudit=false): %v", auditErr)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("session adapter: SwitchPrincipal: commit: %w", err)
	}
	committed = true

	resp := &authpb.SwitchPrincipalResponse{}
	if shouldRotate {
		resp.NewToken = newToken
	}
	// RedirectURL: the composition wrapper / use case computes this from
	// the resolved Principal (Principal.HomeRoute() is composition-layer
	// vocabulary). The adapter returns an empty string; the use case fills
	// it in. This keeps the adapter free of HTTP routing concerns
	// (hexagonal-rules.md §6 anti-patterns: adapters don't reach upward).
	return resp, nil
}

// switchAuditRow is the minimal field set we write to audit_entry on every
// principal switch. ActorType is hardcoded to "USER" since the actor is the
// authenticated human; field-level changes (and forensic metadata like
// Referer / Sec-Fetch-Site / UserAgent) are encoded into Reason text since
// the audit_field_change machinery is entity-CRUD-oriented and would be
// heavier than this single-row event warrants. A follow-up migration can
// promote the metadata into dedicated columns and reparse old rows.
type switchAuditRow struct {
	UserID       string
	WorkspaceID  string
	EntityID     string
	UseCaseLabel string // pre-computed audit-row label for the switch event
	Reason       string
	RotatedToken bool
	RequireAudit bool // when true, audit-insert failure must propagate (caller rolls tx back)
}

// writeSwitchAuditRow inserts a single audit_entry row.
//
// Behavior depends on row.RequireAudit:
//   - RequireAudit=false (explicit-form callers): writes the row inside a
//     SAVEPOINT so that a missing audit_entry table (dev DBs that haven't
//     run all migrations) does not abort the outer rotation. Returns nil
//     on success AND on swallowed savepoint/insert failures — the caller
//     will see no error.
//   - RequireAudit=true (URL-driven callers from workspace_path middleware):
//     writes the row inline (no savepoint) so that any failure bubbles up
//     and aborts the rotation transaction. Closes red-team A-4
//     stealth-rotation attack.
//
// Returns the underlying error only when RequireAudit=true.
func writeSwitchAuditRow(ctx context.Context, tx *sql.Tx, row switchAuditRow) error {
	const auditAction = "AUDIT_ACTION_UPDATE"
	const actorType = "ACTOR_TYPE_USER"

	// occurred_at is timestamptz, populated via NOW() to avoid Go-side TZ
	// nuance. id is generated client-side to match the rest of the codebase.
	auditID := uuid.New().String()
	useCase := row.UseCaseLabel
	if useCase == "" {
		// Defensive default; pre-refactor callers used "switch_principal".
		useCase = "switch_principal"
	}

	const insertSQL = `
		INSERT INTO audit_trail.audit_entry (
			id, workspace_id,
			actor_id, actor_type, actor_ip, actor_user_agent,
			entity_type, entity_id, domain, action,
			permission_code, use_case, reason, method_name,
			request_id, transaction_id, field_count,
			occurred_at
		) VALUES (
			$1, NULLIF($2, '')::uuid,
			$3, $4, NULL, NULL,
			'session', $5, 'auth', $6,
			NULL, $7, $8, '',
			NULL, 0, 0,
			NOW()
		)
	`

	if row.RequireAudit {
		// Strict mode: any failure propagates so the outer tx rolls back.
		if _, err := tx.ExecContext(ctx, insertSQL,
			auditID, row.WorkspaceID,
			row.UserID, actorType,
			row.EntityID, auditAction,
			useCase, row.Reason,
		); err != nil {
			return fmt.Errorf("audit insert (require_audit=true): %w", err)
		}
		return nil
	}

	// Best-effort mode: savepoint isolates audit failures from the rotation.
	if _, spErr := tx.ExecContext(ctx, `SAVEPOINT audit_sp`); spErr != nil {
		log.Printf("[session_switch_principal] audit savepoint failed (non-fatal): %v", spErr)
		return nil
	}
	if _, err := tx.ExecContext(ctx, insertSQL,
		auditID, row.WorkspaceID,
		row.UserID, actorType,
		row.EntityID, auditAction,
		useCase, row.Reason,
	); err != nil {
		// Audit drift: roll back to savepoint so the outer tx stays healthy.
		log.Printf("[session_switch_principal] audit insert failed (non-fatal): %v", err)
		_, _ = tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT audit_sp`)
		return nil
	}
	_, _ = tx.ExecContext(ctx, `RELEASE SAVEPOINT audit_sp`)
	return nil
}

// sanitizeSwitchAuditField scrubs whitespace so the structured reason
// key:value list parses unambiguously. Newlines and carriage returns are
// replaced with spaces; surrounding whitespace is trimmed; the result is
// truncated to keep audit rows bounded.
func sanitizeSwitchAuditField(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.TrimSpace(s)
	const maxLen = 256
	if len(s) > maxLen {
		s = s[:maxLen]
	}
	return s
}

// lockTargetBinding takes a row-level lock on the user's binding for the
// target principal/workspace. Called INSIDE the rotation transaction after
// the session-row lock and before any session writes. Closes the binding
// TOCTOU race (red-team verify S-2 / H-5): an admin could revoke the
// binding between middleware-validates and primitive-writes; without this
// lock the primitive could write a new session row pointing at a revoked
// grant.
//
// Returns nil if a matching active binding row exists (and is now locked
// for the duration of the tx). Returns an error if no matching active row
// is found — rotation aborts cleanly via the deferred tx.Rollback().
func lockTargetBinding(
	ctx context.Context,
	tx *sql.Tx,
	userID string,
	tgt *authpb.Principal,
	actingAsClientID, actingAsSupplierID string,
) error {
	var (
		query string
		args  []any
	)
	switch tgt.GetType() {
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_OPERATOR_OWNER,
		principaltypepb.PrincipalType_PRINCIPAL_TYPE_OPERATOR_STAFF:
		// tgt.PrincipalID == workspace_user.id; user_id + workspace_id are
		// the natural keys but we already have the row id from the loader,
		// so lock by id with the user_id and active=true predicate as a
		// defense-in-depth check.
		query = `
			SELECT id FROM workspace_user
			WHERE id = $1 AND user_id = $2 AND active = true
			LIMIT 1
			FOR UPDATE
		`
		args = []any{tgt.GetPrincipalId(), userID}
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT:
		query = `
			SELECT id FROM client_portal_grant
			WHERE id = $1 AND user_id = $2 AND active = true
			LIMIT 1
			FOR UPDATE
		`
		args = []any{tgt.GetPrincipalId(), userID}
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER:
		query = `
			SELECT id FROM supplier_portal_grant
			WHERE id = $1 AND user_id = $2 AND active = true
			LIMIT 1
			FOR UPDATE
		`
		args = []any{tgt.GetPrincipalId(), userID}
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE:
		// Lock the acting-as row when a specific target is chosen; that's
		// the row an admin would revoke. When N=1 the loader chose it;
		// when N>1 the caller passed acting_as_client_id. If neither is
		// set (multi-target picker not yet resolved), lock the parent
		// delegate row.
		if actingAsClientID != "" {
			// $4 = tgt.WorkspaceID enforces that the locked
			// delegate_client row resolves to the SAME workspace the
			// caller is asking to switch into. A2-followup round-3
			// fix: without this predicate a delegate holding
			// (delegate_id=D, client_id=Z in workspace B) could
			// navigate /w/workspace-A/as/client-Z/... and the lock
			// would still succeed because D→Z is a real grant —
			// session row then mutates to (workspace_id=A,
			// acting_as_client_id=Z), which is incoherent.
			query, args = buildDelegateLockSQL(
				principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE,
				tgt.GetPrincipalId(), actingAsClientID, userID, tgt.GetWorkspaceId(),
			)
		} else {
			query, args = buildDelegateLockSQL(
				principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE,
				tgt.GetPrincipalId(), "", userID, "",
			)
		}
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE:
		if actingAsSupplierID != "" {
			// $4 = tgt.WorkspaceID — symmetric to the client-delegate
			// branch above; see comment there for the round-3 rationale.
			query, args = buildDelegateLockSQL(
				principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE,
				tgt.GetPrincipalId(), actingAsSupplierID, userID, tgt.GetWorkspaceId(),
			)
		} else {
			query, args = buildDelegateLockSQL(
				principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE,
				tgt.GetPrincipalId(), "", userID, "",
			)
		}
	default:
		return fmt.Errorf("session adapter: SwitchPrincipal: unsupported principal type for binding lock: %v", tgt.GetType())
	}

	var lockedID sql.NullString
	err := tx.QueryRowContext(ctx, query, args...).Scan(&lockedID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("session adapter: SwitchPrincipal: binding revoked or not in workspace (type=%s principal_id=%s workspace_id=%s)",
			principalTypeAuditLabel(tgt.GetType()), tgt.GetPrincipalId(), tgt.GetWorkspaceId())
	}
	if err != nil {
		return fmt.Errorf("session adapter: SwitchPrincipal: binding lock query: %w", err)
	}
	return nil
}

// buildDelegateLockSQL returns the SELECT...FOR UPDATE query string AND the
// matching positional-args slice for the delegate binding lock, parameterised
// on the delegate kind and whether an acting-as id was supplied.
//
// **Refactor note (closes codex auth-collapse R4 P2, 2026-05-24):** the
// previous signature returned `(sql string, argCount int)` and required the
// caller to assemble the args slice separately, creating a drift footgun
// where the SQL placeholders ($1..$4) and the args slice could fall out of
// sync silently. The new signature returns BOTH artifacts together — the
// helper is the single source of truth for the lock's positional contract.
//
// Argument shapes (caller passes the same 5 values; helper picks the right
// subset based on `actingAsID` being empty or not):
//
//	actingAsID != "":  args = [delegateID, actingAsID, userID, workspaceID]  (4 args; $1..$4)
//	actingAsID == "":  args = [delegateID, userID]                            (2 args; $1, $2)
//
// Symmetric for ClientDelegate vs SupplierDelegate (different tables /
// alias names; identical workspace-predicate shape).
//
// Unknown kinds return ("", nil) — the caller is the only entrypoint and
// already switches on kind, so this branch is unreachable in production.
func buildDelegateLockSQL(
	kind principaltypepb.PrincipalType,
	delegateID, actingAsID, userID, workspaceID string,
) (string, []any) {
	switch kind {
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE:
		if actingAsID != "" {
			return `
					SELECT dc.id
					FROM delegate_client dc
					JOIN delegate d ON d.id = dc.delegate_id AND d.active = true
					LEFT JOIN client c ON c.id = dc.client_id AND c.active = true
					WHERE dc.delegate_id = $1 AND dc.client_id = $2
						AND dc.active = true
						AND d.user_id = $3
						AND COALESCE(dc.workspace_id, c.workspace_id) = $4
					LIMIT 1
					FOR UPDATE
				`, []any{delegateID, actingAsID, userID, workspaceID}
		}
		return `
					SELECT id FROM delegate
					WHERE id = $1 AND user_id = $2 AND active = true
					LIMIT 1
					FOR UPDATE
				`, []any{delegateID, userID}
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE:
		if actingAsID != "" {
			return `
					SELECT ds.id
					FROM delegate_supplier ds
					JOIN delegate d ON d.id = ds.delegate_id AND d.active = true
					LEFT JOIN supplier s ON s.id = ds.supplier_id AND s.active = true
					WHERE ds.delegate_id = $1 AND ds.supplier_id = $2
						AND ds.active = true
						AND d.user_id = $3
						AND COALESCE(ds.workspace_id, s.workspace_id) = $4
					LIMIT 1
					FOR UPDATE
				`, []any{delegateID, actingAsID, userID, workspaceID}
		}
		return `
					SELECT id FROM delegate
					WHERE id = $1 AND user_id = $2 AND active = true
					LIMIT 1
					FOR UPDATE
				`, []any{delegateID, userID}
	}
	return "", nil
}

// --- helpers (suffixed to avoid clashing with future entity-package helpers) ---

// generateOpaqueSwitchToken matches the format produced by espyna's
// IssueSessionUseCase — 32 random bytes hex-encoded — so the cookie value
// is indistinguishable from a fresh-login token. The session middleware
// resolves either shape through the same `SELECT * FROM session WHERE
// token = $1` path.
func generateOpaqueSwitchToken() (string, error) {
	// Two UUIDs back-to-back = 32 bytes. UUID v4 is crypto/rand-backed.
	a := uuid.New()
	b := uuid.New()
	out := make([]byte, 0, 64)
	for _, x := range a {
		out = append(out, hexDigit(x>>4), hexDigit(x&0xF))
	}
	for _, x := range b {
		out = append(out, hexDigit(x>>4), hexDigit(x&0xF))
	}
	return string(out), nil
}

func hexDigit(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'a' + (n - 10)
}

// coalesceInt32PrincipalTypeString renders the prior principal_type column
// value into the human-readable label for the audit reason text.
// Returns "unset" when the column was NULL (login-bootstrap path).
func coalesceInt32PrincipalTypeString(v sql.NullInt32) string {
	if !v.Valid {
		return "unset"
	}
	return principalTypeAuditLabel(principaltypepb.PrincipalType(v.Int32))
}

// principalTypeAuditLabel maps a proto PrincipalType to the lowercase audit
// label the pre-refactor primitive emitted via adapthttp.PrincipalType.String()
// (apps/service-admin/internal/infrastructure/input/http/principal_loader.go).
// The audit reason text is machine-parsed by forensic tooling that expects
// these stable lowercase labels (e.g. "client_delegate"), NOT the proto enum's
// String() form ("PRINCIPAL_TYPE_CLIENT_DELEGATE"). Codex round 1 P1: keeping
// this in sync preserves audit-row content across the composition→adapter move.
func principalTypeAuditLabel(pt principaltypepb.PrincipalType) string {
	switch pt {
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_OPERATOR_OWNER:
		return "operator_owner"
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_OPERATOR_STAFF:
		return "operator_staff"
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT:
		return "client"
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE:
		return "client_delegate"
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER:
		return "supplier"
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE:
		return "supplier_delegate"
	default:
		return "unspecified"
	}
}

// coalesceSwitchNullString renders a NullString column into "-" when NULL
// or empty, else the underlying string. Used in the audit reason text so
// the (NULL, "") pair both render identically — they mean "no acting-as
// target".
func coalesceSwitchNullString(v sql.NullString) string {
	if !v.Valid || v.String == "" {
		return "-"
	}
	return v.String
}

// coalesceNullStringOrSentinel normalises an empty string to the sentinel
// "-" so the equality check in deriveSwitchUseCaseEnum doesn't false-positive
// on the (null, "") pair which both mean "no acting-as target".
func coalesceNullStringOrSentinel(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// deriveSwitchUseCaseEnum returns one of the 6 audit use_case discriminators
// (as the proto SwitchUseCase enum) for the switch event (A5 — WKR-P1-4,
// red-team X-2).
//
// Decision tree:
//
//	shouldRotate == true → URL_ROTATE or EXPLICIT_ROTATE
//	                       (workspace_id changed; acting-as and principal_type
//	                       deltas are implied by the rotation)
//
//	shouldRotate == false, principalTypeChanged → URL_PRINCIPAL_INPLACE or
//	                       EXPLICIT_INPLACE (same workspace, different
//	                       principal_type; principal_type wins over acting-as
//	                       when both are true)
//
//	shouldRotate == false, !principalTypeChanged, actingAsChanged →
//	                       URL_ACTING_AS_INPLACE or EXPLICIT_ACTING_AS
//
//	shouldRotate == false, neither → URL_PRINCIPAL_INPLACE or
//	                       EXPLICIT_INPLACE (degenerate no-op-like update;
//	                       audit still records the event so nothing goes
//	                       silent)
func deriveSwitchUseCaseEnum(urlDriven, shouldRotate, principalTypeChanged, actingAsChanged bool) authpb.SwitchUseCase {
	if shouldRotate {
		if urlDriven {
			return authpb.SwitchUseCase_SWITCH_USE_CASE_URL_ROTATE
		}
		return authpb.SwitchUseCase_SWITCH_USE_CASE_EXPLICIT_ROTATE
	}
	// In-place mutation.
	if principalTypeChanged || (!actingAsChanged) {
		// principal_type wins; degenerate case maps to principal bucket.
		if urlDriven {
			return authpb.SwitchUseCase_SWITCH_USE_CASE_URL_PRINCIPAL_INPLACE
		}
		return authpb.SwitchUseCase_SWITCH_USE_CASE_EXPLICIT_INPLACE
	}
	// actingAsChanged && !principalTypeChanged
	if urlDriven {
		return authpb.SwitchUseCase_SWITCH_USE_CASE_URL_ACTING_AS_INPLACE
	}
	return authpb.SwitchUseCase_SWITCH_USE_CASE_EXPLICIT_ACTING_AS
}

// switchUseCaseAuditLabel maps the proto SwitchUseCase enum back to the
// stable string label written into audit_trail.audit_entry.use_case. The
// strings are the pre-refactor values that reporting / forensic tooling
// may already grep for; the enum is the wire-shape contract. Mapping at
// write-time keeps both surfaces stable.
func switchUseCaseAuditLabel(uc authpb.SwitchUseCase) string {
	switch uc {
	case authpb.SwitchUseCase_SWITCH_USE_CASE_URL_ROTATE:
		return "switch_url_rotate"
	case authpb.SwitchUseCase_SWITCH_USE_CASE_URL_ACTING_AS_INPLACE:
		return "switch_url_acting_as_inplace"
	case authpb.SwitchUseCase_SWITCH_USE_CASE_URL_PRINCIPAL_INPLACE:
		return "switch_url_principal_inplace"
	case authpb.SwitchUseCase_SWITCH_USE_CASE_EXPLICIT_ROTATE:
		return "switch_explicit_rotate"
	case authpb.SwitchUseCase_SWITCH_USE_CASE_EXPLICIT_INPLACE:
		return "switch_explicit_inplace"
	case authpb.SwitchUseCase_SWITCH_USE_CASE_EXPLICIT_ACTING_AS:
		return "switch_explicit_acting_as"
	}
	// Defensive default — keeps the audit row writable when an upstream
	// caller forgets to set the discriminator. Matches the pre-refactor
	// fallback in apps/service-admin/internal/composition/principal_switch.go.
	return "switch_principal"
}

// actingAsTargetIDsContain reports whether the given acting-as id appears
// in the resolved binding's ActingAsTargets slice. Used by SwitchPrincipal
// to fail closed when the URL- or form-supplied acting-as id doesn't match
// any of the targets the binding resolver returned (A2-followup round-3,
// 2026-05-24).
func actingAsTargetIDsContain(targets []*authpb.ActingAsTarget, id string) bool {
	if id == "" {
		return false
	}
	for _, t := range targets {
		if t.GetId() == id {
			return true
		}
	}
	return false
}

// formatActingAsTargetIDs joins the ids in an ActingAsTargets slice for
// logging / error messages. Output is comma-separated; an empty slice
// returns "(none)" so the resulting error string stays readable.
func formatActingAsTargetIDs(targets []*authpb.ActingAsTarget) string {
	if len(targets) == 0 {
		return "(none)"
	}
	ids := make([]string, 0, len(targets))
	for _, t := range targets {
		ids = append(ids, t.GetId())
	}
	return strings.Join(ids, ",")
}
