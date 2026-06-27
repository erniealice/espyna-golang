//go:build postgresql

package entity

import (
	"os"
	"strings"
	"testing"
)

// TestDelegatePageDataWorkspaceScoped is a source-scanning regression guard for the
// cross-tenant IDOR fix (2026-06-27). The four delegate / delegate_client page-data
// read methods issue hand-written SQL with no ORM tenancy, so each MUST filter by the
// session workspace. This test reads the adapter source and asserts every method body
// (a) reads the workspace from the session identity — never the request — and (b)
// applies a workspace_id predicate. It fails loudly if a future edit drops the scoping.
//
// Why source-scanning and not a live query: the entity-package adapters have no PG
// fixture harness (see client_pagedata_test.go, skipped for the same reason). Empirical
// proof of the isolation is done out-of-band via a transactional two-workspace fixture
// on a live DB; this guard prevents silent regressions in CI without a database.
func TestDelegatePageDataWorkspaceScoped(t *testing.T) {
	cases := []struct {
		file   string
		method string
	}{
		{"delegate.go", "GetDelegateListPageData"},
		{"delegate.go", "GetDelegateItemPageData"},
		{"delegate_client.go", "GetDelegateClientListPageData"},
		{"delegate_client.go", "GetDelegateClientItemPageData"},
	}
	srcCache := map[string]string{}
	for _, c := range cases {
		src, ok := srcCache[c.file]
		if !ok {
			b, err := os.ReadFile(c.file)
			if err != nil {
				t.Fatalf("read %s: %v", c.file, err)
			}
			src = string(b)
			srcCache[c.file] = src
		}
		body := methodBody(t, src, c.method)
		if !strings.Contains(body, "identity.Must(ctx).WorkspaceID") {
			t.Errorf("%s.%s: missing session-workspace read (identity.Must(ctx).WorkspaceID) — cross-tenant IDOR scoping dropped", c.file, c.method)
		}
		if !strings.Contains(body, "workspace_id") {
			t.Errorf("%s.%s: missing workspace_id predicate — cross-tenant IDOR scoping dropped", c.file, c.method)
		}
	}
}

// methodBody returns the source text of the named method, from the line declaring it to
// just before the next top-level func (or EOF).
func methodBody(t *testing.T, src, method string) string {
	t.Helper()
	marker := ") " + method + "("
	i := strings.Index(src, marker)
	if i < 0 {
		t.Fatalf("method %s not found in source", method)
		return ""
	}
	rest := src[i:]
	if j := strings.Index(rest[1:], "\nfunc "); j >= 0 {
		return rest[:j+1]
	}
	return rest
}
