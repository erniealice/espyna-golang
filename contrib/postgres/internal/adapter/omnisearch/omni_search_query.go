//go:build postgresql

// Package omnisearch is the postgres adapter for the service/omni_search read
// (the ⌘K command palette). It implements the GENERATED
// omni_searchv1.OmniSearchServiceServer interface and self-registers via the
// composition-root factory registry (mirrors operation/outcome_matrix_query.go).
//
// SECURITY (design spine, docs/plan/20260710-omni-search + _audit-adherence.md):
//   - Every category arm carries the workspace predicate on the BASE table AND
//     on every JOINED tenant alias (S1 / HAZ-02): the subscription arm binds the
//     joined client c.workspace_id and the subscription_group arm binds the
//     joined price_schedule ps.workspace_id.
//   - Workspace is ctx-derived (identity.FromContext), never a request param (S2).
//   - Person-scoped rows carry principalscope reachability: the client arm scopes
//     c via StaffReachableClientClause, and the subscription arm scopes its JOINED
//     client c the same way (S6 / codex blocker #1 — joined client labels are
//     person data).
//   - The user query is LIKE-escaped (%/_/\) and matched with ILIKE ... ESCAPE
//     '\' (S5), so "%%" returns nothing rather than the first rows of every
//     permitted category.
//   - Table identifiers come exclusively from registry/entityid constants; column
//     allowlists are author-literal (never request-derived); ORDER BY is fixed.
//
// The use case has already gated req.Categories down to the permitted set; this
// adapter queries exactly those keys (an unknown key is defensively skipped).
package omnisearch

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	omnisearchpb "github.com/erniealice/esqyma/pkg/schema/v1/service/omni_search"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	internalregistry "github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// init self-registers the postgres omni-search query with the composition-root
// factory registry (mirrors operation/outcome_matrix_query.go). The registry
// file is tag-free; only THIS file (build-tagged postgresql) calls Register, so
// non-postgres builds never wire it and the composition initializer degrades to
// a nil port (fail-closed empty response).
func init() {
	internalregistry.RegisterOmniSearchFactory(func(db any) any {
		sqlDB, ok := db.(*sql.DB)
		if !ok {
			return nil
		}
		return NewPostgresOmniSearchQuery(sqlDB)
	})
}

// PostgresOmniSearchQuery implements the GENERATED
// omni_searchv1.OmniSearchServiceServer interface (Q-PROTO-MODE: service{rpc}).
// Embedding UnimplementedOmniSearchServiceServer is MANDATORY — the interface
// carries an unexported marker method only the Unimplemented struct can satisfy.
type PostgresOmniSearchQuery struct {
	omnisearchpb.UnimplementedOmniSearchServiceServer
	db *sql.DB
}

// NewPostgresOmniSearchQuery constructs the PG-backed omni-search reader.
func NewPostgresOmniSearchQuery(db *sql.DB) omnisearchpb.OmniSearchServiceServer {
	return &PostgresOmniSearchQuery{db: db}
}

// defaultLimitPerCategory backstops the per-category cap when the (already
// use-case-clamped) request carries no positive limit.
const defaultLimitPerCategory int32 = 5

// SearchEntities runs one parameterized ILIKE arm per permitted category and
// returns the non-empty groups in registry order. Workspace scope is derived
// from the SESSION identity (never a request param); a missing identity fails
// closed to an empty, successful response.
func (a *PostgresOmniSearchQuery) SearchEntities(
	ctx context.Context,
	req *omnisearchpb.OmniSearchRequest,
) (*omnisearchpb.OmniSearchResponse, error) {
	if req == nil || req.GetQuery() == "" || len(req.GetCategories()) == 0 {
		return &omnisearchpb.OmniSearchResponse{Success: true}, nil
	}

	// workspace_id from the session identity — required for multi-tenancy.
	// FromContext (not identity.Must) so a missing identity fails closed to an
	// empty response rather than panicking.
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil || id.WorkspaceID == "" {
		return &omnisearchpb.OmniSearchResponse{Success: true}, nil
	}
	workspaceID := id.WorkspaceID

	limit := defaultLimitPerCategory
	if req.LimitPerCategory != nil && req.GetLimitPerCategory() > 0 {
		limit = req.GetLimitPerCategory()
	}

	pattern := "%" + escapeLike(req.GetQuery()) + "%"

	var categories []*omnisearchpb.OmniSearchCategoryResults
	for _, key := range req.GetCategories() {
		build, ok := categoryBuilders[key]
		if !ok {
			// Unknown key — defensively skipped (the use case validates the set;
			// the adapter never queries a table it has no author-literal spec for).
			continue
		}
		query, args := build(ctx, pattern, limit, workspaceID)
		results, err := a.runCategory(ctx, query, args)
		if err != nil {
			return nil, err
		}
		if len(results) == 0 {
			continue
		}
		categories = append(categories, &omnisearchpb.OmniSearchCategoryResults{
			Category: key,
			Results:  results,
		})
	}

	return &omnisearchpb.OmniSearchResponse{Success: true, Categories: categories}, nil
}

