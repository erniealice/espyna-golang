//go:build postgresql

package core

import (
	"context"
	"database/sql"
	"log"
	"os"
	"sync"

	"github.com/erniealice/espyna-golang/consumer"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/database/model"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// columnLessTenantTables is the set of TENANT-scoped tables that SHOULD be
// workspace-isolated but currently lack a workspace_id column in the baseline
// schema (packages/esqyma/scripts/init/baseline.sql — verified column-less, with
// NO post-baseline migration adding the column as of 2026-05-30). Because they
// are column-less, the WorkspaceAwareOperations decorator's
// tableHasWorkspaceColumn() gate is FALSE for them, so generic Update / Delete /
// HardDelete / Read-by-id silently pass through with NO tenant predicate — a
// cross-tenant IDOR surface (workspace-scoping-1).
//
// SHADOW INSTRUMENTATION (Plan 3 W2 step 1 — zero behavior change): membership in
// this set is used ONLY to decide whether to emit an AUTHZ_WS_SHADOW_PASS log
// line on the column-less pass-through. It does NOT change the verdict — the call
// still passes through exactly as before. This mirrors the W0
// AUTHZ_RBAC_SHADOW_DENY posture (secondary/auth/rbac/authorizer.go): measure the
// real runtime surface before fail-closing. The static set comes from the
// w0-design.md §3 tenancy parent-JOIN map (needsMigration list, §3.2),
// cross-checked against baseline.sql.
//
// Genuinely-GLOBAL column-less tables (workspace_user_role, session, account_group
// and other reference/lookup tables) are deliberately ABSENT — their column-less
// pass-through is CORRECT, so they get no log line (no false positives).
//
// The two proto-only entries (collection_schedule, disbursement_schedule) have no
// baseline DDL table, so they never match a real runtime table; they are listed
// for forward-completeness against the w0-design map and are runtime no-ops.
//
// Maintenance: when a table here receives its additive workspace_id migration
// (the Q-AWS3-A / W2 later sub-wave), tableHasWorkspaceColumn() flips to TRUE and
// the real workspace predicate applies — at that point remove the table from this
// set (it is no longer a column-less pass-through).
//
// REMOVED in W2 step-2 (Plan 3): account, expenditure, journal_entry. The esqyma
// per-row-derivation migration JUST populated a real workspace_id column on all
// three, so tableHasWorkspaceColumn() now returns TRUE for them and they take the
// DIRECT-column path (List injectWorkspaceFilter + Read ownership check) — NO
// parent-JOIN needed. They MUST NOT remain here or they would be double-handled
// (parent-JOIN probe instead of the now-correct direct-column predicate). NOTE:
// tableHasWorkspaceColumn caches information_schema per table-name for the process
// lifetime; a process that cached `false` for these three BEFORE the migration
// applied keeps treating them as column-less until restart — the map removal AND a
// process restart are both required for correct direct-column scoping.
var columnLessTenantTables = map[string]bool{
	"treasury_collection":      true,
	"treasury_disbursement":    true,
	"invoice":                  true,
	"expenditure_line_item":    true,
	"journal_line":             true,
	"security_deposit":         true,
	"loan":                     true,
	"loan_payment":             true,
	"petty_cash_fund":          true,
	"petty_cash_replenishment": true,
	"petty_cash_voucher":       true,
	// proto-only (no baseline DDL) — listed for forward-completeness, runtime no-op:
	"collection_schedule":   true,
	"disbursement_schedule": true,
}

// parentJoinProbe describes how to DERIVE a column-less tenant row's owning
// workspace_id via a single-hop JOIN to a workspace-bearing ancestor, per the
// w0-design.md §3 tenancy parent-JOIN map. It is the W2 step-2 instrument: the
// decorator runs `SELECT <parentWsExpr> FROM <table> <childAlias> LEFT JOIN
// <parentTable> <parentAlias> ON <joinCond> WHERE <childAlias>.id = $1` to compute
// the would-be tenant verdict for a generic Read/Update/Delete/HardDelete by id.
//
// fkColumn is the child FK that anchors the join; childAlias/parentAlias/parentTable
// name the SQL aliases. The probe SELECTs the parent workspace_id (parentWsExpr).
// A NULL anchor (FK NULL) or a row that does not belong to the acting workspace is
// the cross-tenant / fail-closed-exclude case. Mirrors the revenue_attribute.go
// exemplar `LEFT JOIN revenue rv ON ra.revenue_id = rv.id ... AND ($N=” OR
// rv.workspace_id = $N)`, except the decorator computes the verdict in Go (rather
// than embedding the predicate) so it can branch shadow-vs-enforce per row.
type parentJoinProbe struct {
	fkColumn    string // child FK column anchoring the join (for documentation / NULL semantics)
	childAlias  string
	parentAlias string
	parentTable string
	joinCond    string // ON predicate, e.g. "c.revenue_id = p.id"
	parentWs    string // expression yielding the derived workspace_id, e.g. "p.workspace_id"
}

// columnLessTenantParentJoins maps a column-less tenant table to its SINGLE-HOP
// parent-JOIN probe. ONLY tables whose immediate workspace-bearing ancestor
// actually has the column TODAY are listed — these are the rows the decorator can
// derive a workspace for and therefore shadow-DENY (or, under AUTHZ_ENFORCE,
// actually deny) on a cross-tenant by-id access.
//
// Deliberately ABSENT (kept shadow-PASS only — no safe single-hop probe yet):
//   - loan_payment: loan_id → loan, but `loan` is itself still column-less
//     (two-hop loan.account_id → account.workspace_id required; the immediate
//     parent lacks the column). Stays shadow-PASS until `loan` migrates.
//   - petty_cash_replenishment / petty_cash_voucher: fund_id → petty_cash_fund,
//     parent still column-less. Stays shadow-PASS until petty_cash_fund migrates.
//   - petty_cash_fund: only an OPTIONAL location_id → location.workspace_id
//     (custodian_id is GLOBAL). A location-less fund has no anchor; a location-JOIN
//     would silently drop legitimate funds → false-positive deny in enforce mode.
//     w0-design.md §3.1 marks it ambiguous / A-leg-migration. NOT probed.
//   - collection_schedule / disbursement_schedule: proto-only, NO baseline DDL
//     table → tableHasWorkspaceColumn never matches and the probe would be a
//     runtime no-op. NOT probed.
//
// treasury_disbursement, expenditure_line_item, journal_line, security_deposit are
// probe-able NOW precisely because their parents (expenditure / journal_entry /
// account) JUST received the column in the same esqyma migration wave.
//
// NULL-anchor semantics: a row whose fkColumn is NULL yields a NULL derived
// workspace (LEFT JOIN, no parent match). This fail-closed-EXCLUDES it (would-be
// deny) — matching the exemplar's `($N=” OR parent.workspace_id=$N)`, which is
// false when the parent row is absent. This correctly denies cross-tenant access
// but also makes a legitimately-parentless tenant row (treasury_collection
// advance/unscheduled with NULL revenue_id; invoice with NULL subscription_id)
// inaccessible under ENFORCE — documented as the accepted fail-closed cost. In
// SHADOW (default) it only emits a log line; behavior is unchanged.
var columnLessTenantParentJoins = map[string]parentJoinProbe{
	"treasury_collection": {
		fkColumn: "revenue_id", childAlias: "c", parentAlias: "p",
		parentTable: "revenue", joinCond: "c.revenue_id = p.id", parentWs: "p.workspace_id",
	},
	"invoice": {
		fkColumn: "subscription_id", childAlias: "c", parentAlias: "p",
		parentTable: "subscription", joinCond: "c.subscription_id = p.id", parentWs: "p.workspace_id",
	},
	"treasury_disbursement": {
		fkColumn: "expenditure_id", childAlias: "c", parentAlias: "p",
		parentTable: "expenditure", joinCond: "c.expenditure_id = p.id", parentWs: "p.workspace_id",
	},
	"expenditure_line_item": {
		fkColumn: "expenditure_id", childAlias: "c", parentAlias: "p",
		parentTable: "expenditure", joinCond: "c.expenditure_id = p.id", parentWs: "p.workspace_id",
	},
	"journal_line": {
		fkColumn: "journal_entry_id", childAlias: "c", parentAlias: "p",
		parentTable: "journal_entry", joinCond: "c.journal_entry_id = p.id", parentWs: "p.workspace_id",
	},
	"security_deposit": {
		fkColumn: "account_id", childAlias: "c", parentAlias: "p",
		parentTable: "account", joinCond: "c.account_id = p.id", parentWs: "p.workspace_id",
	},
	"loan": {
		fkColumn: "account_id", childAlias: "c", parentAlias: "p",
		parentTable: "account", joinCond: "c.account_id = p.id", parentWs: "p.workspace_id",
	},
}

// authzEnforceEnvVar mirrors secondary/auth/rbac/authorizer.go: the runtime flag
// that flips the decorator from SHADOW (log-but-pass-through) to ENFORCE (actually
// deny/filter the cross-tenant row). Default OFF = shadow. Read ONCE at
// construction; restart the process to change modes. User-gated: do NOT flip until
// the AUTHZ_WS_SHADOW_DENY logs have been reviewed for false-positives (NULL-parent
// rows, multi-binding users), same gate as the W0 AUTHZ_RBAC_SHADOW_DENY check.
const authzEnforceEnvVar = "AUTHZ_ENFORCE"

// parseEnforce returns true only for an explicit truthy value (mirrors
// authorizer.go parseEnforce). Anything else — unset, "", "0", "false", a typo —
// keeps the safe shadow posture.
func parseEnforce(v string) bool {
	switch v {
	case "1", "true", "TRUE", "True", "yes", "on":
		return true
	default:
		return false
	}
}

// shadowLogColumnLessTenantPass emits a structured AUTHZ_WS_SHADOW_PASS line when
// a column-less TENANT table (one in columnLessTenantTables) receives a generic
// workspace-bearing op (wsID != "") that the decorator is about to pass through
// WITHOUT a tenant predicate — i.e. the cross-tenant IDOR surface.
//
// This is SHADOW-ONLY: it logs and returns; the caller's behavior is unchanged.
// Genuinely-global column-less tables are not in the set, so they log nothing.
// The op argument is "update" | "delete" | "harddelete" | "read".
func shadowLogColumnLessTenantPass(op, tableName, id, wsID string) {
	if wsID == "" || !columnLessTenantTables[tableName] {
		return
	}
	// Mirrors the W0 AUTHZ_RBAC_SHADOW_DENY shape (pipe-delimited key=val). Each
	// line is a measurement: a real runtime generic op on a column-less tenant
	// table under a known workspace that currently escapes tenant scoping. Grep
	// these in prod logs to validate the static needsMigration set before
	// fail-closing / applying the workspace_id migrations.
	log.Printf("AUTHZ_WS_SHADOW_PASS | mode=SHADOW(passed) | table=%s | op=%s | id=%s | ws=%s",
		tableName, op, id, wsID)
}

// scopeColumnLessByParent is the W2 step-2 instrument for a column-less TENANT
// table accessed by id (Read/Update/Delete/HardDelete) under a known workspace.
// It returns a non-nil error ONLY when the caller must abort the operation —
// which happens exclusively in ENFORCE mode on a confirmed cross-tenant row.
//
// Decision matrix (op is "read"|"update"|"delete"|"harddelete", wsID != ""):
//
//   - Table has NO single-hop parent-JOIN probe (not in
//     columnLessTenantParentJoins — e.g. loan_payment two-hop, petty_cash_* with
//     a column-less or optional parent, the proto-only schedules): we cannot
//     derive a workspace, so fall back to the W2 step-1 behavior — emit
//     AUTHZ_WS_SHADOW_PASS and return nil (pass through). No probe query is run.
//
//   - Probe query ERRORS: in SHADOW we log AUTHZ_WS_SHADOW_PASS (probe
//     unavailable) and return nil — a probe error must NEVER alter behavior in
//     shadow, honoring the ZERO-behavior-change invariant. In ENFORCE we fail
//     CLOSED and return a 404 (fail-closed is the safe direction; we could not
//     confirm ownership).
//
//   - Derived workspace == acting workspace (row belongs to the tenant): no log,
//     return nil (in-tenant access, allowed in both modes).
//
//   - Derived workspace != acting workspace OR no parent row (NULL anchor /
//     orphan FK → cross-tenant or fail-closed-exclude): this is the would-be
//     block. In SHADOW emit AUTHZ_WS_SHADOW_DENY and return nil (pass through —
//     zero behavior change). In ENFORCE log AUTHZ_WS_DENY and return a 404 so the
//     generic op aborts before touching the row.
//
// This mirrors the W0 hasCode posture (authorizer.go): SHADOW logs the would-be
// deny and allows; ENFORCE returns the real verdict; a lookup error is fail-closed
// in enforce but pass-through in shadow.
func (w *WorkspaceAwareOperations) scopeColumnLessByParent(ctx context.Context, op, tableName, id, wsID string) error {
	probe, ok := columnLessTenantParentJoins[tableName]
	if !ok {
		// No single-hop derivation available — preserve W2 step-1 shadow-PASS.
		shadowLogColumnLessTenantPass(op, tableName, id, wsID)
		return nil
	}

	derivedWs, found, err := w.deriveWorkspaceViaParent(ctx, tableName, probe, id)
	if err != nil {
		if w.enforce {
			// ENFORCE: could not confirm ownership → fail closed.
			log.Printf("AUTHZ_WS_DENY | mode=ENFORCE | table=%s | op=%s | id=%s | actingWs=%s | parentJoin=%s.%s->%s.%s | reason=probe_error | error=%v",
				tableName, op, id, wsID, probe.childAlias, probe.fkColumn, probe.parentTable, probe.parentWs, err)
			return model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
		}
		// SHADOW: a probe error must NOT change behavior — pass through unchanged.
		log.Printf("AUTHZ_WS_SHADOW_PASS | mode=SHADOW(passed) | table=%s | op=%s | id=%s | ws=%s | note=probe_error(%v)",
			tableName, op, id, wsID, err)
		return nil
	}

	// In-tenant: derived workspace matches the acting workspace. Allowed in both
	// modes, no log line (only cross-tenant rows are interesting).
	if found && derivedWs == wsID {
		return nil
	}

	// Cross-tenant (derived != acting) OR fail-closed-exclude (no parent row /
	// NULL anchor). This is the would-be block.
	parentJoinDesc := probe.childAlias + "." + probe.fkColumn + "->" + probe.parentTable + "." + probe.parentWs
	if !found {
		derivedWs = "<null-anchor>"
	}

	if w.enforce {
		log.Printf("AUTHZ_WS_DENY | mode=ENFORCE | table=%s | op=%s | id=%s | actingWs=%s | derivedWs=%s | parentJoin=%s | wouldDeny=true",
			tableName, op, id, wsID, derivedWs, parentJoinDesc)
		return model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
	}

	// SHADOW (default): log the would-be deny, but pass through so nothing breaks.
	// Every AUTHZ_WS_SHADOW_DENY is a site where ENFORCE mode WOULD have blocked a
	// cross-tenant by-id access. Grep these before flipping AUTHZ_ENFORCE on.
	log.Printf("AUTHZ_WS_SHADOW_DENY | mode=SHADOW(allowed) | table=%s | op=%s | id=%s | actingWs=%s | derivedWs=%s | parentJoin=%s | wouldDeny=true",
		tableName, op, id, wsID, derivedWs, parentJoinDesc)
	return nil
}

// deriveWorkspaceViaParent runs the single-hop parent-JOIN probe for one row id and
// returns the derived workspace_id. `found` is false when the LEFT JOIN produced no
// parent row (FK NULL or orphan) OR the child row itself does not exist — both are
// the fail-closed-exclude case. The query is parameterized ($1 = id); the table /
// alias / join fragments come ONLY from the static columnLessTenantParentJoins map
// (never from caller input), so there is no SQL-injection surface.
//
// This is a NET-NEW SQL round-trip on the default (shadow) path for every
// column-less-tenant by-id op that has a probe — accepted as the cost of measuring
// the would-be verdict (risk: see plan). It selects a single scalar and is indexed
// on the child PK, so it is a cheap point lookup; if it ever errors it is handled
// fail-open in shadow / fail-closed in enforce by the caller.
func (w *WorkspaceAwareOperations) deriveWorkspaceViaParent(ctx context.Context, childTable string, probe parentJoinProbe, id string) (string, bool, error) {
	// Assembled shape:
	//   SELECT p.workspace_id
	//   FROM <childTable> c LEFT JOIN <parentTable> p ON <joinCond>
	//   WHERE c.id = $1
	// Every fragment is a static, in-repo constant from columnLessTenantParentJoins
	// (childTable is the map key; alias/table/join/select come from the probe) —
	// the only caller-supplied value is `id`, bound as $1. No injection surface.
	query := "SELECT " + probe.parentWs +
		" FROM " + childTable + " " + probe.childAlias +
		" LEFT JOIN " + probe.parentTable + " " + probe.parentAlias +
		" ON " + probe.joinCond +
		" WHERE " + probe.childAlias + ".id = $1"

	var ws sql.NullString
	err := w.db.QueryRowContext(ctx, query, id).Scan(&ws)
	if err == sql.ErrNoRows {
		// Child row not found by id — no ownership to assert; fail-closed-exclude.
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if !ws.Valid {
		// Parent absent (NULL anchor / orphan FK) → fail-closed-exclude.
		return "", false, nil
	}
	return ws.String, true, nil
}

// WorkspaceAwareOperations is a decorator over DatabaseOperation that
// transparently injects workspace_id filtering derived from the request context.
//
// Every operation checks two preconditions before applying workspace isolation:
//  1. The request context carries a non-empty workspace_id.
//  2. The target table has a workspace_id column (checked via information_schema,
//     result cached per table name).
//
// When either precondition is false the call is forwarded to the inner
// DatabaseOperation unchanged — this makes the decorator safe for
// service-to-service calls and global (non-tenanted) tables.
//
// Usage:
//
//	dbOps := core.NewWorkspaceAwareOperations(db)
type WorkspaceAwareOperations struct {
	inner         interfaces.DatabaseOperation
	db            *sql.DB
	columnCache   map[string]map[string]bool // table name → column name → exists
	columnCacheMu sync.RWMutex
	// enforce gates the W2 step-2 cross-tenant deny/filter. Read ONCE at
	// construction from AUTHZ_ENFORCE (default false = SHADOW). When false the
	// parent-JOIN probe only LOGS the would-be deny (AUTHZ_WS_SHADOW_DENY) and the
	// call passes through unchanged — ZERO behavior change vs the W2 step-1
	// shadow-pass. When true a cross-tenant by-id row is actually denied (404).
	enforce bool
}

// Ensure WorkspaceAwareOperations satisfies the full DatabaseOperation interface
// at compile time.
var _ interfaces.DatabaseOperation = (*WorkspaceAwareOperations)(nil)

// NewWorkspaceAwareOperations returns a DatabaseOperation that wraps a new
// PostgresOperations instance with automatic workspace_id isolation.
func NewWorkspaceAwareOperations(db *sql.DB) interfaces.DatabaseOperation {
	return &WorkspaceAwareOperations{
		inner:       NewPostgresOperations(db),
		db:          db,
		columnCache: make(map[string]map[string]bool),
		enforce:     newWorkspaceEnforce(),
	}
}

// NewWorkspaceAwareOperationsFromInner wraps an existing DatabaseOperation
// with workspace-aware filtering. Use this when you already have an instance
// (e.g. one created with NewPostgresOperationsWithAudit).
func NewWorkspaceAwareOperationsFromInner(db *sql.DB, inner interfaces.DatabaseOperation) interfaces.DatabaseOperation {
	return &WorkspaceAwareOperations{
		inner:       inner,
		db:          db,
		columnCache: make(map[string]map[string]bool),
		enforce:     newWorkspaceEnforce(),
	}
}

// newWorkspaceEnforce reads AUTHZ_ENFORCE once at construction and logs the active
// mode, mirroring secondary/auth/rbac/authorizer.go NewPermissionAuthorizer. The
// decorator and the W0 Authorizer share the SAME flag so a single AUTHZ_ENFORCE=on
// flips both layers together.
func newWorkspaceEnforce() bool {
	enforce := parseEnforce(os.Getenv(authzEnforceEnvVar))
	mode := "SHADOW (log-but-pass)"
	if enforce {
		mode = "ENFORCE (deny cross-tenant)"
	}
	log.Printf("🔐 WorkspaceAwareOperations initialised — mode=%s (AUTHZ_ENFORCE=%q)", mode, os.Getenv(authzEnforceEnvVar))
	return enforce
}

// ── DatabaseOperation methods ────────────────────────────────────────────────

// List delegates to the inner List, optionally prepending a workspace_id
// StringFilter when the context carries a workspace and the table has the
// column.
func (w *WorkspaceAwareOperations) List(ctx context.Context, tableName string, params *interfaces.ListParams) (*interfaces.ListResult, error) {
	wsID := w.getWorkspaceID(ctx)
	if wsID != "" && w.tableHasWorkspaceColumn(ctx, tableName) {
		// Direct-column path. account / expenditure / journal_entry land HERE now
		// that the esqyma migration added their workspace_id column (they were
		// removed from columnLessTenantTables) — the StringFilter predicate scopes
		// them automatically, no parent-JOIN needed.
		params = w.injectWorkspaceFilter(params, wsID)
	} else if wsID != "" && columnLessTenantTables[tableName] {
		// W2 step-2 measurement: a column-less TENANT list returns rows across ALL
		// workspaces (the IDOR surface). The StringFilter injection mechanism cannot
		// express a parent-JOIN predicate, so List is SHADOW-only here in BOTH
		// modes — it logs the unscoped list-by-tenant but does NOT filter (the
		// per-row by-id probe in Read/Update/Delete/HardDelete is the enforce-able
		// surface; List enforce-filtering needs a query-builder JOIN rewrite,
		// tracked as a W2 follow-up). Zero behavior change either way.
		log.Printf("AUTHZ_WS_SHADOW_PASS | mode=SHADOW(passed) | table=%s | op=list | id= | ws=%s",
			tableName, wsID)
	}
	return w.inner.List(ctx, tableName, params)
}

// Create injects workspace_id into the data map before inserting, when the
// context carries a workspace and the table has the column.
func (w *WorkspaceAwareOperations) Create(ctx context.Context, tableName string, data map[string]any) (map[string]any, error) {
	wsID := w.getWorkspaceID(ctx)
	if wsID != "" && w.tableHasWorkspaceColumn(ctx, tableName) {
		// Clone the map to avoid mutating the caller's data.
		cloned := make(map[string]any, len(data)+1)
		for k, v := range data {
			cloned[k] = v
		}
		cloned["workspace_id"] = wsID
		data = cloned
	}
	return w.inner.Create(ctx, tableName, data)
}

// Read delegates to the inner Read, then verifies that the returned record
// belongs to the context workspace. Returns a 404 if the workspace_id does
// not match, preventing cross-workspace data leakage via direct-ID lookup.
//
// NULL-row rejection (Phase 1.5 — 2026-05-10 — codex C1):
// When the context carries a workspace and the table has a workspace_id column,
// a row whose workspace_id is NULL or empty is also rejected as 404. Without
// this check, legacy NULL rows left by the Phase 1 migration are visible across
// all workspaces via direct-ID reads, since any non-NULL != wsID comparison
// returns false (the previous condition only fired on non-nil mismatches).
// Update and Delete already call Read for ownership verification, so they
// inherit this fix automatically.
func (w *WorkspaceAwareOperations) Read(ctx context.Context, tableName string, id string) (map[string]any, error) {
	result, err := w.inner.Read(ctx, tableName, id)
	if err != nil {
		return nil, err
	}

	wsID := w.getWorkspaceID(ctx)
	if wsID == "" {
		return result, nil
	}

	// Only apply direct-column workspace enforcement when the table has the column.
	// Tables without workspace_id are not directly tenant-scoped.
	if !w.tableHasWorkspaceColumn(ctx, tableName) {
		// W2 step-2: if this column-less table is a TENANT table (IDOR surface),
		// derive its owning workspace via the parent-JOIN probe. SHADOW (default)
		// logs AUTHZ_WS_SHADOW_DENY / _PASS and returns the result unchanged;
		// ENFORCE returns a 404 for a cross-tenant row. Genuinely-global tables are
		// not in columnLessTenantTables → no probe, no log, plain pass-through.
		if columnLessTenantTables[tableName] {
			if err := w.scopeColumnLessByParent(ctx, "read", tableName, id, wsID); err != nil {
				return nil, err
			}
		}
		return result, nil
	}

	recordWsID, hasCol := result["workspace_id"]
	if !hasCol {
		// Column not present in result set — defensive pass-through; should not
		// happen for tables that tableHasWorkspaceColumn confirmed.
		return result, nil
	}

	// Reject both NULL workspace_id rows AND rows that belong to a different workspace.
	// A NULL workspace_id means the row pre-dates the tenancy migration and has not
	// been backfilled; treating it as "accessible by anyone" would be a tenancy leak.
	if recordWsID == nil {
		return nil, model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
	}
	if recordWsIDStr, ok := recordWsID.(string); ok {
		if recordWsIDStr == "" || recordWsIDStr != wsID {
			return nil, model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
		}
	}

	return result, nil
}

// Update verifies workspace ownership via a Read before delegating to the
// inner Update. It also strips any workspace_id key from the data payload to
// prevent cross-workspace reassignment.
func (w *WorkspaceAwareOperations) Update(ctx context.Context, tableName string, id string, data map[string]any) (map[string]any, error) {
	wsID := w.getWorkspaceID(ctx)
	if wsID != "" && w.tableHasWorkspaceColumn(ctx, tableName) {
		// Verify ownership — reuse Read which already enforces workspace check.
		if _, err := w.Read(ctx, tableName, id); err != nil {
			return nil, err
		}
		// Strip workspace_id from the update payload; it must never change.
		cloned := make(map[string]any, len(data))
		for k, v := range data {
			if k != "workspace_id" {
				cloned[k] = v
			}
		}
		data = cloned
	} else if wsID != "" && columnLessTenantTables[tableName] {
		// W2 step-2: column-less TENANT table (IDOR surface). Derive the owning
		// workspace via parent-JOIN; SHADOW logs (AUTHZ_WS_SHADOW_DENY / _PASS) and
		// passes through, ENFORCE returns 404 for a cross-tenant row. Global
		// column-less tables are not in the set → untouched.
		if err := w.scopeColumnLessByParent(ctx, "update", tableName, id, wsID); err != nil {
			return nil, err
		}
	}
	return w.inner.Update(ctx, tableName, id, data)
}

// Delete verifies workspace ownership via a Read, then delegates the soft
// delete to the inner operation.
func (w *WorkspaceAwareOperations) Delete(ctx context.Context, tableName string, id string) error {
	wsID := w.getWorkspaceID(ctx)
	if wsID != "" && w.tableHasWorkspaceColumn(ctx, tableName) {
		if _, err := w.Read(ctx, tableName, id); err != nil {
			return err
		}
	} else if wsID != "" && columnLessTenantTables[tableName] {
		// W2 step-2: column-less TENANT table (IDOR surface). SHADOW logs and
		// passes through; ENFORCE returns 404 for a cross-tenant row.
		if err := w.scopeColumnLessByParent(ctx, "delete", tableName, id, wsID); err != nil {
			return err
		}
	}
	return w.inner.Delete(ctx, tableName, id)
}

// HardDelete verifies workspace ownership via a Read, then delegates the
// permanent delete to the inner operation.
func (w *WorkspaceAwareOperations) HardDelete(ctx context.Context, tableName string, id string) error {
	wsID := w.getWorkspaceID(ctx)
	if wsID != "" && w.tableHasWorkspaceColumn(ctx, tableName) {
		if _, err := w.Read(ctx, tableName, id); err != nil {
			return err
		}
	} else if wsID != "" && columnLessTenantTables[tableName] {
		// W2 step-2: column-less TENANT table (IDOR surface). SHADOW logs and
		// passes through; ENFORCE returns 404 for a cross-tenant row.
		if err := w.scopeColumnLessByParent(ctx, "harddelete", tableName, id, wsID); err != nil {
			return err
		}
	}
	return w.inner.HardDelete(ctx, tableName, id)
}

// Query passes through to the inner operation. Injecting workspace filters
// into QueryBuilder is non-trivial; callers that use Query are expected to
// include workspace filtering themselves.
func (w *WorkspaceAwareOperations) Query(ctx context.Context, tableName string, query interfaces.QueryBuilder) ([]map[string]any, error) {
	return w.inner.Query(ctx, tableName, query)
}

// QueryOne passes through to the inner operation (see Query).
func (w *WorkspaceAwareOperations) QueryOne(ctx context.Context, tableName string, query interfaces.QueryBuilder) (map[string]any, error) {
	return w.inner.QueryOne(ctx, tableName, query)
}

// ── Optional interface methods (type-asserted by adapters) ───────────────────

// GetDB returns the underlying *sql.DB so that adapters performing raw SQL
// (CTEs, JOINs) can obtain a connection via the standard type assertion:
//
//	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok { ... }
func (w *WorkspaceAwareOperations) GetDB() *sql.DB {
	return w.db
}

// GetExecutor returns the transaction-aware executor from the inner operation.
// Adapters that type-assert to an executorProvider interface use this to
// participate in active transactions.
func (w *WorkspaceAwareOperations) GetExecutor(ctx context.Context) interfaces.DBExecutor {
	type executorProvider interface {
		GetExecutor(ctx context.Context) interfaces.DBExecutor
	}
	if ep, ok := w.inner.(executorProvider); ok {
		return ep.GetExecutor(ctx)
	}
	return w.db
}

// ── Helper methods ───────────────────────────────────────────────────────────

// getWorkspaceID extracts the workspace_id from the request context.
// Returns an empty string if no workspace is present (e.g. service-to-service
// calls or unauthenticated contexts), which disables all workspace injection.
func (w *WorkspaceAwareOperations) getWorkspaceID(ctx context.Context) string {
	return consumer.GetWorkspaceIDFromContext(ctx)
}

// tableHasWorkspaceColumn reports whether tableName has a workspace_id column.
// Results are cached with a read-preferred RWMutex; the first miss for a table
// performs a live query against information_schema.columns.
func (w *WorkspaceAwareOperations) tableHasWorkspaceColumn(ctx context.Context, tableName string) bool {
	w.columnCacheMu.RLock()
	cols, cached := w.columnCache[tableName]
	w.columnCacheMu.RUnlock()

	if cached {
		return cols["workspace_id"]
	}

	// Cache miss — query the schema. Use the underlying db directly to avoid
	// a potential recursive call through the decorated operation.
	query := `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_name = $1
		ORDER BY ordinal_position
	`
	rows, err := w.db.QueryContext(ctx, query, tableName)
	if err != nil {
		// On error, conservatively skip injection rather than blocking the call.
		return false
	}
	defer rows.Close()

	colMap := make(map[string]bool)
	for rows.Next() {
		var colName string
		if err := rows.Scan(&colName); err != nil {
			continue
		}
		colMap[colName] = true
	}
	if rows.Err() != nil {
		return false
	}

	w.columnCacheMu.Lock()
	w.columnCache[tableName] = colMap
	w.columnCacheMu.Unlock()

	return colMap["workspace_id"]
}

// injectWorkspaceFilter returns a copy of params with a workspace_id
// StringFilter prepended. The original params value is never mutated.
// If params is nil a new ListParams is allocated.
func (w *WorkspaceAwareOperations) injectWorkspaceFilter(params *interfaces.ListParams, wsID string) *interfaces.ListParams {
	wsFilter := &commonpb.TypedFilter{
		Field: "workspace_id",
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{
				Value:         wsID,
				Operator:      commonpb.StringOperator_STRING_EQUALS,
				CaseSensitive: true,
			},
		},
	}

	if params == nil {
		return &interfaces.ListParams{
			Filters: &commonpb.FilterRequest{
				Filters: []*commonpb.TypedFilter{wsFilter},
			},
		}
	}

	// Clone ListParams shallowly, deep-copy only the Filters slice.
	cloned := *params
	if cloned.Filters == nil {
		cloned.Filters = &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{wsFilter},
		}
	} else {
		newFilters := make([]*commonpb.TypedFilter, 0, len(cloned.Filters.Filters)+1)
		newFilters = append(newFilters, wsFilter)
		newFilters = append(newFilters, cloned.Filters.Filters...)
		cloned.Filters = &commonpb.FilterRequest{
			Filters: newFilters,
			Logic:   cloned.Filters.Logic,
		}
	}

	return &cloned
}
