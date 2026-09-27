//go:build mysql

package core

import (
	"context"
	"strings"
	"testing"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

func TestListRequestRejectsUnknownColumnAndInvalidCursor(t *testing.T) {
	allowed := map[string]struct{}{"id": {}, "workspace_id": {}, "name": {}}
	params := &interfaces.ListParams{Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
		Field: "secret", FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
			Operator: commonpb.StringOperator_STRING_EQUALS, Value: "x",
		}},
	}}}}
	if err := validateListRequest(params, "client", allowed); err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("unknown column should fail closed, got %v", err)
	}
	params.Filters = nil
	params.Pagination = &commonpb.PaginationRequest{Method: &commonpb.PaginationRequest_Cursor{
		Cursor: &commonpb.CursorPagination{Token: "offset:not-a-number"},
	}}
	if err := validateListRequest(params, "client", allowed); err == nil {
		t.Fatal("malformed cursor should fail closed")
	}
}

func TestWorkspaceScopeRemainsOutsideCallerOR(t *testing.T) {
	m := &MySQLOperations{dialect: NewMySQLDialect()}
	caller := &commonpb.FilterRequest{Logic: commonpb.FilterLogic_OR, Filters: []*commonpb.TypedFilter{
		{Field: "name", FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Operator: commonpb.StringOperator_STRING_EQUALS, Value: "Alice"}}},
		{Field: "status", FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Operator: commonpb.StringOperator_STRING_EQUALS, Value: "Active"}}},
	}}
	scopeSQL, scopeArgs, next := m.buildFilterConditions(context.Background(), "client", &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{workspaceIDFilter("tenant-1")}}, 1)
	callerSQL, callerArgs, _ := m.buildFilterConditions(context.Background(), "client", caller, next)
	got := strings.Join(append(scopeSQL, groupFilterClauses(caller.Logic, callerSQL)...), " AND ")
	want := "`workspace_id` = ? AND (LOWER(`name`) = ? OR LOWER(`status`) = ?)"
	if got != want || len(scopeArgs) != 1 || scopeArgs[0] != "tenant-1" || len(callerArgs) != 2 {
		t.Fatalf("scoped OR = %q args=%v/%v, want %q", got, scopeArgs, callerArgs, want)
	}
}
