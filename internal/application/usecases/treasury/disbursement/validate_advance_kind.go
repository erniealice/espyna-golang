package disbursement

import (
	"fmt"

	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
)

// validateAdvanceKindNotBurnDown rejects the BURN_DOWN advance kind. See the
// matching collection.validateAdvanceKindNotBurnDown for the rationale.
func validateAdvanceKindNotBurnDown(kind advancekindpb.AdvanceKind) error {
	if kind == advancekindpb.AdvanceKind_ADVANCE_KIND_BURN_DOWN {
		return fmt.Errorf("advance_kind BURN_DOWN is reserved for v2; not enabled in v1")
	}
	return nil
}
