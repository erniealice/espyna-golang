//go:build postgresql

package rbac

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// Login-scoped roles (2026-11-05): role.login_principal_types narrows WHICH
// ROLES load on a login. These tests pin the SQL shape for every kind and,
// against a real Postgres (TEST_DATABASE_URL), the behaviour for NULL / {} /
// {7} / {2} on the Staff login (2) and the Teacher login (7), plus DENY-wins.

// TestBuildPermissionQuerySQL_RoleLoginKindPredicate (AC-1, AC-3): every bound
// kind carries the role-login exclusion with its own kind literal, on the role
// column of its own grant chain, inside user_roles.
func TestBuildPermissionQuerySQL_RoleLoginKindPredicate(t *testing.T) {
	cases := []struct {
		kind    int32
		roleCol string
		acting  [2]string
	}{
		{principalTypeOperatorOwner, "wur.role_id", [2]string{}},
		{principalTypeOperatorStaff, "wur.role_id", [2]string{}},
		{principalTypeClient, "cpg.role_id", [2]string{}},
		{principalTypeClientDelegate, "dc.role_id", [2]string{"client-1", ""}},
		{principalTypeSupplier, "spg.role_id", [2]string{}},
		{principalTypeSupplierDelegate, "ds.role_id", [2]string{"", "supplier-1"}},
		{principalTypeStaff, "wur.role_id", [2]string{}},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("kind_%d", tc.kind), func(t *testing.T) {
			stmt, _, ok := buildPermissionQuerySQL("u", "w", tc.kind, "b", tc.acting[0], tc.acting[1])
			if !ok {
				t.Fatalf("kind %d: want ok", tc.kind)
			}
			cteEnd := strings.Index(stmt, "SELECT DISTINCT p.permission_code")
			if cteEnd < 0 {
				t.Fatalf("no permission select in:\n%s", stmt)
			}
			cte := stmt[:cteEnd]
			for _, want := range []string{
				"WITH user_roles_held AS (",
				"SELECT " + tc.roleCol,
				"user_roles AS (",
				"FROM user_roles_held h",
				"WHERE NOT EXISTS (",
				"FROM role r_login",
				"WHERE r_login.id = h.role_id",
				"AND cardinality(r_login.login_principal_types) > 0",
				fmt.Sprintf("AND NOT COALESCE(%d = ANY(r_login.login_principal_types), false)", tc.kind),
			} {
				if !strings.Contains(cte, want) {
					t.Errorf("user_roles CTE missing %q\n%s", want, cte)
				}
			}
			for other := int32(1); other <= 7; other++ {
				if other != tc.kind && strings.Contains(cte, fmt.Sprintf("COALESCE(%d = ANY(r_login", other)) {
					t.Errorf("kind %d CTE carries kind %d literal", tc.kind, other)
				}
			}
			// ALLOW reads the login-scoped user_roles; DENY reads every held
			// role (user_roles_held), so a scoped role's DENY still applies.
			sel := stmt[cteEnd:]
			if strings.Contains(sel, "login_principal_types") || strings.Contains(sel, "workspace_user_role") {
				t.Errorf("permission select must read roles only via the CTEs:\n%s", sel)
			}
			if strings.Count(sel, "JOIN user_roles ur ON") != 1 || strings.Count(sel, "JOIN user_roles_held ur2 ON") != 1 {
				t.Errorf("want ALLOW on user_roles and DENY on user_roles_held:\n%s", sel)
			}
			// The role column we read must not be the deprecated one.
			if strings.Contains(cte, "r_login.applicable_principal_types") {
				t.Errorf("deprecated role.applicable_principal_types must not be read")
			}
			if cte2, ok := bindingCTEForKind(tc.kind); !ok || !strings.HasPrefix(stmt, cte2) {
				t.Errorf("bindingCTEForKind(%d) differs from the assembled CTE", tc.kind)
			}
		})
	}
}

