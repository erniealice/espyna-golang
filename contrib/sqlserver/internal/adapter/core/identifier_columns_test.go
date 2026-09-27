//go:build sqlserver

package core

import (
	"context"
	"testing"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

func TestIdentifierFilter(t *testing.T) {
	s := &SQLServerOperations{dialect: DefaultDialect}
	for column, want := range map[string]bool{
		"id": true, "origin_id": true, "workspace_id": true,
		"internal_id": false, "tax_id": false, "name": false,
	} {
		if got := s.isIdentifierColumn(context.Background(), "job", column); got != want {
			t.Errorf("isIdentifierColumn(%q) = %v, want %v", column, got, want)
		}
	}
	for _, tc := range []struct {
		name      string
		op        commonpb.StringOperator
		exact     bool
		wantSQL   string
		wantValue string
	}{
		{"id equality", commonpb.StringOperator_STRING_EQUALS, true, "[origin_id] = @p1", "abc"},
		{"id inequality", commonpb.StringOperator_STRING_NOT_EQUALS, true, "[origin_id] != @p1", "abc"},
		{"text equality", commonpb.StringOperator_STRING_EQUALS, false, "LOWER([origin_id]) = @p1", "abc"},
		{"id pattern", commonpb.StringOperator_STRING_CONTAINS, true, "LOWER([origin_id]) LIKE @p1", "%abc%"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotSQL, gotValues, next := s.buildStringFilter("origin_id", &commonpb.StringFilter{
				Value: "AbC", Operator: tc.op,
			}, tc.exact, 1)
			if gotSQL != tc.wantSQL || len(gotValues) != 1 || gotValues[0] != tc.wantValue || next != 2 {
				t.Fatalf("got (%q, %v, %d), want (%q, [%q], 2)", gotSQL, gotValues, next, tc.wantSQL, tc.wantValue)
			}
		})
	}
}
