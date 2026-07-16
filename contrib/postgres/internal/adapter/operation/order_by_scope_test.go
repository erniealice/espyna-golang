//go:build postgresql

package operation

// Finding-3 close-out shape tests: the job / job_activity list adapters order the
// OUTER `SELECT e.* FROM enriched e` subquery, so their sort whitelist + fallback
// must reference the outer alias `e`, never the inner `j` / `ja` (out of scope in
// the outer SELECT). These lock that the emitted ORDER BY is outer-scoped, quoted
// per component, and terminates in the id tiebreaker.

import (
	"strings"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

func sortByCol(col string, dir commonpb.SortDirection) *commonpb.SortRequest {
	return &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: col, Direction: dir}}}
}

func TestJobListOrderByOuterScope(t *testing.T) {
	cases := []struct {
		name  string
		cols  []string
		sort  *commonpb.SortRequest
		fb    string
		want  string
		inner string // inner alias that must NOT appear
	}{
		{"job_fallback", jobSortableSQLCols, nil, "e.date_created DESC",
			`ORDER BY e.date_created DESC, "id" ASC`, `"j".`},
		{"job_sort", jobSortableSQLCols, sortByCol("e.status", commonpb.SortDirection_ASC), "e.date_created DESC",
			`ORDER BY "e"."status" ASC, "id" ASC`, `"j".`},
		{"job_activity_fallback", jobActivitySortableSQLCols, nil, "e.date_created DESC",
			`ORDER BY e.date_created DESC, "id" ASC`, `"ja".`},
		{"job_activity_sort", jobActivitySortableSQLCols, sortByCol("e.total_cost", commonpb.SortDirection_DESC), "e.date_created DESC",
			`ORDER BY "e"."total_cost" DESC, "id" ASC`, `"ja".`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := postgresCore.BuildOrderBy(tc.cols, tc.sort, tc.fb)
			if err != nil {
				t.Fatalf("BuildOrderBy: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
			if strings.Contains(got, tc.inner) {
				t.Errorf("emitted clause leaks inner alias %s: %q", tc.inner, got)
			}
		})
	}
}
