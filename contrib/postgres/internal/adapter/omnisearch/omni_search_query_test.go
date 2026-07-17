//go:build postgresql

package omnisearch

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// staffCtx returns a context carrying a STAFF principal identity so the
// principalscope.StaffReachableClientClause helpers emit their reachability
// predicate (a non-staff / no-identity ctx leaves the query unscoped, which
// would hide the very clause these tests assert).
func staffCtx() context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{
		WorkspaceID:   "ws-1",
		PrincipalType: principalTypeStaffForTest,
		PrincipalID:   "staff-1",
	})
}

// principalTypeStaffForTest mirrors principalscope.PrincipalTypeStaff (7) — the
// esqyma PrincipalType PRINCIPAL_TYPE_STAFF integer. Kept local so the test does
// not depend on the unexported const's package path.
const principalTypeStaffForTest int32 = 7

// TestEveryCategoryIsWorkspaceBound pins S1 (HAZ-02 regression guard): every
// category arm carries workspace_id = $3 on its base table, iterated over the
// full builder registry so a new category cannot silently ship unbound.
func TestEveryCategoryIsWorkspaceBound(t *testing.T) {
	ctx := staffCtx()
	baseCol := map[string]string{
		"client":             "c.workspace_id = $3",
		"subscription":       "s.workspace_id = $3",
		"subscription_group": "sg.workspace_id = $3",
		"plan":               "p.workspace_id = $3",
		"price_schedule":     "prs.workspace_id = $3",
		"product":            "pr.workspace_id = $3",
		"staff":              "s.workspace_id = $3",
	}
	for key, build := range categoryBuilders {
		sql, _ := build(ctx, "%q%", 5, "ws-1")
		want, ok := baseCol[key]
		if !ok {
			t.Fatalf("category %q has no expected base workspace predicate registered in the test", key)
		}
		if !strings.Contains(sql, want) {
			t.Errorf("category %q missing base workspace predicate %q:\n%s", key, want, sql)
		}
	}
	// The two tables are exhaustive-checked in lock-step: if a builder is added
	// without a baseCol entry the loop above t.Fatals; guard the reverse too.
	if len(baseCol) != len(categoryBuilders) {
		t.Errorf("baseCol map (%d) out of sync with categoryBuilders (%d)", len(baseCol), len(categoryBuilders))
	}
}

// TestJoinedTenantAliasesAreWorkspaceBound pins S1's amendment: a JOINED tenant
// alias must carry its own workspace predicate, not just the base table (codex:
// a subscription→client join binds the joined client's workspace too).
func TestJoinedTenantAliasesAreWorkspaceBound(t *testing.T) {
	ctx := staffCtx()

	// subscription joins client c — c must be workspace-bound.
	subSQL, _ := buildSubscription(ctx, "%q%", 5, "ws-1")
	if !strings.Contains(subSQL, "c.workspace_id = $3") {
		t.Errorf("subscription arm: joined client alias c not workspace-bound:\n%s", subSQL)
	}
	// The join to client must be an INNER join (an unbound LEFT join would leak
	// rows whose client is foreign/absent).
	if !strings.Contains(subSQL, "JOIN "+entityid.Client+" c") ||
		strings.Contains(subSQL, "LEFT JOIN "+entityid.Client+" c") {
		t.Errorf("subscription arm: client must be INNER-joined (workspace-bound):\n%s", subSQL)
	}

	// subscription_group joins price_schedule ps — ps must be workspace-bound in
	// its ON clause.
	sgSQL, _ := buildSubscriptionGroup(ctx, "%q%", 5, "ws-1")
	if !strings.Contains(sgSQL, "ps.workspace_id = $3") {
		t.Errorf("subscription_group arm: joined price_schedule alias ps not workspace-bound:\n%s", sgSQL)
	}

	// product's levels subquery joins product_plan pp → plan pl. product_plan is a
	// pure JUNCTION table with NO workspace_id column, so it CANNOT be bound; it is
	// transitively tenant-scoped by its two workspace-bound endpoints — the outer
	// product (pr.workspace_id) via pp.product_id = pr.id, and the plan
	// (pl.workspace_id) via the join. Assert the plan bind + the transitive tie, and
	// assert pp is NOT falsely workspace-bound (would be a runtime "column does not
	// exist" — the live bug this test now guards against regressing).
	prodSQL, _ := buildProduct(ctx, "%q%", 5, "ws-1")
	if strings.Contains(prodSQL, "pp.workspace_id") {
		t.Errorf("product arm: product_plan (junction, no workspace_id column) must NOT be bound:\n%s", prodSQL)
	}
	if !strings.Contains(prodSQL, "pl.workspace_id = $3") {
		t.Errorf("product arm: levels subquery plan alias pl not workspace-bound:\n%s", prodSQL)
	}
	if !strings.Contains(prodSQL, "pp.product_id = pr.id") {
		t.Errorf("product arm: junction not tied to the workspace-bound outer product:\n%s", prodSQL)
	}

	// staff joins workspace_user wu (for the result id) — wu is a tenant alias and
	// must be workspace-bound in its ON clause.
	staffSQL, _ := buildStaff(ctx, "%q%", 5, "ws-1")
	if !strings.Contains(staffSQL, "wu.workspace_id = $3") {
		t.Errorf("staff arm: joined workspace_user alias wu not workspace-bound:\n%s", staffSQL)
	}
}

