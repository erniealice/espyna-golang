package listdata

import (
	"errors"
	"testing"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

func TestListAllWalksPagesInIDOrderUntilAShortPage(t *testing.T) {
	var pages []int32
	got, err := ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]int, error) {
		if s.GetFields()[0].GetField() != "id" {
			t.Fatalf("sort = %v, want id", s)
		}
		pages = append(pages, p.GetOffset().GetPage())
		if p.GetOffset().GetPage() < 3 {
			return make([]int, ScanPageLimit), nil
		}
		return []int{1}, nil
	})
	if err != nil || len(got) != int(2*ScanPageLimit)+1 || len(pages) != 3 {
		t.Fatalf("got %d rows over pages %v, err %v", len(got), pages, err)
	}
}

func TestListAllFailsClosedPastThePageBound(t *testing.T) {
	_, err := ListAll(func(*commonpb.PaginationRequest, *commonpb.SortRequest) ([]int, error) {
		return make([]int, ScanPageLimit), nil
	})
	if err == nil {
		t.Fatal("an unbounded read must be refused, not truncated")
	}
}

func TestListAllReturnsTheFetchError(t *testing.T) {
	boom := errors.New("boom")
	if _, err := ListAll(func(*commonpb.PaginationRequest, *commonpb.SortRequest) ([]int, error) { return nil, boom }); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
}

func TestEqFilterIsOneCaseSensitiveEquality(t *testing.T) {
	f := EqFilter("client_id", "c-1").GetFilters()
	if len(f) != 1 || f[0].GetField() != "client_id" || f[0].GetStringFilter().GetValue() != "c-1" ||
		!f[0].GetStringFilter().GetCaseSensitive() || f[0].GetStringFilter().GetOperator() != commonpb.StringOperator_STRING_EQUALS {
		t.Fatalf("filter = %v", f)
	}
}
