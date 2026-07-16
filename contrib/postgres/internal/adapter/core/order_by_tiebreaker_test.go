//go:build postgresql

package core

// Regression tests for the platform-pagination fix: the generic List ORDER BY
// must always terminate in the unique primary key `id` so that LIMIT/OFFSET
// pages over rows that TIE on the leading sort key (e.g. many rows bulk-created
// under a single date_created) are deterministic, disjoint, and complete.
//
// Root cause pinned this wave: 130 job_templates sharing ONE date_created were
// paged with `ORDER BY date_created DESC LIMIT/OFFSET`; Postgres is free to return
// tied rows in a different order per fetch, so ~14 rows never surfaced on any page.

import (
	"sort"
	"strings"
	"testing"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// sortParams is a tiny constructor for a ListParams carrying one or more sort
// fields, keeping the table cases below terse.
func sortParams(fields ...*commonpb.SortField) *interfaces.ListParams {
	return &interfaces.ListParams{Sort: &commonpb.SortRequest{Fields: fields}}
}

func field(name string, dir commonpb.SortDirection) *commonpb.SortField {
	return &commonpb.SortField{Field: name, Direction: dir}
}

// TestBuildListOrderByAppendsIDTiebreaker locks the shape of the emitted ORDER BY:
// the default and every caller-specified sort must END in the unique `id` key,
// except when the caller already sorts by id (no redundant double-append).
func TestBuildListOrderByAppendsIDTiebreaker(t *testing.T) {
	cases := []struct {
		name   string
		params *interfaces.ListParams
		want   string
	}{
		{
			name:   "default_no_sort_appends_id",
			params: nil,
			want:   "ORDER BY date_created DESC, id ASC",
		},
		{
			name:   "empty_sort_request_appends_id",
			params: &interfaces.ListParams{Sort: &commonpb.SortRequest{}},
			want:   "ORDER BY date_created DESC, id ASC",
		},
		// NOTE: an unset NullOrder is the enum zero value NULLS_FIRST, so caller sort
		// fields carry " NULLS FIRST" — this is PRE-EXISTING base-List behavior that
		// the tiebreaker refactor preserves byte-for-byte. The appended `id ASC`
		// tiebreaker is an author-controlled literal with no NULLS suffix.
		{
			name:   "caller_asc_appends_id",
			params: sortParams(field("name", commonpb.SortDirection_ASC)),
			want:   "ORDER BY name ASC NULLS FIRST, id ASC",
		},
		{
			name:   "caller_desc_still_appends_ascending_id",
			params: sortParams(field("date_created", commonpb.SortDirection_DESC)),
			want:   "ORDER BY date_created DESC NULLS FIRST, id ASC",
		},
		{
			name: "multi_key_appends_id_last",
			params: sortParams(
				field("status", commonpb.SortDirection_ASC),
				field("amount", commonpb.SortDirection_DESC),
			),
			want: "ORDER BY status ASC NULLS FIRST, amount DESC NULLS FIRST, id ASC",
		},
		{
			name:   "caller_already_sorts_by_id_no_double_append",
			params: sortParams(field("id", commonpb.SortDirection_DESC)),
			want:   "ORDER BY id DESC NULLS FIRST",
		},
		{
			name: "caller_sorts_by_id_among_keys_no_double_append",
			params: sortParams(
				field("name", commonpb.SortDirection_ASC),
				field("id", commonpb.SortDirection_ASC),
			),
			want: "ORDER BY name ASC NULLS FIRST, id ASC NULLS FIRST",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildListOrderByClause(tc.params)
			if err != nil {
				t.Fatalf("buildListOrderByClause returned error: %v", err)
			}
			if got != tc.want {
				t.Errorf("ORDER BY mismatch\n got: %q\nwant: %q", got, tc.want)
			}
			// The determinism invariant: every emitted clause must reference the
			// unique PK so the total order is strict.
			if !strings.Contains(got, "id") {
				t.Errorf("emitted ORDER BY lacks the `id` tiebreaker (non-deterministic pagination): %q", got)
			}
		})
	}
}

