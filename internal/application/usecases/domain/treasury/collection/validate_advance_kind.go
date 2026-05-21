package collection

import (
	"fmt"

	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
)

// validateAdvanceKindNotBurnDown rejects the BURN_DOWN advance kind. Plan B
// Phase 0 hard rule: BURN_DOWN is declared in the proto but disabled for v1;
// create/update setters MUST reject this value until v2 enables it.
//
// 20260518-hexagonal-strict-adherence Phase 1.C-iv: moved here from the
// postgres adapter (F4 layer-violation fix). The check is duplicated on the
// disbursement side because each entity owns its own validation surface; the
// previously-shared treasury.ValidateAdvanceKindNotBurnDown helper is deleted.
func validateAdvanceKindNotBurnDown(kind advancekindpb.AdvanceKind) error {
	if kind == advancekindpb.AdvanceKind_ADVANCE_KIND_BURN_DOWN {
		return fmt.Errorf("advance_kind BURN_DOWN is reserved for v2; not enabled in v1")
	}
	return nil
}
