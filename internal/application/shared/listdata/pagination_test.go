package listdata

import (
	"testing"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

func TestValidatePaginationRequest_NilRequest(t *testing.T) {
	t.Parallel()

	p := NewPaginationUtils()
	result := p.ValidatePaginationRequest(nil)

	if result == nil {
		t.Fatal("expected non-nil result for nil input")
	}
	if result.Limit != 20 {
		t.Errorf("default limit = %d, want 20", result.Limit)
	}
	if result.Method == nil {
		t.Fatal("expected non-nil pagination method")
	}
	offset, ok := result.Method.(*commonpb.PaginationRequest_Offset)
	if !ok {
		t.Fatalf("expected offset pagination, got %T", result.Method)
	}
	if offset.Offset.Page != 1 {
		t.Errorf("default page = %d, want 1", offset.Offset.Page)
	}
}

func TestValidatePaginationRequest_EnforcesLimits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		inputLim  int32
		wantLimit int32
	}{
		{name: "negative_limit", inputLim: -1, wantLimit: 20},
		{name: "zero_limit", inputLim: 0, wantLimit: 20},
		{name: "valid_limit", inputLim: 50, wantLimit: 50},
		{name: "max_limit_exceeded", inputLim: 200, wantLimit: 100},
		{name: "exact_max", inputLim: 100, wantLimit: 100},
	}

	p := NewPaginationUtils()
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := &commonpb.PaginationRequest{
				Limit: tc.inputLim,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			}
			result := p.ValidatePaginationRequest(req)
			if result.Limit != tc.wantLimit {
				t.Errorf("limit = %d, want %d", result.Limit, tc.wantLimit)
			}
		})
	}
}

func TestValidatePaginationRequest_InvalidPage(t *testing.T) {
	t.Parallel()

	p := NewPaginationUtils()
	req := &commonpb.PaginationRequest{
		Limit: 20,
		Method: &commonpb.PaginationRequest_Offset{
			Offset: &commonpb.OffsetPagination{Page: 0},
		},
	}
	result := p.ValidatePaginationRequest(req)
	offset := result.Method.(*commonpb.PaginationRequest_Offset)
	if offset.Offset.Page != 1 {
		t.Errorf("page = %d, want 1", offset.Offset.Page)
	}
}

func TestValidatePaginationRequest_NilMethod(t *testing.T) {
	t.Parallel()

	p := NewPaginationUtils()
	req := &commonpb.PaginationRequest{
		Limit:  20,
		Method: nil,
	}
	result := p.ValidatePaginationRequest(req)
	if result.Method == nil {
		t.Fatal("expected non-nil method after validation")
	}
}

func TestCalculateOffsetAndLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		page       int32
		limit      int32
		wantOffset int32
		wantLimit  int32
	}{
		{name: "page_1", page: 1, limit: 20, wantOffset: 0, wantLimit: 20},
		{name: "page_2", page: 2, limit: 20, wantOffset: 20, wantLimit: 20},
		{name: "page_3_limit_10", page: 3, limit: 10, wantOffset: 20, wantLimit: 10},
		{name: "page_5_limit_50", page: 5, limit: 50, wantOffset: 200, wantLimit: 50},
	}

	p := NewPaginationUtils()
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := &commonpb.PaginationRequest{
				Limit: tc.limit,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: tc.page},
				},
			}
			offset, limit := p.CalculateOffsetAndLimit(req)
			if offset != tc.wantOffset {
				t.Errorf("offset = %d, want %d", offset, tc.wantOffset)
			}
			if limit != tc.wantLimit {
				t.Errorf("limit = %d, want %d", limit, tc.wantLimit)
			}
		})
	}
}

func TestCalculateOffsetAndLimit_NilMethod(t *testing.T) {
	t.Parallel()

	p := NewPaginationUtils()
	req := &commonpb.PaginationRequest{
		Limit:  20,
		Method: nil,
	}
	offset, limit := p.CalculateOffsetAndLimit(req)
	if offset != 0 {
		t.Errorf("offset = %d, want 0", offset)
	}
	if limit != 20 {
		t.Errorf("limit = %d, want 20", limit)
	}
}

func TestCreatePaginationResponse_OffsetPagination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		page        int32
		limit       int32
		totalItems  int32
		hasNext     bool
		wantPages   int32
		wantHasPrev bool
		wantHasNext bool
	}{
		{
			name: "first_page_with_more", page: 1, limit: 10, totalItems: 25, hasNext: true,
			wantPages: 3, wantHasPrev: false, wantHasNext: true,
		},
		{
			name: "middle_page", page: 2, limit: 10, totalItems: 25, hasNext: true,
			wantPages: 3, wantHasPrev: true, wantHasNext: true,
		},
		{
			name: "last_page", page: 3, limit: 10, totalItems: 25, hasNext: false,
			wantPages: 3, wantHasPrev: true, wantHasNext: false,
		},
		{
			name: "single_page", page: 1, limit: 20, totalItems: 5, hasNext: false,
			wantPages: 1, wantHasPrev: false, wantHasNext: false,
		},
		{
			name: "empty_results", page: 1, limit: 20, totalItems: 0, hasNext: false,
			wantPages: 1, wantHasPrev: false, wantHasNext: false,
		},
	}

	p := NewPaginationUtils()
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := &commonpb.PaginationRequest{
				Limit: tc.limit,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: tc.page},
				},
			}
			resp := p.CreatePaginationResponse(req, tc.totalItems, tc.hasNext)

			if resp.TotalItems != tc.totalItems {
				t.Errorf("TotalItems = %d, want %d", resp.TotalItems, tc.totalItems)
			}
			if resp.TotalPages != nil && *resp.TotalPages != tc.wantPages {
				t.Errorf("TotalPages = %d, want %d", *resp.TotalPages, tc.wantPages)
			}
			if resp.HasPrev != tc.wantHasPrev {
				t.Errorf("HasPrev = %v, want %v", resp.HasPrev, tc.wantHasPrev)
			}
			if resp.HasNext != tc.wantHasNext {
				t.Errorf("HasNext = %v, want %v", resp.HasNext, tc.wantHasNext)
			}
		})
	}
}

func TestDecodeCursor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		token string
		want  int32
	}{
		{name: "empty_token", token: "", want: 0},
		{name: "valid_offset", token: "offset:20", want: 20},
		{name: "zero_offset", token: "offset:0", want: 0},
		{name: "invalid_format", token: "invalid", want: 0},
		{name: "wrong_prefix", token: "cursor:20", want: 0},
		{name: "non_numeric", token: "offset:abc", want: 0},
	}

	p := NewPaginationUtils()
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := p.decodeCursor(tc.token)
			if got != tc.want {
				t.Errorf("decodeCursor(%q) = %d, want %d", tc.token, got, tc.want)
			}
		})
	}
}
