//go:build postgresql

package core

import (
	"strings"
	"testing"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

func TestKeysetToken_ValidationAndFallback(t *testing.T) {
	const id = "12345678-1234-1234-1234-123456789abc"
	for _, direction := range []string{"next", "prev"} {
		raw := encodeKeysetToken(3, direction, id)
		got, ok := decodeKeysetToken(raw, 20)
		if !ok || got.page != 3 || got.direction != direction || got.id != id {
			t.Fatalf("decodeKeysetToken(%q) = (%+v,%v)", raw, got, ok)
		}
	}
	for _, raw := range []string{
		"k1:0:next:" + id, "k1:-1:prev:" + id, "k1:100000:next:" + id,
		"k1:2:sideways:" + id, "k1:2:next:foreign-id", "k1:x:prev:" + id,
		"offset:-1", "offset:x", strings.Repeat("x", maxQueryCursorRunes+1),
	} {
		if _, ok := decodeKeysetToken(raw, 20); ok {
			t.Fatalf("accepted invalid keyset %q", raw)
		}
		limit, offset, cursor, err := listPaginationBounds(&commonpb.PaginationRequest{
			Limit: 20, Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: raw}},
		})
		if err != nil || limit != 20 || offset != 0 || !cursor {
			t.Errorf("invalid %q = (%d,%d,%t,%v), want page-one cursor", raw, limit, offset, cursor, err)
		}
	}
	limit, offset, cursor, err := listPaginationBounds(&commonpb.PaginationRequest{
		Limit: 20, Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: "offset:40"}},
	})
	if err != nil || limit != 20 || offset != 40 || !cursor {
		t.Fatalf("legacy offset = (%d,%d,%t,%v)", limit, offset, cursor, err)
	}
}

func TestKeysetWhere_NullsAndMixedDirections(t *testing.T) {
	keys := []listSortKey{{"parent_job_id", false, true}, {"date_created", true, false}, {"id", false, false}}
	where, args := keysetWhere(keys, []any{nil, int64(123), "12345678-1234-1234-1234-123456789abc"}, 5, false)
	for _, part := range []string{`"parent_job_id" IS NOT NULL`, `"date_created" < $6`, `"date_created" IS NULL`, `"id" > $7`, "IS NOT DISTINCT FROM $5"} {
		if !strings.Contains(where, part) {
			t.Errorf("forward predicate %q lacks %q", where, part)
		}
	}
	if len(args) != 3 || args[0] != nil {
		t.Fatalf("forward args = %v", args)
	}
	back, _ := keysetWhere(keys, []any{nil, int64(123), "12345678-1234-1234-1234-123456789abc"}, 5, true)
	if strings.Contains(back, `"parent_job_id" IS NOT NULL`) || !strings.Contains(back, `"date_created" > $6`) {
		t.Fatalf("reverse predicate = %q", back)
	}
}
