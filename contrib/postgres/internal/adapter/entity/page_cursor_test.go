//go:build postgresql

package entity

import (
	"strings"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

func TestEntityPageCursor_TokenAndScopedSQLShape(t *testing.T) {
	const uuid1 = "0199a2a2-0000-7000-8000-000000000001"
	page := int32(2)
	total := int32(4)
	pagination := &commonpb.PaginationResponse{CurrentPage: &page, TotalPages: &total, HasNext: true, HasPrev: true}
	setPageCursors(pagination, uuid1, uuid1, 7)
	if !strings.HasPrefix(pagination.GetNextCursor(), "k1:3:next:") || !strings.HasPrefix(pagination.GetPrevCursor(), "k1:1:prev:") {
		t.Fatalf("UUID page cursors = %q / %q", pagination.GetNextCursor(), pagination.GetPrevCursor())
	}
	request, err := postgresCore.BoundedPageRequest(cursorToken(7, pagination.GetNextCursor()), 50)
	if err != nil || request.Mode != postgresCore.PageModeKeyset || request.BoundaryID != uuid1 {
		t.Fatalf("decoded keyset = %#v, %v", request, err)
	}
	setPageCursors(pagination, "legacy-1", "legacy-2", 7)
	if pagination.GetNextCursor() != "offset:14" || pagination.GetPrevCursor() != "offset:0" {
		t.Fatalf("legacy page cursors = %q / %q", pagination.GetNextCursor(), pagination.GetPrevCursor())
	}
	request, err = postgresCore.BoundedPageRequest(cursorToken(7, pagination.GetNextCursor()), 50)
	if err != nil || request.Mode != postgresCore.PageModeOffset || request.Number != 3 {
		t.Fatalf("decoded legacy offset = %#v, %v", request, err)
	}
	set := postgresCore.ScopedPageSet{
		SQL:  "SELECT id, name FROM client WHERE workspace_id = $1 AND name ILIKE $2",
		Args: []any{"workspace", "%a%"}, Sort: pageSortKeys(nil, "name", false),
	}
	q, err := postgresCore.ResolveScopedPage(nil, nil, set, request)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"workspace_id = $1", "name ILIKE $2", "ORDER BY", "LIMIT $3 OFFSET $4"} {
		if !strings.Contains(q.PageSQL, fragment) {
			t.Fatalf("page SQL missing %q: %s", fragment, q.PageSQL)
		}
	}
	if !strings.Contains(q.CountSQL, "workspace_id = $1 AND name ILIKE $2") {
		t.Fatalf("count SQL lost scope: %s", q.CountSQL)
	}
}
