package amortization

// NOTE: pnpm build must be run in packages/esqyma/ to generate the proto
// types before this package compiles. The import path
// github.com/erniealice/esqyma/pkg/schema/v1/service/amortization will not
// resolve until generation completes.

import (
	amortizeschedule "github.com/erniealice/espyna-golang/internal/application/shared/amortize_schedule"

	amortizationpb "github.com/erniealice/esqyma/pkg/schema/v1/service/amortization"
)

// protoProrationToHelper translates the proto ProrationPolicy enum to the
// pure-math helper's ProrationPolicy type.
func protoProrationToHelper(p amortizationpb.ProrationPolicy) amortizeschedule.ProrationPolicy {
	switch p {
	case amortizationpb.ProrationPolicy_PRORATION_POLICY_DAY_PRORATED:
		return amortizeschedule.ProrationPolicyDayProrated
	case amortizationpb.ProrationPolicy_PRORATION_POLICY_FULL_TRANCHE:
		return amortizeschedule.ProrationPolicyFullTranche
	case amortizationpb.ProrationPolicy_PRORATION_POLICY_NEXT_PERIOD_START:
		return amortizeschedule.ProrationPolicyNextPeriodStart
	default:
		// UNSPECIFIED normalizes to FULL_TRANCHE per Decision 13.
		return amortizeschedule.ProrationPolicyFullTranche
	}
}

// HelperProrationToProto translates the pure-math helper's ProrationPolicy
// to the proto enum. Exported for use by domain-layer callers that need to
// convert from the advance_kind proto enum to the amortization service proto
// enum via the helper as an intermediate.
func HelperProrationToProto(p amortizeschedule.ProrationPolicy) amortizationpb.ProrationPolicy {
	switch p {
	case amortizeschedule.ProrationPolicyDayProrated:
		return amortizationpb.ProrationPolicy_PRORATION_POLICY_DAY_PRORATED
	case amortizeschedule.ProrationPolicyFullTranche:
		return amortizationpb.ProrationPolicy_PRORATION_POLICY_FULL_TRANCHE
	case amortizeschedule.ProrationPolicyNextPeriodStart:
		return amortizationpb.ProrationPolicy_PRORATION_POLICY_NEXT_PERIOD_START
	default:
		return amortizationpb.ProrationPolicy_PRORATION_POLICY_UNSPECIFIED
	}
}
