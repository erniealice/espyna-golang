//go:build postgresql

package operation

import (
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
)

func TestScopedPageSort_ValidatedOrderAndNulls(t *testing.T) {
	keys, err := scopedPageSort(`ORDER BY "e"."date_created" DESC, "id" ASC`)
	if err != nil || len(keys) != 2 || keys[0].Column != "date_created" || !keys[0].Desc || !keys[0].NullsFirst || keys[1].Column != "id" {
		t.Fatalf("sort keys = %+v, %v", keys, err)
	}
	if _, err := scopedPageSort(`ORDER BY random() ASC`); err == nil {
		t.Fatal("expression sort accepted")
	}
}

func TestScopedPageMetadata_CursorBoundaries(t *testing.T) {
	first := "00000000-0000-0000-0000-000000000001"
	last := "00000000-0000-0000-0000-000000000002"
	page := scopedPageMetadata(postgresCore.Page{Limit: 2, Number: 2}, 5, first, last)
	if page.GetNextCursor() != "k1:3:next:"+last || page.GetPrevCursor() != "k1:1:prev:"+first || page.GetTotalItems() != 5 {
		t.Fatalf("page metadata = %+v", page)
	}
}
