package authcheck_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// skipDirs lists directories to skip during the authcheck coverage scan.
// Each entry should be the directory name (not a full path).
//
// Reasons:
//   - authcheck:   the authcheck package itself (not a use case)
//   - contextutil: utility package, not a use case
//   - integration: external service adapters (email, payment, scheduler, tabular)
//                  that don't need permission checks
//   - common:      cross-domain helpers (attribute, category) that are called by
//                  other use cases and don't carry AuthorizationService
var skipDirs = map[string]bool{
	"authcheck":   true,
	"contextutil": true,
	"integration": true,
	"common":      true,
}

// skipFilePatterns lists filename substrings for files that legitimately
// don't need direct authcheck.Check calls because they delegate to inner
// use cases that already enforce permissions.
var skipFilePatterns = []string{
	"_by_code",  // ByCode composition use cases delegate to inner create use cases
	"helpers",   // Helper/utility files without Execute methods
	"calculate", // Pure calculation helpers
}

func TestAllUseCasesHaveAuthCheck(t *testing.T) {
	// Walk from the usecases directory (parent of authcheck)
	usecasesDir := filepath.Join("..")

	var missing []string

	err := filepath.Walk(usecasesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories in the skip list
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Only check .go files, skip test files
		if !strings.HasSuffix(info.Name(), ".go") ||
			strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}

		// Skip files named "usecases.go" (aggregate structs, not use cases)
		if info.Name() == "usecases.go" {
			return nil
		}

		// Skip files matching known delegation/helper patterns
		for _, pattern := range skipFilePatterns {
			if strings.Contains(info.Name(), pattern) {
				return nil
			}
		}

		// Parse the file
		fset := token.NewFileSet()
		node, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil // skip unparseable files
		}

		// Find Execute methods
		for _, decl := range node.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "Execute" || fn.Recv == nil {
				continue
			}

			// Check if function body contains "authcheck" reference
			hasAuthCheck := false
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if sel, ok := n.(*ast.SelectorExpr); ok {
					if ident, ok := sel.X.(*ast.Ident); ok {
						if ident.Name == "authcheck" {
							hasAuthCheck = true
							return false
						}
					}
				}
				return true
			})

			if !hasAuthCheck {
				// Get receiver type name for the error message
				recvType := "unknown"
				if len(fn.Recv.List) > 0 {
					switch t := fn.Recv.List[0].Type.(type) {
					case *ast.StarExpr:
						if ident, ok := t.X.(*ast.Ident); ok {
							recvType = ident.Name
						}
					case *ast.Ident:
						recvType = t.Name
					}
				}

				// Use forward slashes for consistent output
				relPath := strings.ReplaceAll(path, "\\", "/")
				missing = append(missing, relPath+": "+recvType+".Execute()")
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to walk usecases directory: %v", err)
	}

	if len(missing) > 0 {
		t.Errorf("Found %d Execute() methods missing authcheck.Check():\n", len(missing))
		for _, m := range missing {
			t.Errorf("  MISSING: %s", m)
		}
	}
}
