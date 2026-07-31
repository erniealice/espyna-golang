//go:build sqlserver

package core

// Ported from contrib/mysql/internal/adapter/core/order_by_tiebreaker_test.go
// (itself ported from the postgres original), with expectations re-quoted to
// T-SQL square brackets.
//
// PORT DEVIATION: the postgres original also tests buildListOrderByClause, the
// GENERIC base-List ORDER BY builder (postgres core/operations.go). The sqlserver
// core package has no such function — its generic List inlines the ordering logic
// in operations.go — so those base-List cases cannot compile here and are
// intentionally not ported. What carries over:
//   - the package-local field() helper (shared with order_by_test.go),
//   - the pagination disjoint+complete simulation, re-targeted at BuildOrderBy
//     (the helper this wave gives the tiebreaker), with quote-aware id detection
//     via the production orderByHasIDKey instead of a local bare-token scan
//     (sqlserver emits a bracket-quoted [id]).
//
// The simulation models SQL Server's freedom to return tied rows in any order per
// fetch and proves the clause BuildOrderBy emits makes OFFSET/FETCH pages
// disjoint+complete regardless of that freedom. If a future change drops the
// [id] tiebreaker, the emitted clause no longer resolves ties and this test
// fails — exactly the bug it guards.

import (
	"sort"
	"strings"
	"testing"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

func field(name string, dir commonpb.SortDirection) *commonpb.SortField {
	return &commonpb.SortField{Field: name, Direction: dir}
}

// --- Pagination disjoint+complete guard --------------------------------------

type tiedRow struct {
	id          int
	dateCreated int64 // identical across all rows: every row ties on the leading key
}

// paginateTiedRows returns the ids in the [offset, offset+limit) window after
// ordering rows by the given ORDER BY clause. It honors ONLY the two columns the
// fixture carries (date_created, id). fetchSeed models a distinct query execution:
// when the clause does not fully disambiguate tied rows, SQL Server may hand them
// back in a different order — simulated here by reversing each unresolved tied
// block on an odd seed. When the clause ends in the unique [id], no ties remain,
// so fetchSeed has no effect (deterministic). Id-key detection reuses the
// production quote-aware orderByHasIDKey so the bracket-quoted [id] form is
// recognized.
func paginateTiedRows(orderBy string, rows []tiedRow, limit, offset, fetchSeed int) []int {
	body := strings.TrimPrefix(orderBy, "ORDER BY ")
	resolvesByID := orderByHasIDKey(body)

	ordered := make([]tiedRow, len(rows))
	copy(ordered, rows)

	if resolvesByID {
		descByID := strings.Contains(body, "[id] DESC")
		sort.SliceStable(ordered, func(i, j int) bool {
			if descByID {
				return ordered[i].id > ordered[j].id
			}
			return ordered[i].id < ordered[j].id
		})
	} else {
		// All rows tie on date_created and nothing disambiguates them → the fetch
		// is free to reorder the whole tied block. Model that per-fetch freedom.
		if fetchSeed%2 == 1 {
			for i, j := 0, len(ordered)-1; i < j; i, j = i+1, j-1 {
				ordered[i], ordered[j] = ordered[j], ordered[i]
			}
		}
	}

	out := []int{}
	for i := offset; i < offset+limit && i < len(ordered); i++ {
		out = append(out, ordered[i].id)
	}
	return out
}

func TestListPaginationDisjointCompleteOverTiedRows(t *testing.T) {
	// Mirror the real incident: 130 rows sharing ONE date_created, page size 100.
	const total, pageSize = 130, 100
	rows := make([]tiedRow, total)
	for i := range rows {
		rows[i] = tiedRow{id: i, dateCreated: 1_700_000_000_000}
	}

	// Default fallback path: BuildOrderBy appends the unique [id] tiebreaker.
	clause, err := BuildOrderBy([]string{"date_created"}, nil, "date_created DESC")
	if err != nil {
		t.Fatalf("BuildOrderBy: %v", err)
	}
	if want := "ORDER BY date_created DESC, [id] ASC"; clause != want {
		t.Fatalf("default clause = %q, want %q", clause, want)
	}

	// Two separate fetches (page 1, page 2) modeled as distinct query executions
	// with different fetch orderings of any unresolved tied block.
	page1 := paginateTiedRows(clause, rows, pageSize, 0, 0)
	page2 := paginateTiedRows(clause, rows, pageSize, pageSize, 1)

	seen := make(map[int]int, total)
	for _, id := range page1 {
		seen[id]++
	}
	for _, id := range page2 {
		seen[id]++
	}

	// Complete: every row appears.
	if len(seen) != total {
		t.Errorf("pagination lost rows: covered %d distinct ids, want %d", len(seen), total)
	}
	// Disjoint: no row appears on both pages.
	for id, n := range seen {
		if n != 1 {
			t.Errorf("row id=%d appeared %d times across pages (want exactly 1)", id, n)
		}
	}

	// Control: prove the fixture actually models the bug — the SAME simulation over
	// an un-tiebroken clause is NOT disjoint+complete, so the guard above has teeth.
	buggy := "ORDER BY date_created DESC"
	bp1 := paginateTiedRows(buggy, rows, pageSize, 0, 0)
	bp2 := paginateTiedRows(buggy, rows, pageSize, pageSize, 1)
	bseen := map[int]int{}
	for _, id := range append(append([]int{}, bp1...), bp2...) {
		bseen[id]++
	}
	if len(bseen) == total {
		t.Error("control failed: un-tiebroken clause unexpectedly covered all rows; simulation does not model the bug")
	}
}
