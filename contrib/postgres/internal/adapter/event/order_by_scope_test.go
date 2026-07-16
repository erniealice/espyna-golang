//go:build postgresql

package event

// Finding-3 close-out shape test: event_occurrence's list adapter orders the OUTER
// `SELECT e.* FROM enriched e` subquery, so its sort whitelist + map + fallback must
// reference the outer alias `e`, never the inner `eo`. Locks the emitted ORDER BY as
// outer-scoped, per-component quoted, terminating in the id tiebreaker.

import (
	"strings"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

func TestEventOccurrenceListOrderByOuterScope(t *testing.T) {
	sortReq := func(col string, dir commonpb.SortDirection) *commonpb.SortRequest {
		return &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: col, Direction: dir}}}
	}
	cases := []struct {
		name string
		sort *commonpb.SortRequest
		want string
	}{
		{"fallback", nil, `ORDER BY e.start_date_time_utc ASC, "id" ASC`},
		{"sort", sortReq("e.event_id", commonpb.SortDirection_ASC), `ORDER BY "e"."event_id" ASC, "id" ASC`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := postgresCore.BuildOrderBy(eventOccurrenceSortableSQLCols, tc.sort, "e.start_date_time_utc ASC")
			if err != nil {
				t.Fatalf("BuildOrderBy: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
			if strings.Contains(got, `"eo".`) {
				t.Errorf("emitted clause leaks inner alias eo: %q", got)
			}
		})
	}
}
