//go:build !mock_auth

// Regression assertion for W0 — the Layer-4 Authorizer boot-fail guard.
//
// Two complementary assertions, both modeled on the existing AST coverage test
// internal/application/shared/authcheck/authcheck_coverage_test.go (which walks
// the AST with go/parser to enforce a security invariant — the template cloned
// here):
//
//  1. AST guard: parse usecases.go, find getServices, and assert that the
//     branch assigning mockAuth.NewAllowAllAuth() to authSvc is GUARDED by
//     allowAllFallbackPermitted() — i.e. there is no UNCONDITIONAL assignment
//     of AllowAll to authSvc. This is the structural guarantee that a
//     password / non-dev build can never silently select allow-all.
//
//  2. Behavioral: with a stub providerManager whose GetAuthProvider() returns
//     nil + a registered PermissionQuery factory, getServices yields a
//     *rbacauth.PermissionAuthorizer (NOT *AllowAllAuthService). Then with
//     CONFIG_AUTH_PROVIDER unset and NO registered PermissionQuery (no *sql.DB),
//     getServices returns a non-nil boot-fail error.
package core

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sync"
	"testing"

	securityports "github.com/erniealice/espyna-golang/internal/application/ports/security"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/composition/providers"
	mockAuth "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/auth/mock"
	rbacauth "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/auth/rbac"
	internalregistry "github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// -----------------------------------------------------------------------------
// Assertion 1 — AST guard
// -----------------------------------------------------------------------------

// TestAllowAllBranchGuardedByFallbackPermitted parses usecases.go, locates the
// getServices method, and asserts that every assignment of
// mockAuth.NewAllowAllAuth() to authSvc is lexically nested inside an `if`
// whose condition references allowAllFallbackPermitted(). An unconditional
// assignment (the pre-W0 leak) would fail this test.
func TestAllowAllBranchGuardedByFallbackPermitted(t *testing.T) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "usecases.go", nil, 0)
	if err != nil {
		t.Fatalf("failed to parse usecases.go: %v", err)
	}

	var getServices *ast.FuncDecl
	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == "getServices" && fn.Recv != nil {
			getServices = fn
			break
		}
	}
	if getServices == nil {
		t.Fatal("could not find getServices method in usecases.go")
	}

	// Collect every call to NewAllowAllAuth and the every if-statement that
	// references allowAllFallbackPermitted, recording source positions so we can
	// test containment.
	type span struct{ start, end token.Pos }
	var allowAllCalls []token.Pos
	var guardedSpans []span

	ast.Inspect(getServices.Body, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.CallExpr:
			if sel, ok := v.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "NewAllowAllAuth" {
				allowAllCalls = append(allowAllCalls, v.Pos())
			}
		case *ast.IfStmt:
			// Does this if-condition reference allowAllFallbackPermitted()?
			refsGuard := false
			ast.Inspect(v.Cond, func(c ast.Node) bool {
				if id, ok := c.(*ast.Ident); ok && id.Name == "allowAllFallbackPermitted" {
					refsGuard = true
					return false
				}
				return true
			})
			if refsGuard {
				guardedSpans = append(guardedSpans, span{v.Pos(), v.End()})
			}
		}
		return true
	})

	if len(allowAllCalls) == 0 {
		t.Fatal("expected at least one NewAllowAllAuth() call in getServices (the guarded dev/mock fallback)")
	}

	for _, callPos := range allowAllCalls {
		guarded := false
		for _, s := range guardedSpans {
			if callPos >= s.start && callPos <= s.end {
				guarded = true
				break
			}
		}
		if !guarded {
			t.Errorf("UNGUARDED NewAllowAllAuth() call at %s — every AllowAll selection MUST be nested inside an `if allowAllFallbackPermitted()` block (Q-AWS2=C boot-fail guard)",
				fset.Position(callPos))
		}
	}
}

// TestGetServicesHasBootFailError asserts that getServices contains a return
// statement that produces a non-nil error on the forbidden fallback path — the
// structural counterpart to the behavioral boot-fail test below.
func TestGetServicesHasBootFailError(t *testing.T) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "usecases.go", nil, 0)
	if err != nil {
		t.Fatalf("failed to parse usecases.go: %v", err)
	}

	var getServices *ast.FuncDecl
	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == "getServices" && fn.Recv != nil {
			getServices = fn
			break
		}
	}
	if getServices == nil {
		t.Fatal("could not find getServices method in usecases.go")
	}

	foundErrorReturn := false
	ast.Inspect(getServices.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		// fmt.Errorf(...) in getServices is the boot-fail error.
		if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "fmt" && sel.Sel.Name == "Errorf" {
			foundErrorReturn = true
			return false
		}
		return true
	})

	if !foundErrorReturn {
		t.Error("getServices must construct a non-nil error (fmt.Errorf) for the forbidden AllowAll-fallback path — the Q-AWS2=C boot-fail guarantee")
	}
}

// -----------------------------------------------------------------------------
// Assertion 2 — behavioral
// -----------------------------------------------------------------------------

// stubPermissionQuery is a registered-factory-shaped PermissionQuery used to
// prove getServices builds a *PermissionAuthorizer rather than AllowAll.
type stubPermissionQuery struct{}

func (stubPermissionQuery) GetUserPermissionCodes(
	ctx context.Context,
	userID, workspaceID string,
	bindingKind int32,
	bindingID string,
	actingAsClientID, actingAsSupplierID string,
) ([]string, error) {
	return []string{}, nil
}

var _ securityports.PermissionQuery = stubPermissionQuery{}

