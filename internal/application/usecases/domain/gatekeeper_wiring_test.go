package domain

// gatekeeper_wiring_test.go is a SOURCE-SCANNING guard against the
// "grouped NewUseCases drops the ActionGatekeeper" regression class.
//
// Background: many domain packages declare a top-level *Services struct that
// carries `ActionGatekeeper *actiongate.ActionGatekeeper`, then fan it out into
// per-operation sub-services structs (CreateXServices, ReadXServices, ...). Each
// such sub-services struct ALSO declares ActionGatekeeper, and each use case's
// Execute calls `uc.services.ActionGatekeeper.Check(...)`. If the grouped
// constructor builds the sub-services struct as a KEYED composite literal but
// omits `ActionGatekeeper:`, the field is left nil. actiongate.ActionGatekeeper
// .Check fail-closes on a nil receiver ("authorization denied: action gatekeeper
// not configured") BEFORE the IsEnabled shadow short-circuit, so every operation
// in that package is silently denied at runtime even though the binary compiles.
//
// This test needs no DI wiring and imports nothing from the domain packages: it
// walks the .go sources under this directory with go/parser and FAILS when a
// composite literal of an ActionGatekeeper-declaring *Services type is written
// as a keyed literal that does not assign the ActionGatekeeper field
// (i.e. "declares > assigns").
//
// Scope notes (intentional, to avoid false positives):
//   - Type CONVERSIONS (e.g. `CreateXServices(s)`) are not composite literals and
//     are not flagged; Go copies every field, including ActionGatekeeper, as long
//     as the layouts are identical (which the compiler enforces).
//   - Whole-struct pass-through (`services: services`) is not a *Services literal
//     and is not flagged; the field rides along in the copied value.
//   - POSITIONAL literals are not flagged: Go requires every field to be present,
//     so the gatekeeper cannot be silently dropped.
//   - EMPTY literals (`T{}`) are not flagged: they are a distinct shape used as
//     typed zero-value placeholders and are not the observed regression.
//
// Run in isolation (no build tags, no sibling packages needed):
//   go test ./internal/application/usecases/domain/ -run TestActionGatekeeperWiring

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestActionGatekeeperWiring(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed: cannot locate test source to derive scan root")
	}
	root := filepath.Dir(thisFile)

	fset := token.NewFileSet()
	filesByDir := map[string][]*ast.File{}

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		// parser.ParseFile parses syntax regardless of //go:build constraints,
		// so tag-gated files (e.g. postgresql) are still scanned.
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return perr
		}
		dir := filepath.Dir(path)
		filesByDir[dir] = append(filesByDir[dir], f)
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walking domain sources: %v", walkErr)
	}

	// Pass 1: per-directory set of *Services type names that declare an
	// ActionGatekeeper field. Grouped by directory because a sub-services type is
	// usually declared in a sibling file of the same package as its literal.
	declByDir := make(map[string]map[string]bool, len(filesByDir))
	for dir, files := range filesByDir {
		declares := map[string]bool{}
		for _, f := range files {
			for _, decl := range f.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.TYPE {
					continue
				}
				for _, spec := range gd.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || !strings.HasSuffix(ts.Name.Name, "Services") {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						continue
					}
					for _, field := range st.Fields.List {
						for _, name := range field.Names {
							if name.Name == "ActionGatekeeper" {
								declares[ts.Name.Name] = true
							}
						}
					}
				}
			}
		}
		declByDir[dir] = declares
	}

	// Pass 2: flag keyed composite literals of a declaring type that omit the field.
	type offense struct {
		pos     string
		typName string
	}
	var offenses []offense

	for dir, files := range filesByDir {
		declares := declByDir[dir]
		if len(declares) == 0 {
			continue
		}
		for _, f := range files {
			ast.Inspect(f, func(n ast.Node) bool {
				cl, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}
				// Same-package reference => bare identifier (e.g. CreateJobServices{}).
				id, ok := cl.Type.(*ast.Ident)
				if !ok || !declares[id.Name] {
					return true
				}
				if len(cl.Elts) == 0 {
					return true // empty literal: out of scope (see header)
				}
				hasKeyedField := false
				assignsGatekeeper := false
				for _, elt := range cl.Elts {
					kv, ok := elt.(*ast.KeyValueExpr)
					if !ok {
						continue // positional literal element => out of scope
					}
					hasKeyedField = true
					if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "ActionGatekeeper" {
						assignsGatekeeper = true
					}
				}
				if hasKeyedField && !assignsGatekeeper {
					offenses = append(offenses, offense{
						pos:     fset.Position(cl.Pos()).String(),
						typName: id.Name,
					})
				}
				return true
			})
		}
	}

	for _, o := range offenses {
		t.Errorf("%s: %s is built as a keyed literal that omits ActionGatekeeper "+
			"(type declares the field => nil ActionGatekeeper => fail-closed at runtime). "+
			"Add `ActionGatekeeper: <servicesParam>.ActionGatekeeper,` to the literal.",
			o.pos, o.typName)
	}
	if len(offenses) > 0 {
		t.Fatalf("%d sub-services literal(s) declare ActionGatekeeper but do not assign it; "+
			"see the errors above", len(offenses))
	}
}
