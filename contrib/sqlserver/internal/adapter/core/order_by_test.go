//go:build sqlserver

package core

// Regression tests for core.BuildOrderBy, the sort-whitelist helper consumed by
// the hand-written OFFSET/FETCH list-page adapters. Ported from
// contrib/mysql/internal/adapter/core/order_by_test.go with expectations
// re-quoted to T-SQL square brackets. Every emitted ORDER BY must terminate in a
// UNIQUE id key so OFFSET/FETCH pages over rows that TIE on the leading sort key
// are deterministic, disjoint, and complete.
//
// The cases below assert the emitted clause per CALLER CLASS:
//   - bare-table / single-relation outer (bare table, single aliased table, or a
//     CTE-wrapped `SELECT ... FROM enriched` subquery alias) → default "id".
//   - joined outer (two+ relations with `id` in scope) → an alias-qualified id.
// In every case the id key must terminate the clause EXACTLY ONCE. The whitelisted
// sort column AND the tiebreaker are bracket-quoted PER COMPONENT ([a].[b]), and
// the no-double-append detector + this counter are quote-aware (embedded commas /
// canonical [rb].[id] forms handled).

import (
	"strings"
	"testing"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// sortReqOf builds a *commonpb.SortRequest from sort fields (BuildOrderBy takes a
// SortRequest). Reuses the package-local field() helper defined in
// order_by_tiebreaker_test.go.
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
	// Whitelist covers bare columns plus dotted qualified entries (payroll_run's
	// `pr.date_created` shape, rate_band's `rb.id` shape) and a pathological
	// embedded-comma column, to exercise the quote-aware paths.
	allowed := []string{"name", "date_created", "id", "amount", "rb.id", "rb.ordinal", "x,id", "pr.date_created"}

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
			want:        "ORDER BY date_created DESC, [id] ASC",
			wantIDCount: 1,
		},
		{
			name:        "bare_empty_sort_uses_fallback_plus_id",
			sort:        sortReqOf(),
			fallback:    "name ASC",
			want:        "ORDER BY name ASC, [id] ASC",
			wantIDCount: 1,
		},
		{
			name:        "bare_caller_asc_appends_id",
			sort:        sortReqOf(field("name", commonpb.SortDirection_ASC)),
			fallback:    "date_created DESC",
			want:        "ORDER BY [name] ASC, [id] ASC",
			wantIDCount: 1,
		},
		{
			name:        "bare_caller_desc_still_appends_ascending_id",
			sort:        sortReqOf(field("date_created", commonpb.SortDirection_DESC)),
			fallback:    "date_created DESC",
			want:        "ORDER BY [date_created] DESC, [id] ASC",
			wantIDCount: 1,
		},
		{
			// Subquery-alias class (`SELECT * FROM enriched`): the exposed column is
			// a bare, quoted name — identical emitted shape to the bare-table class,
			// which is why the default "id" is correct for CTE-wrapped callers.
			name:        "subquery_alias_quoted_col_appends_bare_id",
			sort:        sortReqOf(field("amount", commonpb.SortDirection_DESC)),
			fallback:    "date_created DESC",
			want:        "ORDER BY [amount] DESC, [id] ASC",
			wantIDCount: 1,
		},
		{
			// Caller already sorts by id → no redundant double-append. The canonical
			// quoted [id] form must be recognized by the detector.
			name:        "caller_sorts_by_id_no_double_append",
			sort:        sortReqOf(field("id", commonpb.SortDirection_DESC)),
			fallback:    "date_created DESC",
			want:        "ORDER BY [id] DESC",
			wantIDCount: 1,
		},

		// --- dotted-identifier quoting (plan §5 step 1) -----------------------
		{
			// THE case this wave fixes: a dotted whitelist entry must be quoted PER
			// COMPONENT — [pr].[date_created] (valid qualified reference), never the
			// single identifier [pr.date_created] in ONE bracket pair (T-SQL reads
			// that as a column literally named "pr.date_created" → Invalid column
			// name; payroll_run's whitelist is all-dotted, so its sorts were broken).
			name:        "dotted column quotes per component",
			sort:        sortReqOf(field("pr.date_created", commonpb.SortDirection_DESC)),
			fallback:    "pr.date_created DESC",
			want:        "ORDER BY [pr].[date_created] DESC, [id] ASC",
			wantIDCount: 1,
		},

		// --- quote/qualifier handling -----------------------------------------
		{
			// A dotted whitelist entry (`rb.id`) must emit the qualified reference
			// [rb].[id] (valid) — NOT the single identifier [rb.id] in one pair
			// (undefined column). Because its FINAL component is `id`, the detector
			// recognizes it and suppresses a duplicate tiebreaker.
			name:        "dotted_whitelist_id_emits_per_component_and_no_double",
			sort:        sortReqOf(field("rb.id", commonpb.SortDirection_DESC)),
			fallback:    "rb.ordinal ASC",
			tiebreaker:  []string{"rb.id"},
			want:        "ORDER BY [rb].[id] DESC",
			wantIDCount: 1,
		},
		{
			// A pathological column literally named `x,id`: quoted as one identifier
			// [x,id] in a single bracket pair, the embedded comma must NOT be split
			// into a phantom id segment, so the real tiebreaker is still appended
			// (exactly one id key).
			name:        "quoted_embedded_comma_col_not_phantom_id",
			sort:        sortReqOf(field("x,id", commonpb.SortDirection_ASC)),
			fallback:    "date_created DESC",
			want:        "ORDER BY [x,id] ASC, [id] ASC",
			wantIDCount: 1,
		},

		// --- CLASS: joined outer (explicit alias-qualified tiebreaker) --------
		{
			// rate_band shape: FROM rate_band rb LEFT JOIN rate_table rt.
			name:        "joined_default_fallback_appends_alias_id",
			sort:        nil,
			fallback:    "rb.ordinal ASC",
			tiebreaker:  []string{"rb.id"},
			want:        "ORDER BY rb.ordinal ASC, [rb].[id] ASC",
			wantIDCount: 1,
		},
		{
			// payroll_remittance shape: caller sort on a quoted bare col + an
			// alias-qualified id tiebreaker terminating the joined-outer clause.
			name:        "joined_caller_sort_appends_alias_id",
			sort:        sortReqOf(field("date_created", commonpb.SortDirection_DESC)),
			fallback:    "rem.date_created DESC",
			tiebreaker:  []string{"rem.id"},
			want:        "ORDER BY [date_created] DESC, [rem].[id] ASC",
			wantIDCount: 1,
		},
		{
			// Empty-string tiebreaker keeps the "id" default (explicit opt-in only).
			name:        "empty_tiebreaker_string_falls_back_to_id",
			sort:        nil,
			fallback:    "date_created DESC",
			tiebreaker:  []string{""},
			want:        "ORDER BY date_created DESC, [id] ASC",
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
			// A request-shaped injection string as the tiebreaker is rejected
			// fail-closed rather than interpolated verbatim.
			name:       "injection_tiebreaker_errors",
			sort:       nil,
			fallback:   "date_created DESC",
			tiebreaker: []string{"id ASC; DROP TABLE x; --"},
			wantErr:    true,
		},
		{
			// A T-SQL-flavoured bracket-escape injection attempt is rejected too:
			// ValidateSQLIdent admits no brackets at all.
			name:       "bracket_injection_tiebreaker_errors",
			sort:       nil,
			fallback:   "date_created DESC",
			tiebreaker: []string{"id] ASC; DROP TABLE x; --"},
			wantErr:    true,
		},
		{
			// A non-id tiebreaker silently defeats the uniqueness invariant →
			// rejected.
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
// qualified column becomes [a].[b] (valid), never the single identifier [a.b]
// inside one bracket pair. It also locks the T-SQL escape rule (a ']' inside an
// identifier is doubled), which is the sqlserver-specific half of this wave.
func TestQuoteSortIdentPerComponent(t *testing.T) {
	cases := map[string]string{
		"id":              "[id]",
		"date_created":    "[date_created]",
		"rb.id":           "[rb].[id]",
		"pr.date_created": "[pr].[date_created]",
		"x,id":            "[x,id]", // no dot → one component, comma stays inside brackets
		"x]y":             "[x]]y]", // T-SQL escape: a closing bracket is doubled
	}
	for in, want := range cases {
		if got := quoteSortIdent(in); got != want {
			t.Errorf("quoteSortIdent(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestBracketAwareParsingRoundTrip locks the bracket-aware scanners against the
// escaped-bracket form quoteSortIdent can emit: an identifier carrying a doubled
// ']' must not terminate the quoted region early, so a comma or dot inside it is
// never treated as a separator and the unquoted final component round-trips.
func TestBracketAwareParsingRoundTrip(t *testing.T) {
	quoted := quoteSortIdent("a]b,c") // → [a]]b,c]
	if quoted != "[a]]b,c]" {
		t.Fatalf("quoteSortIdent = %q, want %q", quoted, "[a]]b,c]")
	}
	if parts := splitOutsideQuotes(quoted+" ASC, [id] ASC", ','); len(parts) != 2 {
		t.Errorf("splitOutsideQuotes produced %d segments (%q), want 2 — the escaped ']' closed the region early", len(parts), parts)
	}
	if got := leadingSortToken(quoted + " DESC"); got != quoted {
		t.Errorf("leadingSortToken = %q, want %q", got, quoted)
	}
	if got := finalPathComponent(quoted); got != "a]b,c" {
		t.Errorf("finalPathComponent = %q, want %q", got, "a]b,c")
	}
}