// stubSQLDBProvider implements contracts.Provider + GetConnection() any so that
// resolvePermissionQuery's GetConnection→*sql.DB extraction succeeds. The
// *sql.DB is opened lazily (no real connection) — the postgres/stub factory
// only wraps it, never dials.
type stubSQLDBProvider struct {
	db *sql.DB
}

func (p *stubSQLDBProvider) Type() contracts.ProviderType   { return "" }
func (p *stubSQLDBProvider) Name() string                   { return "stub-sql" }
func (p *stubSQLDBProvider) Initialize(_ interface{}) error { return nil }
func (p *stubSQLDBProvider) Health(_ context.Context) error { return nil }
func (p *stubSQLDBProvider) Close() error                   { return nil }
func (p *stubSQLDBProvider) GetConnection() any             { return p.db }

// nilDriver is a no-op database/sql driver registered once so that
// sql.Open(...) returns a usable *sql.DB handle WITHOUT dialing — the stub
// PermissionQuery factory only wraps the handle, it never queries. This keeps
// the behavioral test self-contained and free of any real DB dependency.
type nilDriver struct{}

func (nilDriver) Open(string) (driver.Conn, error) { return nil, driver.ErrBadConn }

var registerNilDriverOnce sync.Once

func openLazySQLDB() (*sql.DB, error) {
	registerNilDriverOnce.Do(func() {
		sql.Register("rbac-test-nildriver", nilDriver{})
	})
	return sql.Open("rbac-test-nildriver", "")
}

// TestGetServicesWiresRealAuthorizer asserts that, with a registered
// PermissionQuery factory and a SQL-backed database provider but NO auth
// provider, getServices wires *rbacauth.PermissionAuthorizer — NOT
// *mockAuth.AllowAllAuthService.
func TestGetServicesWiresRealAuthorizer(t *testing.T) {
	// Register a stub PermissionQuery factory (postgres-shaped: takes *sql.DB).
	internalregistry.RegisterPermissionQueryFactory(func(db any) any {
		if _, ok := db.(*sql.DB); !ok {
			return nil
		}
		return stubPermissionQuery{}
	})

	db, err := openLazySQLDB()
	if err != nil {
		t.Fatalf("could not open a lazy *sql.DB handle: %v", err)
	}
	defer db.Close()

	mgr := &providers.Manager{}
	mgr.SetDatabaseProvider(&stubSQLDBProvider{db: db})
	// Deliberately do NOT set an auth provider — forces resolution to step 2.

	uci := NewUseCaseInitializer(mgr)
	container := &Container{}

	authSvc, _, _, _, gerr := uci.getServices(container)
	if gerr != nil {
		t.Fatalf("getServices returned unexpected error: %v", gerr)
	}
	if _, ok := authSvc.(*rbacauth.PermissionAuthorizer); !ok {
		t.Fatalf("expected *rbacauth.PermissionAuthorizer, got %T", authSvc)
	}
	if _, isAllowAll := authSvc.(*mockAuth.AllowAllAuthService); isAllowAll {
		t.Fatal("getServices wired *AllowAllAuthService on a postgres-shaped build — the W0 leak must be closed")
	}
}

// TestGetServicesBootFailsWithoutPermissionQuery asserts that with NO auth
// provider, NO *sql.DB (so no PermissionQuery resolvable), and
// CONFIG_AUTH_PROVIDER unset (allowAllFallbackPermitted()==false), getServices
// returns a non-nil boot-fail error rather than silently selecting AllowAll.
func TestGetServicesBootFailsWithoutPermissionQuery(t *testing.T) {
	// Ensure the dev/mock escape hatch is closed for this test.
	prev, had := os.LookupEnv("CONFIG_AUTH_PROVIDER")
	os.Unsetenv("CONFIG_AUTH_PROVIDER")
	t.Cleanup(func() {
		if had {
			os.Setenv("CONFIG_AUTH_PROVIDER", prev)
		}
	})

	// No database provider at all → resolvePermissionQuery returns nil.
	mgr := &providers.Manager{}
	uci := NewUseCaseInitializer(mgr)
	container := &Container{}

	_, _, _, _, gerr := uci.getServices(container)
	if gerr == nil {
		t.Fatal("expected a non-nil boot-fail error when no RBAC Authorizer is available and AllowAll is forbidden (CONFIG_AUTH_PROVIDER unset), got nil")
	}
}

// TestGetServicesAllowsMockWhenPermitted asserts the dev/mock escape hatch:
// with CONFIG_AUTH_PROVIDER=mock_auth and no PermissionQuery, getServices
// selects AllowAll without error (the guarded fallback).
func TestGetServicesAllowsMockWhenPermitted(t *testing.T) {
	prev, had := os.LookupEnv("CONFIG_AUTH_PROVIDER")
	os.Setenv("CONFIG_AUTH_PROVIDER", "mock_auth")
	t.Cleanup(func() {
		if had {
			os.Setenv("CONFIG_AUTH_PROVIDER", prev)
		} else {
			os.Unsetenv("CONFIG_AUTH_PROVIDER")
		}
	})

	mgr := &providers.Manager{}
	uci := NewUseCaseInitializer(mgr)
	container := &Container{}

	authSvc, _, _, _, gerr := uci.getServices(container)
	if gerr != nil {
		t.Fatalf("getServices returned unexpected error with mock_auth permitted: %v", gerr)
	}
	if authSvc == nil {
		t.Fatal("expected a non-nil AllowAll authSvc when CONFIG_AUTH_PROVIDER=mock_auth")
	}
}