// TestBuildListOrderByPreservesNullOrdering pins that the tiebreaker is additive:
// NULLS FIRST/LAST on a caller key survives, and `id ASC` is still appended.
func TestBuildListOrderByPreservesNullOrdering(t *testing.T) {
	params := sortParams(&commonpb.SortField{
		Field:     "planned_end",
		Direction: commonpb.SortDirection_ASC,
		NullOrder: commonpb.NullOrder_NULLS_LAST,
	})
	got, err := buildListOrderByClause(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ORDER BY planned_end ASC NULLS LAST, id ASC" {
		t.Errorf("NULL ordering not preserved with tiebreaker: %q", got)
	}
}

// TestBuildListOrderByRejectsInvalidSortField keeps the fail-closed identifier
// guard intact after the refactor.
func TestBuildListOrderByRejectsInvalidSortField(t *testing.T) {
	params := sortParams(field("name; DROP TABLE job--", commonpb.SortDirection_ASC))
	if _, err := buildListOrderByClause(params); err == nil {
		t.Fatal("expected an error for a non-identifier sort field, got nil")
	}
}

// --- Pagination disjoint+complete guard --------------------------------------
//
// There is no in-memory harness for the live List (it executes against a real
// DB), so this test models Postgres's DOCUMENTED freedom to return tied rows in
// any order per fetch and proves that the clause buildListOrderByClause emits
// makes LIMIT/OFFSET pages disjoint+complete regardless of that freedom. If a
// future change drops the `id` tiebreaker, the emitted clause no longer resolves
// ties and this test fails — exactly the bug it guards.

type tiedRow struct {
	id          int
	dateCreated int64 // identical across all rows: every row ties on the leading key
}

// paginateTiedRows returns the ids in the [offset, offset+limit) window after
// ordering rows by the given ORDER BY clause. It honors ONLY the two columns the
// fixture carries (date_created, id). fetchSeed models a distinct query execution:
// when the clause does not fully disambiguate tied rows, Postgres may hand them
// back in a different order — simulated here by reversing each unresolved tied
// block on an odd seed. When the clause ends in the unique `id`, no ties remain,
// so fetchSeed has no effect (deterministic).
func paginateTiedRows(orderBy string, rows []tiedRow, limit, offset, fetchSeed int) []int {
	resolvesByID := containsIDKey(orderBy)
	descByDate := strings.Contains(orderBy, "date_created DESC")

	ordered := make([]tiedRow, len(rows))
	copy(ordered, rows)

	if resolvesByID {
		descByID := strings.Contains(orderBy, "id DESC")
		sort.SliceStable(ordered, func(i, j int) bool {
			if descByID {
				return ordered[i].id > ordered[j].id
			}
			return ordered[i].id < ordered[j].id
		})
	} else {
		// All rows tie on date_created and nothing disambiguates them → the fetch
		// is free to reorder the whole tied block. Model that per-fetch freedom.
		_ = descByDate
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

// containsIDKey reports whether the ORDER BY names the `id` column as a sort key.
func containsIDKey(orderBy string) bool {
	body := strings.TrimPrefix(orderBy, "ORDER BY ")
	for _, part := range strings.Split(body, ",") {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) > 0 && fields[0] == "id" {
			return true
		}
	}
	return false
}

func TestListPaginationDisjointCompleteOverTiedRows(t *testing.T) {
	// Mirror the real incident: 130 rows sharing ONE date_created, page size 100.
	const total, pageSize = 130, 100
	rows := make([]tiedRow, total)
	for i := range rows {
		rows[i] = tiedRow{id: i, dateCreated: 1_700_000_000_000}
	}

	clause, err := buildListOrderByClause(nil) // default: date_created DESC, id ASC
	if err != nil {
		t.Fatalf("buildListOrderByClause: %v", err)
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
