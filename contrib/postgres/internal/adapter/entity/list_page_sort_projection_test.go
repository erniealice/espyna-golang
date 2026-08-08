//go:build postgresql

package entity

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestListPageSortFieldsAreProjected keeps ORDER BY valid for the outer
// "FROM enriched e" query. BuildOrderBy intentionally emits bare allowlisted
// names, so each one must be selected by the enriched CTE.
func TestListPageSortFieldsAreProjected(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		alias   string
		allowed []string
	}{
		{"location", "location.go", "l", locationSortableSQLCols},
		{"workspace", "workspace.go", "w", workspaceSortableSQLCols},
		{"supplier", "supplier.go", "s", supplierSortableSQLCols},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := listPageAdapterSource(t, tt.file)
			projection := enrichedProjection(t, source)
			for _, field := range tt.allowed {
				if !strings.Contains(projection, tt.alias+"."+field) {
					t.Errorf("allowlisted sort field %q is missing from enriched projection", field)
				}
			}
		})
	}

	for _, field := range workspaceSortableSQLCols {
		if field == "status" {
			t.Fatal("workspace.status is not a schema field and must not be allowlisted for sorting")
		}
	}
}

func listPageAdapterSource(t *testing.T, file string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	source, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), file))
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	return string(source)
}

func enrichedProjection(t *testing.T, source string) string {
	t.Helper()
	start := strings.Index(source, "enriched AS (")
	if start < 0 {
		t.Fatal("enriched CTE not found")
	}
	projectionStart := strings.Index(source[start:], "SELECT")
	from := strings.Index(source[start+projectionStart:], "FROM `+entityid.")
	if projectionStart < 0 || from < 0 {
		t.Fatal("enriched projection boundaries not found")
	}
	return source[start+projectionStart : start+projectionStart+from]
}
