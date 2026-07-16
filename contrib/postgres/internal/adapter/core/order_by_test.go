//go:build postgresql

package core

// Regression tests for the platform-pagination fix extended to core.BuildOrderBy,
// the sort-whitelist helper consumed by the hand-written LIMIT/OFFSET list-page
// adapters (~110 call sites). Like the base List.buildListOrderByClause fix, every
// emitted ORDER BY must terminate in a UNIQUE id key so LIMIT/OFFSET pages over
// rows that TIE on the leading sort key are deterministic, disjoint, and complete.
//
// The cases below assert the emitted clause per CALLER CLASS:
//   - bare-table / single-relation outer (bare table, single aliased table, or a
//     CTE-wrapped `SELECT ... FROM enriched e` subquery alias) → default "id".
//   - joined outer (two+ relations with `id` in scope) → an alias-qualified id.
// In every case the id key must terminate the clause EXACTLY ONCE. The whitelisted
// sort column AND the tiebreaker are double-quoted PER COMPONENT (`"a"."b"`), and
// the no-double-append detector + this counter are quote-aware (embedded commas /
// canonical `"rb"."id"` forms handled).

import (
	"strings"
	"testing"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// sortReqOf builds a *commonpb.SortRequest from sort fields (BuildOrderBy takes a
// SortRequest, unlike buildListOrderByClause which takes ListParams). Reuses the
// package-local field() helper defined in order_by_tiebreaker_test.go.
func sortReqOf(fields ...*commonpb.SortField) *commonpb.SortRequest {
	return &commonpb.SortRequest{Fields: fields}
}

// countIDKeyTerminations reports how many sort segments of the ORDER BY body name
// an id key — reusing the production quote-aware parsing (splitOutsideQuotes /
// leadingSortToken / finalPathComponent) so the test agrees exactly with the
// helper's own no-double-append logic. Used to assert the tiebreaker terminates
// the clause exactly once.
func countIDKeyTerminations(orderBy string) int {
	body := strings.TrimPrefix(orderBy, "ORDER BY ")
	n := 0
	for _, seg := range splitOutsideQuotes(body, ',') {
		tok := leadingSortToken(seg)
		if tok == "" {
			continue
		}
		if finalPathComponent(tok) == "id" {
			n++
		}
	}
	return n
}

func TestBuildOrderByTiebreaker(t *testing.T) {
	// Whitelist covers bare columns plus a dotted qualified entry (rate_band's
	// `rb.id` shape) and a pathological quoted-comma column, to exercise the
	// quote-aware paths.
	allowed := []string{"name", "date_created", "id", "amount", "rb.id", "rb.ordinal", "x,id"}

	cases := []struct {
		name        string
		sort        *commonpb.SortRequest
		fallback    string
		tiebreaker  []string // variadic passthrough; nil = default "id"
		want        string
		wantErr     bool
		wantIDCount int
	}{
		// --- CLASS: bare-table / single-relation outer (default "id") ---------
		{
			name:        "bare_default_fallback_appends_id",
			sort:        nil,
			fallback:    "date_created DESC",
			want:        `ORDER BY date_created DESC, "id" ASC`,
			wantIDCount: 1,
		},
		{
			name:        "bare_empty_sort_uses_fallback_plus_id",
			sort:        sortReqOf(),
			fallback:    "name ASC",
			want:        `ORDER BY name ASC, "id" ASC`,
			wantIDCount: 1,
		},
		{
			name:        "bare_caller_asc_appends_id",
			sort:        sortReqOf(field("name", commonpb.SortDirection_ASC)),
			fallback:    "date_created DESC",
			want:        `ORDER BY "name" ASC, "id" ASC`,
			wantIDCount: 1,
		},
		{
			name:        "bare_caller_desc_still_appends_ascending_id",
			sort:        sortReqOf(field("date_created", commonpb.SortDirection_DESC)),
			fallback:    "date_created DESC",
			want:        `ORDER BY "date_created" DESC, "id" ASC`,
			wantIDCount: 1,
		},
		{
			// Subquery-alias class (`SELECT * FROM enriched`): the exposed column is
			// a bare, quoted name — identical emitted shape to the bare-table class,
			// which is why the default "id" is correct for CTE-wrapped callers.
			name:        "subquery_alias_quoted_col_appends_bare_id",
			sort:        sortReqOf(field("amount", commonpb.SortDirection_DESC)),
			fallback:    "date_created DESC",
			want:        `ORDER BY "amount" DESC, "id" ASC`,
			wantIDCount: 1,
		},
		{
			// Caller already sorts by id → no redundant double-append. The canonical
			// quoted `"id"` form must be recognized by the detector.
			name:        "caller_sorts_by_id_no_double_append",
			sort:        sortReqOf(field("id", commonpb.SortDirection_DESC)),
			fallback:    "date_created DESC",
			want:        `ORDER BY "id" DESC`,
			wantIDCount: 1,
		},

		// --- FINDING 3: quote/qualifier handling -----------------------------
		{
			// A dotted whitelist entry (`rb.id`) must emit the qualified reference
			// `"rb"."id"` (valid) — NOT the single identifier `"rb.id"` (undefined
			// column). Because its FINAL component is `id`, the detector recognizes
			// it and suppresses a duplicate tiebreaker.
			name:        "dotted_whitelist_id_emits_per_component_and_no_double",
			sort:        sortReqOf(field("rb.id", commonpb.SortDirection_DESC)),
			fallback:    "rb.ordinal ASC",
			tiebreaker:  []string{"rb.id"},
			want:        `ORDER BY "rb"."id" DESC`,
			wantIDCount: 1,
		},
		{
			// A pathological column literally named `x,id`: quoted as one identifier
			// `"x,id"`, the embedded comma must NOT be split into a phantom `id`
			// segment, so the real tiebreaker is still appended (exactly one id key).
			name:        "quoted_embedded_comma_col_not_phantom_id",
			sort:        sortReqOf(field("x,id", commonpb.SortDirection_ASC)),
			fallback:    "date_created DESC",
			want:        `ORDER BY "x,id" ASC, "id" ASC`,
			wantIDCount: 1,
		},

		// --- CLASS: joined outer (explicit alias-qualified tiebreaker) --------
		{
			// rate_band shape: FROM rate_band rb LEFT JOIN rate_table rt.
			name:        "joined_default_fallback_appends_alias_id",
			sort:        nil,
			fallback:    "rb.ordinal ASC",
			tiebreaker:  []string{"rb.id"},
			want:        `ORDER BY rb.ordinal ASC, "rb"."id" ASC`,
			wantIDCount: 1,
		},
		{
			// payroll_remittance / cost_plan shape: caller sort on a quoted bare col
			// + an alias-qualified id tiebreaker terminating the joined-outer clause.
			name:        "joined_caller_sort_appends_alias_id",
			sort:        sortReqOf(field("date_created", commonpb.SortDirection_DESC)),
			fallback:    "rem.date_created DESC",
			tiebreaker:  []string{"rem.id"},
			want:        `ORDER BY "date_created" DESC, "rem"."id" ASC`,
			wantIDCount: 1,
		},
		{
			// Empty-string tiebreaker keeps the "id" default (explicit opt-in only).
			name:        "empty_tiebreaker_string_falls_back_to_id",
			sort:        nil,
			fallback:    "date_created DESC",
			tiebreaker:  []string{""},
			want:        `ORDER BY date_created DESC, "id" ASC`,
			wantIDCount: 1,
		},

		// --- fail-closed guards preserved ------------------------------------
		{
			name:     "unknown_sort_column_errors",
			sort:     sortReqOf(field("password", commonpb.SortDirection_ASC)),
			fallback: "date_created DESC",
			wantErr:  true,
		},
		{
			// FINDING 4: a request-shaped injection string as the tiebreaker is
			// rejected fail-closed rather than interpolated verbatim.
			name:       "injection_tiebreaker_errors",
			sort:       nil,
			fallback:   "date_created DESC",
			tiebreaker: []string{"id ASC; DROP TABLE x; --"},
			wantErr:    true,
		},
		{
			// FINDING 4: a non-id tiebreaker silently defeats the uniqueness
			// invariant → rejected.
			name:       "non_id_tiebreaker_errors",
			sort:       nil,
			fallback:   "date_created DESC",
			tiebreaker: []string{"rb.name"},
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := BuildOrderBy(allowed, tc.sort, tc.fallback, tc.tiebreaker...)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got clause %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("BuildOrderBy returned error: %v", err)
			}
			if got != tc.want {
				t.Errorf("ORDER BY mismatch\n got: %q\nwant: %q", got, tc.want)
			}
			// Determinism invariant: the emitted clause references a unique id key
			// exactly once (a missing key = nondeterministic pagination; a doubled
			// key = a redundant/duplicate sort column).
			if n := countIDKeyTerminations(got); n != tc.wantIDCount {
				t.Errorf("id-key count = %d, want %d in %q", n, tc.wantIDCount, got)
			}
		})
	}
}

// TestBuildOrderByRejectsInjectionColumn keeps the fail-closed whitelist intact
// after the tiebreaker refactor: a non-whitelisted (and injection-shaped) column
// still errors rather than being interpolated.
func TestBuildOrderByRejectsInjectionColumn(t *testing.T) {
	allowed := []string{"name", "date_created"}
	_, err := BuildOrderBy(allowed, sortReqOf(field("name; DROP TABLE x--", commonpb.SortDirection_ASC)), "date_created DESC")
	if err == nil {
		t.Fatal("expected an error for a non-whitelisted sort column, got nil")
	}
}

// TestQuoteSortIdentPerComponent locks the per-component quoting fix directly: a
// qualified column becomes "a"."b" (valid), never the single identifier "a.b".
func TestQuoteSortIdentPerComponent(t *testing.T) {
	cases := map[string]string{
		"id":           `"id"`,
		"date_created": `"date_created"`,
		"rb.id":        `"rb"."id"`,
		"x,id":         `"x,id"`, // no dot → one component, comma stays inside quotes
	}
	for in, want := range cases {
		if got := quoteSortIdent(in); got != want {
			t.Errorf("quoteSortIdent(%q) = %q, want %q", in, got, want)
		}
	}
}
