package collectionmethodgrant

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
	grantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_grant"
)

// validateAudienceModeGuardrail enforces the §E-4 audience-mode rule at grant /
// issuance time. Given a CollectionMethod TEMPLATE id and the COUNT of ACTIVE
// grants that WOULD exist after the operation under evaluation, it asserts:
//
//	OPEN          → zero grants allowed (eligibility bypasses grant lookup);
//	                a non-zero count is rejected.
//	RESTRICTED    → at least one grant required.
//	SINGLE_CLIENT → exactly one grant required.
//	(SEGMENT_SCOPED reserved v2 — treated as not-yet-supported.)
//
// It is shared by create + bulk_grant (phases.md Stage 3 sub-step 3b/3c). The
// helper resolves the method's audience_mode via the injected CollectionMethod
// repository; callers pass the already-computed prospective ACTIVE-grant count so
// the helper stays a pure policy check with a single repo read.
//
// methodRepo may be nil in mock/dev wiring; when it is, the guardrail is a no-op
// (the use case still functions, mirroring the nil-safe repo convention).
func validateAudienceModeGuardrail(
	ctx context.Context,
	translator ports.Translator,
	methodRepo collectionmethodpb.CollectionMethodDomainServiceServer,
	collectionMethodID string,
	prospectiveActiveGrantCount int,
) error {
	if methodRepo == nil {
		// Nil-safe: no method repository wired (mock/dev) — skip the policy check.
		return nil
	}
	if collectionMethodID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, translator,
			"collection_method_grant.validation.collection_method_id_required",
			"[ERR-DEFAULT] Collection method ID is required"))
	}

	resp, err := methodRepo.ReadCollectionMethod(ctx, &collectionmethodpb.ReadCollectionMethodRequest{
		Data: &collectionmethodpb.CollectionMethod{Id: collectionMethodID},
	})
	if err != nil {
		return fmt.Errorf("%s: %w", contextutil.GetTranslatedMessageWithContext(ctx, translator,
			"collection_method_grant.errors.method_lookup_failed",
			"[ERR-DEFAULT] Failed to resolve the collection method for audience-mode validation"), err)
	}
	if resp == nil || len(resp.GetData()) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, translator,
			"collection_method_grant.validation.method_not_found",
			"[ERR-DEFAULT] Collection method not found for audience-mode validation"))
	}

	mode := resp.GetData()[0].GetAudienceMode()
	switch mode {
	case collectionmethodpb.CollectionMethodAudienceMode_COLLECTION_METHOD_AUDIENCE_MODE_OPEN:
		if prospectiveActiveGrantCount != 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, translator,
				"collection_method_grant.validation.open_no_grants",
				"[ERR-DEFAULT] OPEN methods allow zero audience grants"))
		}
	case collectionmethodpb.CollectionMethodAudienceMode_COLLECTION_METHOD_AUDIENCE_MODE_RESTRICTED:
		if prospectiveActiveGrantCount < 1 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, translator,
				"collection_method_grant.validation.restricted_requires_grant",
				"[ERR-DEFAULT] RESTRICTED methods require at least one audience grant"))
		}
	case collectionmethodpb.CollectionMethodAudienceMode_COLLECTION_METHOD_AUDIENCE_MODE_SINGLE_CLIENT:
		if prospectiveActiveGrantCount != 1 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, translator,
				"collection_method_grant.validation.single_client_requires_exactly_one",
				"[ERR-DEFAULT] SINGLE_CLIENT methods require exactly one audience grant"))
		}
	case collectionmethodpb.CollectionMethodAudienceMode_COLLECTION_METHOD_AUDIENCE_MODE_SEGMENT_SCOPED:
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, translator,
			"collection_method_grant.validation.segment_scoped_unsupported",
			"[ERR-DEFAULT] SEGMENT_SCOPED audience mode is reserved for v2"))
	default:
		// UNSPECIFIED — the method has no declared audience policy yet; nothing to
		// enforce. Treated as permissive (matches the additive Stage-1 default where
		// audience_mode is unset on legacy methods).
	}
	return nil
}

// countActiveGrantsForMethod returns the number of ACTIVE grants currently bound to
// the given CollectionMethod template. It lists grants filtered by
// collection_method_id and counts the ACTIVE ones in-app (robust regardless of the
// adapter's enum-filter support). grantRepo may be nil (mock/dev) → returns 0.
func countActiveGrantsForMethod(
	ctx context.Context,
	grantRepo grantpb.CollectionMethodGrantDomainServiceServer,
	collectionMethodID string,
) (int, error) {
	if grantRepo == nil {
		return 0, nil
	}
	resp, err := grantRepo.ListCollectionMethodGrants(ctx, &grantpb.ListCollectionMethodGrantsRequest{})
	if err != nil {
		return 0, err
	}
	if resp == nil {
		return 0, nil
	}
	count := 0
	for _, g := range resp.GetData() {
		if g.GetCollectionMethodId() != collectionMethodID {
			continue
		}
		if g.GetStatus() == grantpb.CollectionMethodGrantStatus_COLLECTION_METHOD_GRANT_STATUS_ACTIVE {
			count++
		}
	}
	return count, nil
}
