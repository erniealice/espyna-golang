//go:build mysql

package core

// Regression tests for core.BuildFilterWhere, the shared filter/search WHERE
// builder consumed by the hand-written list-page adapters. Filter VALUES are
// bound as ? placeholders (categorically injection-proof); filter FIELD names
// and search columns are interpolated into query text, so every one must pass
// ValidateSQLIdent and the call must FAIL CLOSED (error, nil clauses) on the
// first bad identifier — never silently drop the offending filter, which would
// widen the result set (fail-open). Mirrors the guard semantics of
// contrib/postgres/internal/adapter/core/filter_builder.go so the two dialect
// suites read as twins; expectations are backtick-quoted (MySQL) and use LIKE
// (postgres: double quotes + ILIKE).

import (
	"testing"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// stringEqualsFilter builds a FilterRequest with a single STRING_EQUALS filter
// on the given field — the minimal shape that forces the field name through the
// interpolation path.
func stringEqualsFilter(field string) *commonpb.FilterRequest {
	return &commonpb.FilterRequest{
		Filters: []*commonpb.TypedFilter{
			{
				Field: field,
				FilterType: &commonpb.TypedFilter_StringFilter{
					StringFilter: &commonpb.StringFilter{
						Value:    "v",
						Operator: commonpb.StringOperator_STRING_EQUALS,
					},
				},
			},
		},
	}
}

func TestBuildFilterWhereFieldGuard(t *testing.T) {
	tests := []struct {
		name       string
		field      string
		wantErr    bool
		wantClause string // asserted only when wantErr == false
	}{
		{
			name:       "bare identifier accepted",
			field:      "name",
			wantErr:    false,
			wantClause: "`name` = ?",
		},
		{
			name:    "injection string rejected",
			field:   "name; DROP TABLE user--",
			wantErr: true,
		},
		{
			name:       "single-qualified identifier accepted",
			field:      "a.b",
			wantErr:    false,
			wantClause: "`a`.`b` = ?",
		},
		{
			name:    "double-qualified identifier rejected",
			field:   "a.b.c",
			wantErr: true,
		},
		{
			// The empty-string trap (infra-sql-no-direct-sql-rule.md): a blank
			// field must be REJECTED, not allowed to skip the guard and
			// interpolate an empty identifier into query text.
			name:    "empty field rejected",
			field:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clauses, args, nextIdx, err := BuildFilterWhere(
				stringEqualsFilter(tt.field), nil, nil, 2)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("BuildFilterWhere(%q) err = nil, want error (fail closed)", tt.field)
				}
				if clauses != nil {
					t.Fatalf("BuildFilterWhere(%q) clauses = %v, want nil on error", tt.field, clauses)
				}
				if args != nil {
					t.Fatalf("BuildFilterWhere(%q) args = %v, want nil on error", tt.field, args)
				}
				if nextIdx != 2 {
					t.Fatalf("BuildFilterWhere(%q) nextIdx = %d, want startIdx 2 on error", tt.field, nextIdx)
				}
				return
			}

			if err != nil {
				t.Fatalf("BuildFilterWhere(%q) unexpected err: %v", tt.field, err)
			}
			if len(clauses) != 1 {
				t.Fatalf("BuildFilterWhere(%q) clauses = %v, want exactly 1", tt.field, clauses)
			}
			if clauses[0] != tt.wantClause {
				t.Errorf("BuildFilterWhere(%q) clause = %q, want %q", tt.field, clauses[0], tt.wantClause)
			}
			if len(args) != 1 || args[0] != "v" {
				t.Errorf("BuildFilterWhere(%q) args = %v, want [v]", tt.field, args)
			}
			if nextIdx != 3 {
				t.Errorf("BuildFilterWhere(%q) nextIdx = %d, want 3", tt.field, nextIdx)
			}
		})
	}
}

func TestBuildFilterWhereSearchFieldGuard(t *testing.T) {
	search := &commonpb.SearchRequest{Query: "abc"}

	t.Run("injection search field rejected", func(t *testing.T) {
		clauses, args, nextIdx, err := BuildFilterWhere(
			nil, search, []string{"n) OR 1=1 --"}, 2)
		if err == nil {
			t.Fatal("BuildFilterWhere err = nil, want error (fail closed)")
		}
		if clauses != nil || args != nil {
			t.Fatalf("clauses = %v, args = %v, want nil on error", clauses, args)
		}
		if nextIdx != 2 {
			t.Fatalf("nextIdx = %d, want startIdx 2 on error", nextIdx)
		}
	})

	t.Run("bad field among good ones still fails closed", func(t *testing.T) {
		_, _, _, err := BuildFilterWhere(
			nil, search, []string{"c.name", "n) OR 1=1 --"}, 2)
		if err == nil {
			t.Fatal("BuildFilterWhere err = nil, want error (fail closed on ANY bad field)")
		}
	})

	t.Run("valid search fields emit quoted LIKE OR block", func(t *testing.T) {
		clauses, args, nextIdx, err := BuildFilterWhere(
			nil, search, []string{"c.name", "internal_id"}, 2)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		want := "(`c`.`name` LIKE ? OR `internal_id` LIKE ?)"
		if len(clauses) != 1 || clauses[0] != want {
			t.Errorf("clauses = %v, want [%s]", clauses, want)
		}
		if len(args) != 2 || args[0] != "%abc%" || args[1] != "%abc%" {
			t.Errorf("args = %v, want [%%abc%% %%abc%%]", args)
		}
		if nextIdx != 4 {
			t.Errorf("nextIdx = %d, want 4", nextIdx)
		}
	})
}

func TestBuildFilterWhereNoInput(t *testing.T) {
	clauses, args, nextIdx, err := BuildFilterWhere(nil, nil, nil, 5)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if clauses != nil || args != nil {
		t.Errorf("clauses = %v, args = %v, want nil/nil with no input", clauses, args)
	}
	if nextIdx != 5 {
		t.Errorf("nextIdx = %d, want startIdx 5 passthrough", nextIdx)
	}
}