// TestProductSublabelIsWorkspaceBoundLevelJoin pins the wave-2 product sublabel
// change: the sublabel is the string_agg of the joined plan (level) names. The
// plan alias pl is workspace-bound; the product_plan junction (no workspace_id
// column) is transitively scoped via pp.product_id = pr.id, NOT a false pp bind.
// The old product_kind sublabel must be gone.
func TestProductSublabelIsWorkspaceBoundLevelJoin(t *testing.T) {
	sql, _ := buildProduct(staffCtx(), "%q%", 5, "ws-1")
	for _, want := range []string{
		"string_agg(DISTINCT pl.name",
		entityid.ProductPlan + " pp",
		"JOIN " + entityid.Plan + " pl",
		"pl.workspace_id = $3",
		"pp.product_id = pr.id",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("product sublabel arm missing %q:\n%s", want, sql)
		}
	}
	if strings.Contains(sql, "pp.workspace_id") {
		t.Errorf("product arm falsely binds the workspace_id-less product_plan junction:\n%s", sql)
	}
	if strings.Contains(sql, "product_kind") {
		t.Errorf("product arm still emits the old product_kind sublabel:\n%s", sql)
	}
}

// TestStaffArmSearchesJoinedUserName pins the wave-2 staff arm: it is
// workspace-bound on the base staff row, searches the joined (global) user name
// with escaped ILIKE, and emits the workspace_user.id as the result id.
func TestStaffArmSearchesJoinedUserName(t *testing.T) {
	sql, args := buildStaff(staffCtx(), "%q%", 5, "ws-1")

	if !strings.Contains(sql, "s.workspace_id = $3") {
		t.Errorf("staff arm not workspace-bound on the base staff row:\n%s", sql)
	}
	// Result id column is the workspace_user id (staff has no detail page).
	// DISTINCT dedups staff rows sharing one workspace_user (no uniqueness on staff).
	if !strings.Contains(sql, "SELECT DISTINCT\n    wu.id,") {
		t.Errorf("staff arm must project DISTINCT wu.id (workspace_user) as the result id:\n%s", sql)
	}
	// Searches the joined user name with escaped ILIKE.
	if !strings.Contains(sql, `u.first_name ILIKE $1 ESCAPE '\'`) ||
		!strings.Contains(sql, `u.last_name ILIKE $1 ESCAPE '\'`) {
		t.Errorf("staff arm must ILIKE-ESCAPE the joined user first/last name:\n%s", sql)
	}
	// The joined "user" alias is GLOBAL (no workspace_id) and must be left unbound:
	// its JOIN is keyed only on u.id = s.user_id, with no workspace predicate
	// appended. (A blunt "u.workspace_id" substring check would false-match the
	// legitimate "wu.workspace_id" bind, so assert the exact id-only join line.)
	if !strings.Contains(sql, `JOIN "`+entityid.User+`" u ON u.id = s.user_id`+"\n") {
		t.Errorf("staff arm's global user join must be id-only (no workspace bind):\n%s", sql)
	}
	// No principalscope args — the staff arm binds only $1/$2/$3.
	if len(args) != 3 {
		t.Fatalf("staff arm arg count = %d, want 3 (pattern,limit,ws): %#v", len(args), args)
	}
}

// TestEveryArmUsesEscapedILIKE pins S5: every arm matches with ILIKE ... ESCAPE
// '\' (parameterized $1), so an escaped "%%" pattern matches nothing rather than
// every row.
func TestEveryArmUsesEscapedILIKE(t *testing.T) {
	ctx := staffCtx()
	for key, build := range categoryBuilders {
		sql, _ := build(ctx, "%q%", 5, "ws-1")
		if !strings.Contains(sql, `ILIKE $1 ESCAPE '\'`) {
			t.Errorf("category %q does not use `ILIKE $1 ESCAPE '\\'`:\n%s", key, sql)
		}
	}
}