// TestBuildPermissionQuerySQL_UnionPathDropsLoginScopedRoles (AC-2): the
// legacy exact-zero path has no binding kind, so it keeps only roles whose
// login set is empty (fails closed); DENY still reads every held role.
func TestBuildPermissionQuerySQL_UnionPathDropsLoginScopedRoles(t *testing.T) {
	stmt, args, ok := buildPermissionQuerySQL("u", "w", 0, "", "", "")
	if !ok || len(args) != 2 {
		t.Fatalf("legacy union: ok=%v args=%d", ok, len(args))
	}
	for _, want := range []string{
		"WITH user_roles_held AS (",
		"FROM user_roles_held h",
		"AND COALESCE(cardinality(r_login.login_principal_types), 0) > 0",
		"JOIN user_roles ur ON",
		"JOIN user_roles_held ur2 ON",
	} {
		if !strings.Contains(stmt, want) {
			t.Errorf("legacy union missing %q\n%s", want, stmt)
		}
	}
	if strings.Contains(stmt, "= ANY(r_login.login_principal_types)") {
		t.Errorf("legacy union must not test a kind (it has none)")
	}
}

// TestPermissionQuery_RoleLoginBehaviour runs the assembled SQL against real
// Postgres. All fixtures are TEMP tables (they shadow the real tables inside
// one transaction that is rolled back), so any database works and nothing is
// written. Skips without TEST_DATABASE_URL.
func TestPermissionQuery_RoleLoginBehaviour(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("TEST_DATABASE_URL unusable: %v", err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("TEST_DATABASE_URL unreachable: %v", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback() //nolint:errcheck

	ddl := `
CREATE TEMP TABLE workspace_user (id text, user_id text, workspace_id text, active boolean) ON COMMIT DROP;
CREATE TEMP TABLE workspace_user_role (workspace_user_id text, role_id text, active boolean) ON COMMIT DROP;
CREATE TEMP TABLE staff (id text, user_id text, workspace_id text, active boolean) ON COMMIT DROP;
CREATE TEMP TABLE role (id text, active boolean, applicable_principal_types integer[], login_principal_types integer[]) ON COMMIT DROP;
CREATE TEMP TABLE permission (id text, permission_code text, active boolean, applicable_principal_types integer[]) ON COMMIT DROP;
CREATE TEMP TABLE role_permission (role_id text, permission_id text, permission_type text, active boolean) ON COMMIT DROP;
-- Empty grant chains for the legacy union path.
CREATE TEMP TABLE client_portal_grant (id text, user_id text, workspace_id text, role_id text, active boolean) ON COMMIT DROP;
CREATE TEMP TABLE supplier_portal_grant (id text, user_id text, workspace_id text, role_id text, active boolean) ON COMMIT DROP;
CREATE TEMP TABLE delegate (id text, user_id text, active boolean) ON COMMIT DROP;
CREATE TEMP TABLE delegate_client (delegate_id text, client_id text, workspace_id text, role_id text, active boolean) ON COMMIT DROP;
CREATE TEMP TABLE delegate_supplier (delegate_id text, supplier_id text, workspace_id text, role_id text, active boolean) ON COMMIT DROP;
CREATE TEMP TABLE client (id text, workspace_id text, active boolean) ON COMMIT DROP;
CREATE TEMP TABLE supplier (id text, workspace_id text, active boolean) ON COMMIT DROP;
INSERT INTO workspace_user VALUES ('wu1','u1','w1',true);
INSERT INTO staff VALUES ('s1','u1','w1',true);
-- applicable_principal_types {1,2} on every role, as in prod: must be ignored.
-- The temp table has no CHECK, so it can hold the NULL-element sets the
-- real column forbids: they must fail closed (load nowhere).
INSERT INTO role VALUES
 ('r_null',  true, '{1,2}', NULL),
 ('r_empty', true, '{1,2}', '{}'),
 ('r_k7',    true, '{1,2}', '{7}'),
 ('r_k2',    true, '{1,2}', '{2}'),
 ('r_k7null', true, '{1,2}', '{7,NULL}'),
 ('r_nullel', true, '{1,2}', '{NULL}'),
 ('r_deny_null', true, '{1,2}', NULL),
 ('r_deny_k2',   true, '{1,2}', '{2}');
INSERT INTO workspace_user_role SELECT 'wu1', id, true FROM role;
INSERT INTO permission VALUES
 ('p_null','code.null',true,'{1,2,7}'), ('p_empty','code.empty',true,'{1,2,7}'),
 ('p_k7','code.k7',true,'{1,2,7}'),     ('p_k2','code.k2',true,'{1,2,7}'),
 ('p_shared','code.shared',true,'{1,2,7}'), ('p_x','code.x',true,'{1,2,7}'),
 ('p_staffonly','code.staffonly',true,'{1,2}'),
 ('p_k7null','code.k7null',true,'{1,2,7}'), ('p_nullel','code.nullel',true,'{1,2,7}');
INSERT INTO role_permission VALUES
 ('r_null','p_null','PERMISSION_TYPE_ALLOW',true),
 ('r_empty','p_empty','PERMISSION_TYPE_ALLOW',true),
 ('r_k7','p_k7','PERMISSION_TYPE_ALLOW',true),
 ('r_k2','p_k2','PERMISSION_TYPE_ALLOW',true),
 ('r_null','p_staffonly','PERMISSION_TYPE_ALLOW',true),
 -- DENY-wins from an every-login role: code.shared gone on both logins.
 ('r_k7','p_shared','PERMISSION_TYPE_ALLOW',true),
 ('r_deny_null','p_shared','PERMISSION_TYPE_DENY',true),
 -- DENY from a held role applies on every login, even where the role's
 -- ALLOWs are scoped away: r_deny_k2 ({2}) denies code.x on all logins.
 ('r_null','p_x','PERMISSION_TYPE_ALLOW',true),
 ('r_deny_k2','p_x','PERMISSION_TYPE_DENY',true),
 -- {7,NULL}: loads on the Teacher login only; {NULL}: loads nowhere.
 ('r_k7null','p_k7null','PERMISSION_TYPE_ALLOW',true),
 ('r_nullel','p_nullel','PERMISSION_TYPE_ALLOW',true);`
	if _, err := tx.ExecContext(ctx, ddl); err != nil {
		t.Fatalf("fixtures: %v", err)
	}

	run := func(kind int32, binding string) []string {
		t.Helper()
		stmt, args, ok := buildPermissionQuerySQL("u1", "w1", kind, binding, "", "")
		if !ok {
			t.Fatalf("kind %d not ok", kind)
		}
		rows, err := tx.QueryContext(ctx, stmt, args...)
		if err != nil {
			t.Fatalf("kind %d query: %v", kind, err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var c string
			if err := rows.Scan(&c); err != nil {
				t.Fatal(err)
			}
			out = append(out, c)
		}
		sort.Strings(out)
		return out
	}

	cases := []struct {
		name    string
		kind    int32
		binding string
		want    []string
	}{
		// Staff login (2): r_k7, r_k7null and r_nullel drop; code.shared is
		// denied by an every-login role; code.x is denied by r_deny_k2.
		{"staff_login_kind2", principalTypeOperatorStaff, "wu1",
			[]string{"code.empty", "code.k2", "code.null", "code.staffonly"}},
		// Owner login (1): r_k2, r_k7, r_k7null, r_nullel drop; r_deny_k2's DENY
		// still applies (DENY reads every held role), so code.x stays denied.
		{"owner_login_kind1", principalTypeOperatorOwner, "wu1",
			[]string{"code.empty", "code.null", "code.staffonly"}},
		// Teacher login (7): r_k2 drops; r_k7null loads ({7,NULL} names 7);
		// r_nullel drops; code.x stays denied; code.staffonly is cut by the
		// permission tag.
		{"teacher_login_kind7", principalTypeStaff, "s1",
			[]string{"code.empty", "code.k7", "code.k7null", "code.null"}},
		// Legacy union (no binding): every scoped role drops (r_k2, r_k7,
		// r_k7null, r_nullel); DENYs from all held roles apply.
		{"legacy_union_kind0", principalTypeUnspecified, "",
			[]string{"code.empty", "code.null", "code.staffonly"}},
	}
	for _, tc := range cases {
		got := run(tc.kind, tc.binding)
		if strings.Join(got, ",") != strings.Join(tc.want, ",") {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}
