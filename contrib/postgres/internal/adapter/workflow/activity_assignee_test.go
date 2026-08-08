//go:build postgresql

package workflow

import "testing"

func TestPendingActivitiesPagination(t *testing.T) {
	tests := []struct {
		name       string
		limit, off int
		wantLimit  int
		wantOffset int
		wantErr    bool
	}{
		{name: "defaults", wantLimit: 50},
		{name: "negative compatibility", limit: -1, off: -1, wantLimit: 50},
		{name: "requested", limit: 100, off: 1_000_000, wantLimit: 100, wantOffset: 1_000_000},
		{name: "limit rejected", limit: 101, wantErr: true},
		{name: "offset rejected", off: 1_000_001, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, offset, err := pendingActivitiesPagination(tt.limit, tt.off)
			if (err != nil) != tt.wantErr {
				t.Fatalf("pendingActivitiesPagination() error = %v, wantErr %t", err, tt.wantErr)
			}
			if !tt.wantErr && (limit != tt.wantLimit || offset != tt.wantOffset) {
				t.Fatalf("pendingActivitiesPagination() = (%d, %d), want (%d, %d)", limit, offset, tt.wantLimit, tt.wantOffset)
			}
		})
	}
}