// runCategory executes one category arm and scans its (id, label, sublabel) rows.
// Every category SQL projects exactly those three columns in that order.
func (a *PostgresOmniSearchQuery) runCategory(ctx context.Context, query string, args []any) ([]*omnisearchpb.OmniSearchResult, error) {
	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("omni_search: category query: %w", err)
	}
	defer rows.Close()

	var out []*omnisearchpb.OmniSearchResult
	for rows.Next() {
		var id, label, sublabel string
		if err := rows.Scan(&id, &label, &sublabel); err != nil {
			return nil, fmt.Errorf("omni_search: scan result: %w", err)
		}
		r := &omnisearchpb.OmniSearchResult{Id: id, Label: label}
		if sublabel != "" {
			sl := sublabel
			r.Sublabel = &sl
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("omni_search: result rows: %w", err)
	}
	return out, nil
}

// escapeLike escapes the SQL LIKE/ILIKE wildcards (\, %, _) in the user query so
// they are matched literally under ILIKE ... ESCAPE '\'. Without this, a query of
// "%%" or "__" would match the first rows of every permitted category past the
// min-length rule (codex finding #4). Backslash is escaped FIRST so an escape
// char introduced for % / _ is not itself re-escaped.
func escapeLike(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 8)
	for _, r := range s {
		switch r {
		case '\\', '%', '_':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// categoryBuild builds one category's parameterized SQL + positional args. The
// arg order is FIXED across every category: $1 = the LIKE pattern, $2 = the
// per-category LIMIT, $3 = the session workspace_id; principalscope args (if any)
// follow at $4+. Keeping the base three positions identical lets the shape test
// assert the predicates uniformly.
type categoryBuild func(ctx context.Context, pattern string, limit int32, workspaceID string) (string, []any)

// categoryBuilders maps the (already permission-gated) category key to its
// author-literal SQL spec. Keys mirror the espyna usecases/service/omnisearch
// registry verbatim.
var categoryBuilders = map[string]categoryBuild{
	"client":             buildClient,
	"subscription":       buildSubscription,
	"subscription_group": buildSubscriptionGroup,
	"plan":               buildPlan,
	"price_schedule":     buildPriceSchedule,
	"product":            buildProduct,
	"staff":              buildStaff,
}

// buildClient — search client.name / client.internal_id / joined user first/last;
// label = COALESCE(name, trimmed user name, id); sublabel = internal_id. Scoped
// by workspace ($3) + StaffReachableClientClause on c (person data — a teacher
// persona palette-finds only reachable clients). The joined user table is global
// (FK by user_id) and carries no workspace_id, so it needs no tenant predicate.
func buildClient(ctx context.Context, pattern string, limit int32, workspaceID string) (string, []any) {
	args := []any{pattern, limit, workspaceID}
	clientScope, clientScopeArgs := principalscope.StaffReachableClientClause(ctx, "c", 4)
	q := `
SELECT
    c.id,
    COALESCE(
        NULLIF(c.name, ''),
        NULLIF(TRIM(CONCAT(u.first_name, ' ', u.last_name)), ''),
        c.id
    ) AS label,
    COALESCE(c.internal_id, '') AS sublabel
FROM ` + entityid.Client + ` c
LEFT JOIN "` + entityid.User + `" u ON c.user_id = u.id
WHERE c.workspace_id = $3
  AND c.active = true
  AND (
        c.name ILIKE $1 ESCAPE '\'
     OR c.internal_id ILIKE $1 ESCAPE '\'
     OR u.first_name ILIKE $1 ESCAPE '\'
     OR u.last_name ILIKE $1 ESCAPE '\'
  )` + clientScope + `
ORDER BY label ASC
LIMIT $2`
	args = append(args, clientScopeArgs...)
	return q, args
}

// buildSubscription — search subscription.name / subscription.code / joined
// client name; label = COALESCE(name, code, id); sublabel = client label. The
// joined client alias c is a TENANT table, so it carries BOTH its own workspace
// bound (c.workspace_id = $3) AND principalscope reachability (codex blocker #1:
// joined client labels are person data — reachability applies to any category
// that searches/renders them). The base subscription row is workspace-bound too.
func buildSubscription(ctx context.Context, pattern string, limit int32, workspaceID string) (string, []any) {
	args := []any{pattern, limit, workspaceID}
	// Reachability applies to the JOINED client alias c (its labels are person
	// data), mirroring the client arm — a staff persona only surfaces
	// subscriptions whose client it can reach.
	clientScope, clientScopeArgs := principalscope.StaffReachableClientClause(ctx, "c", 4)
	q := `
SELECT
    s.id,
    COALESCE(NULLIF(s.name, ''), NULLIF(s.code, ''), s.id) AS label,
    COALESCE(
        NULLIF(c.name, ''),
        NULLIF(TRIM(CONCAT(u.first_name, ' ', u.last_name)), ''),
        ''
    ) AS sublabel
FROM ` + entityid.Subscription + ` s
JOIN ` + entityid.Client + ` c
       ON c.id = s.client_id AND c.workspace_id = $3 AND c.active = true
LEFT JOIN "` + entityid.User + `" u ON c.user_id = u.id
WHERE s.workspace_id = $3
  AND s.active = true
  AND (
        s.name ILIKE $1 ESCAPE '\'
     OR s.code ILIKE $1 ESCAPE '\'
     OR c.name ILIKE $1 ESCAPE '\'
     OR u.first_name ILIKE $1 ESCAPE '\'
     OR u.last_name ILIKE $1 ESCAPE '\'
  )` + clientScope + `
ORDER BY label ASC
LIMIT $2`
	args = append(args, clientScopeArgs...)
	return q, args
}

// buildSubscriptionGroup — search subscription_group.name; label = name;
// sublabel = joined price_schedule name. The joined price_schedule alias ps is a
// TENANT table, so its workspace is bound in the LEFT JOIN's ON clause
// (ps.workspace_id = $3) — a foreign-workspace price_schedule id must never
// surface a sublabel.
func buildSubscriptionGroup(_ context.Context, pattern string, limit int32, workspaceID string) (string, []any) {
	args := []any{pattern, limit, workspaceID}
	q := `
SELECT
    sg.id,
    sg.name AS label,
    COALESCE(ps.name, '') AS sublabel
FROM ` + entityid.SubscriptionGroup + ` sg
LEFT JOIN ` + entityid.PriceSchedule + ` ps
       ON ps.id = sg.price_schedule_id AND ps.workspace_id = $3 AND ps.active = true
WHERE sg.workspace_id = $3
  AND sg.active = true
  AND sg.name ILIKE $1 ESCAPE '\'
ORDER BY label ASC
LIMIT $2`
	return q, args
}

// buildPlan — search plan.name; label = name; no sublabel.
func buildPlan(_ context.Context, pattern string, limit int32, workspaceID string) (string, []any) {
	args := []any{pattern, limit, workspaceID}
	q := `
SELECT p.id, p.name AS label, '' AS sublabel
FROM ` + entityid.Plan + ` p
WHERE p.workspace_id = $3
  AND p.active = true
  AND p.name ILIKE $1 ESCAPE '\'
ORDER BY label ASC
LIMIT $2`
	return q, args
}

// buildPriceSchedule — search price_schedule.name; label = name; no sublabel.
func buildPriceSchedule(_ context.Context, pattern string, limit int32, workspaceID string) (string, []any) {
	args := []any{pattern, limit, workspaceID}
	q := `
SELECT prs.id, prs.name AS label, '' AS sublabel
FROM ` + entityid.PriceSchedule + ` prs
WHERE prs.workspace_id = $3
  AND prs.active = true
  AND prs.name ILIKE $1 ESCAPE '\'
ORDER BY label ASC
LIMIT $2`
	return q, args
}

// buildProduct — search product.name; label = name; sublabel = the comma-joined
// LEVELS (plan names) the product is offered at (owner request, wave-2), replacing
// the wave-1 product_kind sublabel. The levels come from a correlated subquery over
// product_plan pp → plan pl (product→plans is 1-to-MANY). Workspace scoping
// (codex-impl HIGH #1, applied correctly): product_plan is a pure JUNCTION table
// with NO workspace_id column, so it is NOT a tenant alias and CANNOT be bound;
// it is transitively tenant-scoped because BOTH its endpoints are workspace-bound —
// the outer product (pr.workspace_id = $3) via pp.product_id = pr.id, and the plan
// (pl.workspace_id = $3) via the join. A pp row bridging to a foreign plan is
// dropped by pl.workspace_id; a foreign product never matches pp.product_id = pr.id.
// A product with zero plans yields an empty sublabel (the COALESCE turns the NULL
// string_agg into ''), never the old "service".
func buildProduct(_ context.Context, pattern string, limit int32, workspaceID string) (string, []any) {
	args := []any{pattern, limit, workspaceID}
	q := `
SELECT
    pr.id,
    pr.name AS label,
    COALESCE(
        (
            SELECT string_agg(DISTINCT pl.name, ', ' ORDER BY pl.name)
            FROM ` + entityid.ProductPlan + ` pp
            JOIN ` + entityid.Plan + ` pl
                   ON pl.id = pp.plan_id AND pl.workspace_id = $3 AND pl.active = true
            WHERE pp.product_id = pr.id AND pp.active = true
        ),
        ''
    ) AS sublabel
FROM ` + entityid.Product + ` pr
WHERE pr.workspace_id = $3
  AND pr.active = true
  AND pr.name ILIKE $1 ESCAPE '\'
ORDER BY label ASC
LIMIT $2`
	return q, args
}

// buildStaff — search the STAFF category (wave-2), mirroring the client arm's
// joined-user pattern. staff has NO name columns; the searchable name lives on the
// GLOBAL user table (staff.user_id → user.first_name/last_name), so the arm ILIKEs
// the joined user name and sublabels the joined user.email_address.
//
// Result id = the staff's workspace_user.id (NOT staff.id): staff has no detail
// page, so a staff row links to the workspace_user detail page. workspace_user is
// resolved by an INNER JOIN staff→workspace_user on user_id (unique_together
// workspace_id,user_id makes it 1:1 within a workspace; all staff carry a
// workspace_user row). That INNER join also guarantees wu.id is non-null.
//
// Workspace binding (codex-impl HIGH #1): staff (s.workspace_id = $3) AND the
// joined TENANT alias workspace_user (wu.workspace_id = $3) are both bound. The
// joined "user" alias u is a GLOBAL table with NO workspace_id column, so it is
// deliberately left UNBOUND — exactly like the client arm's user join. (The base
// staff row's workspace bind + the FK chain already tenant-scope which user rows
// are reachable; user itself has no tenant column to bind.)
func buildStaff(_ context.Context, pattern string, limit int32, workspaceID string) (string, []any) {
	args := []any{pattern, limit, workspaceID}
	// DISTINCT guards against duplicate active staff rows sharing one workspace_user
	// (staff has no (workspace_id,user_id) uniqueness) emitting the same (id,label,
	// sublabel) row twice — plain DISTINCT (not DISTINCT ON) so it composes with the
	// ORDER BY label. Live invariant today: 0 dup (workspace_id,user_id) and 0 active
	// staff with NULL workspace_id on education1 (backfill complete) — the
	// s.workspace_id bind drops no live staff; the DISTINCT is a forward guard.
	q := `
SELECT DISTINCT
    wu.id,
    COALESCE(NULLIF(TRIM(CONCAT(u.first_name, ' ', u.last_name)), ''), '') AS label,
    COALESCE(u.email_address, '') AS sublabel
FROM ` + entityid.Staff + ` s
JOIN ` + entityid.WorkspaceUser + ` wu
       ON wu.user_id = s.user_id AND wu.workspace_id = $3
JOIN "` + entityid.User + `" u ON u.id = s.user_id
WHERE s.workspace_id = $3
  AND s.active = true
  AND (
        u.first_name ILIKE $1 ESCAPE '\'
     OR u.last_name ILIKE $1 ESCAPE '\'
  )
ORDER BY label ASC
LIMIT $2`
	return q, args
}
