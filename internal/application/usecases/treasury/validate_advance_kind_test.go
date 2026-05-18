package treasury

import (
	"strings"
	"testing"

	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
)

// TestValidateAdvanceKindNotBurnDown exercises the Plan B Phase 0 hard rule
// that BURN_DOWN is declared in the proto but disabled until v2.
func TestValidateAdvanceKindNotBurnDown(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		kind    advancekindpb.AdvanceKind
		wantErr bool
	}{
		{"unspecified ok", advancekindpb.AdvanceKind_ADVANCE_KIND_UNSPECIFIED, false},
		{"none ok", advancekindpb.AdvanceKind_ADVANCE_KIND_NONE, false},
		{"time_based ok", advancekindpb.AdvanceKind_ADVANCE_KIND_TIME_BASED, false},
		{"milestone ok", advancekindpb.AdvanceKind_ADVANCE_KIND_MILESTONE, false},
		{"unscheduled ok", advancekindpb.AdvanceKind_ADVANCE_KIND_UNSCHEDULED, false},
		{"burn_down rejected", advancekindpb.AdvanceKind_ADVANCE_KIND_BURN_DOWN, true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateAdvanceKindNotBurnDown(tc.kind)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %v, got nil", tc.kind)
				}
				if !strings.Contains(err.Error(), "BURN_DOWN") {
					t.Errorf("expected error to mention BURN_DOWN, got %q", err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error for %v, got %v", tc.kind, err)
			}
		})
	}
}
