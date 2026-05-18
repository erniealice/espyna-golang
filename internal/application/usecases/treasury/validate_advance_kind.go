package treasury

import (
	"fmt"

	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
)

// ValidateAdvanceKindNotBurnDown returns an error if kind is BURN_DOWN.
// Plan B Phase 0 hard rule: BURN_DOWN enum value is declared but disabled
// for v1; create/update setters MUST reject this value until v2 enables.
//
// Wired into the Postgres adapter Create/Update paths for both
// TreasuryCollection and TreasuryDisbursement (see contrib/postgres/internal/
// adapter/treasury/collection.go + disbursement.go).
func ValidateAdvanceKindNotBurnDown(kind advancekindpb.AdvanceKind) error {
	if kind == advancekindpb.AdvanceKind_ADVANCE_KIND_BURN_DOWN {
		return fmt.Errorf("advance_kind BURN_DOWN is reserved for v2; not enabled in v1")
	}
	return nil
}
