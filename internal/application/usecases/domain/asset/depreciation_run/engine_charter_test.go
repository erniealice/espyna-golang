package depreciation_run

import (
	"go/parser"
	"go/token"
	"testing"
)

// TestEngineCharter_StdlibOnly enforces the engine.go file-level CHARTER: the
// pure-compute depreciation engine must import the Go standard library ONLY.
// It must NOT acquire proto entity types (esqyma/...), DB drivers/adapters, or
// any internal/application/usecases/... import — those belong in the surrounding
// depreciation_run use cases, never in the engine.
//
// Rationale: engine.go was relocated into package depreciation_run (which itself
// imports proto + ports), so the purity boundary is no longer enforced by a
// package wall (Q1-grain trade-off, docs/plan/20260615-espyna-stray-layer-relocation/decisions.html).
// This test restores that guarantee at the file level.
func TestEngineCharter_StdlibOnly(t *testing.T) {
	allowed := map[string]bool{
		`"errors"`: true,
		`"math"`:   true,
		`"time"`:   true,
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "engine.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse engine.go: %v", err)
	}

	for _, imp := range f.Imports {
		if !allowed[imp.Path.Value] {
			t.Errorf("engine.go CHARTER violation: forbidden import %s — the depreciation engine is a pure-compute leaf and must import stdlib only (errors/math/time); move I/O, proto, or repository access into the depreciation_run use cases", imp.Path.Value)
		}
	}
}
