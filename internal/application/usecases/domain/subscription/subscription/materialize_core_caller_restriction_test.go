package subscription

// Item #5 (option a) — caller-restriction guard for the ungated materialization
// cores.
//
// The split introduced two UNGATED internal cores:
//   - MaterializeJobsForSubscriptionUseCase.materializeCore, and
//   - MaterializeInstanceJobsForSubscriptionUseCase.executeInternal.
//
// Each is reachable only behind an authorizing ancestor:
//   - the subscription:update-gated public Execute wrappers, and
//   - the create side effect (CreateSubscriptionUseCase, already authorized by
//     subscription:create) via the MaterializeJobsForSubscriptionInstantiator
//     adapter methods (InstantiateJobsFromPlan / InstantiateJobsFromPlanDetailed).
//
// A charter doc-comment says so, but a comment cannot fail CI. Codex flagged
// (item #5 HIGH) that the adapter type + its methods remain exported, so any new
// in-module caller could invoke the ungated core with an unauthorized context
// and spawn jobs. This AST test is the enforcement: it asserts the ONLY non-test
// callers of each core are the allowlisted ancestor files. A new caller in any
// other file fails here — forcing the "did you mean to bypass the gate?"
// conversation before the bypass can ship.
//
// Test files are intentionally exempt: the unit tests drive the cores directly
// (the charter permits "tests" as callers), and CI test binaries never serve a
// route.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// coreCallerAllowlist maps each ungated core method to the set of non-test files
// permitted to call it. Adding a file here is the deliberate, reviewable signal
// that a new authorized ancestor is being wired.
var coreCallerAllowlist = map[string]map[string]bool{
	"materializeCore": {
		"materialize_jobs_for_subscription.go": true, // public Execute -> core (gated)
		"job_instantiator_adapter.go":          true, // InstantiateJobsFromPlan (create side effect)
		"create_subscription.go":               true, // InstantiateJobsFromPlanDetailed (create side effect)
	},
	"executeInternal": {
		"materialize_instance_jobs_for_subscription.go": true, // public Execute -> core (gated)
	},
}

func TestUngatedMaterializeCore_CallerRestriction(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}

	var violations []string
	sawCall := map[string]bool{} // sanity: each core must be called at least once

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		fset := token.NewFileSet()
		node, perr := parser.ParseFile(fset, name, nil, 0)
		if perr != nil {
			t.Fatalf("parse %s: %v", name, perr)
		}

		ast.Inspect(node, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			allowed, guarded := coreCallerAllowlist[sel.Sel.Name]
			if !guarded {
				return true
			}
			sawCall[sel.Sel.Name] = true
			if !allowed[name] {
				violations = append(violations,
					name+" calls ungated "+sel.Sel.Name+"() — only "+
						joinAllowed(allowed)+" may (item #5 charter). If this is a "+
						"newly-authorized ancestor, add it to coreCallerAllowlist and "+
						"prove the gate; otherwise route through the gated public Execute.")
			}
			return true
		})
	}

	for method := range coreCallerAllowlist {
		if !sawCall[method] {
			t.Errorf("no non-test call to %q found — the method was renamed/removed; "+
				"update coreCallerAllowlist so this guard keeps biting", method)
		}
	}

	for _, v := range violations {
		t.Errorf("BYPASS: %s", v)
	}
}

func joinAllowed(m map[string]bool) string {
	files := make([]string, 0, len(m))
	for f := range m {
		files = append(files, f)
	}
	return strings.Join(files, ", ")
}
