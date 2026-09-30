//go:build postgresql

package expenditure

import (
	"context"
	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core/scopetest"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	expenditurelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
)

// F7 (pre-existing bug B): on a schema carrying BOTH line_amount and the descriptor-aligned
// total_price column, the generic row decoded to two spellings of one proto field and every
// list/read failed ("duplicate field totalPrice"). Real adapter, real database, rolled back.
func TestExpenditureLineItemDecodesWhenBothAmountColumnsExist(t *testing.T) {
	h := scopetest.New(t, "expenditure_line_item")
	repo := NewPostgresExpenditureLineItemRepository(h.Ops, "expenditure_line_item")
	const exp = "f7li-exp"
	h.Run(t, func(base, ctxA, _, _ context.Context) {
		exec := postgresCore.TxExecutor(base, h.Ops)
		// Row written by Create (line_amount only; total_price NULL) ...
		if _, err := repo.CreateExpenditureLineItem(ctxA, &expenditurelineitempb.CreateExpenditureLineItemRequest{Data: &expenditurelineitempb.ExpenditureLineItem{
			Id: "f7li-1", Active: true, ExpenditureId: exp, Description: "Power", Quantity: 2, UnitPrice: 500, TotalPrice: 1000}}); err != nil {
			t.Fatalf("create: %v", err)
		}
		// ... and a row that also carries the descriptor column (as ETL/other writers do).
		if _, err := exec.ExecContext(base, `INSERT INTO expenditure_line_item (id, expenditure_id, description, quantity, unit_price, line_amount, total_price) VALUES ('f7li-2', $1, 'Water', 1, 300, 300, 300)`, exp); err != nil {
			t.Fatalf("seed second row: %v", err)
		}
		list, err := repo.ListExpenditureLineItems(ctxA, &expenditurelineitempb.ListExpenditureLineItemsRequest{ExpenditureId: scopetest.Str(exp)})
		if err != nil || len(list.Data) != 2 {
			t.Fatalf("list must decode both rows, got err=%v rows=%v", err, list)
		}
		want := map[string]int64{"f7li-1": 1000, "f7li-2": 300}
		for _, it := range list.Data {
			if it.GetTotalPrice() != want[it.GetId()] {
				t.Errorf("row %s total_price=%d, want %d", it.GetId(), it.GetTotalPrice(), want[it.GetId()])
			}
		}
		for id, total := range want {
			got, err := repo.ReadExpenditureLineItem(ctxA, &expenditurelineitempb.ReadExpenditureLineItemRequest{Data: &expenditurelineitempb.ExpenditureLineItem{Id: id}})
			if err != nil || len(got.Data) != 1 || got.Data[0].GetTotalPrice() != total {
				t.Errorf("read %s: err=%v got=%v want total %d", id, err, got, total)
			}
		}
	})
}

// V1: ListExpenditureLineItems forwards req.Pagination (and applies expenditure_id before paging):
// 3 rows, limit 2 -> page 1 has 2, page 2 has the remaining 1, no overlap. Rolled-back transaction.
func TestListExpenditureLineItemsHonoursPagination(t *testing.T) {
	h := scopetest.New(t, "expenditure_line_item")
	repo := NewPostgresExpenditureLineItemRepository(h.Ops, "expenditure_line_item")
	const exp = "v1pg-exp"
	h.Run(t, func(base, ctxA, _, _ context.Context) {
		for _, id := range []string{"v1pg-1", "v1pg-2", "v1pg-3"} {
			if _, err := repo.CreateExpenditureLineItem(ctxA, &expenditurelineitempb.CreateExpenditureLineItemRequest{Data: &expenditurelineitempb.ExpenditureLineItem{
				Id: id, Active: true, ExpenditureId: exp, Description: id, Quantity: 1, UnitPrice: 100, TotalPrice: 100}}); err != nil {
				t.Fatalf("create %s: %v", id, err)
			}
		}
		page := func(n int32) []string {
			resp, err := repo.ListExpenditureLineItems(ctxA, &expenditurelineitempb.ListExpenditureLineItemsRequest{
				ExpenditureId: scopetest.Str(exp),
				Sort:          &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "id", Direction: commonpb.SortDirection_ASC}}},
				Pagination:    &commonpb.PaginationRequest{Limit: 2, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: n}}},
			})
			if err != nil {
				t.Fatalf("page %d: %v", n, err)
			}
			var ids []string
			for _, it := range resp.Data {
				ids = append(ids, it.GetId())
			}
			return ids
		}
		p1, p2 := page(1), page(2)
		if len(p1) != 2 || len(p2) != 1 || p2[0] != "v1pg-3" || p1[0] != "v1pg-1" || p1[1] != "v1pg-2" {
			t.Fatalf("page1=%v page2=%v, want [v1pg-1 v1pg-2] and [v1pg-3]", p1, p2)
		}
	})
}