// TestEscapeLikeNeutralisesWildcards pins the escaper: %, _ and \ are all
// backslash-escaped so a "%%"/"__" query is matched literally under ESCAPE '\'.
func TestEscapeLikeNeutralisesWildcards(t *testing.T) {
	cases := map[string]string{
		"%%":    `\%\%`,
		"__":    `\_\_`,
		`a\b`:   `a\\b`,
		"nick":  "nick",
		`50%_x`: `50\%\_x`,
	}
	for in, want := range cases {
		if got := escapeLike(in); got != want {
			t.Errorf("escapeLike(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestSubscriptionArmCarriesClientReachability pins S6 / codex blocker #1: the
// subscription arm applies principalscope client-reachability to its JOINED
// client alias c (its labels are person data). With a STAFF identity in ctx the
// clause materialises as an "AND c.id IN (...)" reachability subquery and appends
// the staff + workspace args.
func TestSubscriptionArmCarriesClientReachability(t *testing.T) {
	ctx := staffCtx()
	sql, args := buildSubscription(ctx, "%q%", 5, "ws-1")

	if !strings.Contains(sql, "c.id IN (") {
		t.Errorf("subscription arm missing joined-client reachability subquery:\n%s", sql)
	}
	// The reachability clause binds the session staff.id + workspace.id at $4/$5.
	if !strings.Contains(sql, "$4") || !strings.Contains(sql, "$5") {
		t.Errorf("subscription reachability clause did not bind staff/workspace at $4/$5:\n%s", sql)
	}
	// args: $1 pattern, $2 limit, $3 workspace, $4 staffID, $5 workspaceID.
	if len(args) != 5 {
		t.Fatalf("subscription arm arg count = %d, want 5 (pattern,limit,ws,staff,ws): %#v", len(args), args)
	}
	if args[3] != "staff-1" {
		t.Errorf("reachability arg $4 = %v, want staff-1", args[3])
	}
	if args[4] != "ws-1" {
		t.Errorf("reachability arg $5 = %v, want ws-1", args[4])
	}
}

// TestClientArmCarriesClientReachability pins S6 for the base client category.
func TestClientArmCarriesClientReachability(t *testing.T) {
	ctx := staffCtx()
	sql, args := buildClient(ctx, "%q%", 5, "ws-1")
	if !strings.Contains(sql, "c.id IN (") {
		t.Errorf("client arm missing reachability subquery:\n%s", sql)
	}
	if len(args) != 5 {
		t.Fatalf("client arm arg count = %d, want 5: %#v", len(args), args)
	}
}

// TestNonStaffContextOmitsReachability confirms the fail-open-for-workspace,
// fail-closed-for-scope contract: a non-staff (or no-identity) ctx leaves the
// reachability clause OFF (principalscope returns ""), so the arm still carries
// its workspace predicate but no staff subquery, and appends no extra args.
func TestNonStaffContextOmitsReachability(t *testing.T) {
	ctx := context.Background() // no identity → non-staff
	sql, args := buildClient(ctx, "%q%", 5, "ws-1")
	if strings.Contains(sql, "c.id IN (") {
		t.Errorf("non-staff client arm should not emit a reachability subquery:\n%s", sql)
	}
	if len(args) != 3 {
		t.Fatalf("non-staff client arm arg count = %d, want 3 (pattern,limit,ws): %#v", len(args), args)
	}
	if !strings.Contains(sql, "c.workspace_id = $3") {
		t.Errorf("non-staff client arm dropped its workspace predicate:\n%s", sql)
	}
}

// TestTableIdentifiersComeFromEntityid confirms the arms name tables via the
// entityid constants (no bare literals) — a light guard that the constants are
// actually the source (the value equals the table name, so this also documents
// the expected physical names).
func TestTableIdentifiersComeFromEntityid(t *testing.T) {
	ctx := staffCtx()
	subSQL, _ := buildSubscription(ctx, "%q%", 5, "ws-1")
	for _, tbl := range []string{entityid.Subscription, entityid.Client, entityid.User} {
		if !strings.Contains(subSQL, tbl) {
			t.Errorf("subscription arm does not reference entityid table %q:\n%s", tbl, subSQL)
		}
	}
}
