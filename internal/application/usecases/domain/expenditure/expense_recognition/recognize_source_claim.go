package expenserecognition

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	costsourcecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
)

// S1 shared source claim (20260927-usage-and-pass-through-charges, build-spec §6.3): an expenditure's
// cost_source_component rows are claimed by RECOGNITION here or by ALLOCATION in
// allocation_batch.PublishAllocationBatch, exactly one wins under the component row locks.

// Named refusals (C2): message translated from expense_recognition.errors.<code>, ErrorCode() = <code>.
// Consumers read the code via interface{ ErrorCode() string }; nothing here is an exported sentinel.
const (
	codeSourceClaimedByAllocation = "source_claimed_by_allocation"
	codeTransactionRequired       = "transaction_required"
	codeLockUnavailable           = "lock_unavailable"
)

var claimFallbacks = map[string]string{
	codeSourceClaimedByAllocation: "This expenditure's cost is already allocated to customers and cannot be recognized as an expense.",
	codeTransactionRequired:       "Recognizing an expenditure with recoverable cost components needs a database transaction.",
	codeLockUnavailable:           "Recognizing an expenditure with recoverable cost components needs row locking, which the storage provider does not support.",
}

func (uc *RecognizeFromExpenditureUseCase) refuseClaim(ctx context.Context, code string) error {
	return usecaseerr.New("", code, contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "expense_recognition.errors."+code, claimFallbacks[code]))
}

// expenditureHasSourceComponents is the plain existence read that selects the claim path. It is
// workspace-scoped (the adapter's predicate) and limited to one row; a failure is logged with the
// operation and id and refuses the recognition (fail closed) rather than guessing "no components".
func (uc *RecognizeFromExpenditureUseCase) expenditureHasSourceComponents(ctx context.Context, expenditureID string) (bool, error) {
	resp, err := uc.repositories.CostSourceComponent.ListCostSourceComponents(ctx, &costsourcecomponentpb.ListCostSourceComponentsRequest{
		Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
			Field: "expenditure_id",
			FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
				Value: expenditureID, Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true}},
		}}},
		Pagination: &commonpb.PaginationRequest{Limit: 1},
	})
	if err != nil {
		log.Printf("expense_recognition: list cost_source_component expenditure=%s: %v", expenditureID, err)
		return false, fmt.Errorf("expense_recognition: read cost source components: %w", err)
	}
	for _, c := range resp.GetData() {
		if c.GetExpenditureId() == expenditureID { // never trust an adapter that ignored the filter
			return true, nil
		}
	}
	return false, nil
}

// lockSourceComponents locks the expenditure's components FOR UPDATE (id ascending; the postgres
// adapter's locker, through the ambient tx executor) and refuses when any is claimed by
// ALLOCATION. A repository without the locker capability fails closed (C4): there is no
// unlocked-list fallback.
func (uc *RecognizeFromExpenditureUseCase) lockSourceComponents(ctx context.Context, expenditureID string) ([]*costsourcecomponentpb.CostSourceComponent, error) {
	l, ok := uc.repositories.CostSourceComponent.(domainports.CostSourceComponentLocker)
	if !ok {
		return nil, uc.refuseClaim(ctx, codeLockUnavailable)
	}
	comps, err := l.LockCostSourceComponentsByExpenditure(ctx, expenditureID)
	if err != nil {
		log.Printf("expense_recognition: lock cost_source_component expenditure=%s: %v", expenditureID, err)
		return nil, fmt.Errorf("lock cost source components: %w", err)
	}
	sort.SliceStable(comps, func(i, j int) bool { return comps[i].GetId() < comps[j].GetId() })
	for _, c := range comps {
		if c.ClaimKind != nil && c.GetClaimKind() == costsourcecomponentpb.SourceClaimKind_SOURCE_CLAIM_KIND_ALLOCATION {
			return nil, uc.refuseClaim(ctx, codeSourceClaimedByAllocation)
		}
	}
	return comps, nil
}

// markRecognitionClaim stamps the RECOGNITION claim on every still-unclaimed component. A
// component already claimed by an earlier recognition (multi-period amortisation) keeps its claim.
func (uc *RecognizeFromExpenditureUseCase) markRecognitionClaim(ctx context.Context, comps []*costsourcecomponentpb.CostSourceComponent, recognitionID string) error {
	now := time.Now()
	ms, ts := now.UnixMilli(), now.Format(time.RFC3339)
	kind := costsourcecomponentpb.SourceClaimKind_SOURCE_CLAIM_KIND_RECOGNITION
	for _, c := range comps {
		if c.ClaimKind != nil {
			continue
		}
		if _, err := uc.repositories.CostSourceComponent.UpdateCostSourceComponent(ctx, &costsourcecomponentpb.UpdateCostSourceComponentRequest{Data: &costsourcecomponentpb.CostSourceComponent{
			Id: c.GetId(), ClaimKind: &kind, ClaimRefId: &recognitionID, ClaimedAt: &ms,
			DateModified: &ms, DateModifiedString: &ts,
		}}); err != nil {
			return fmt.Errorf("claim cost source component %s: %w", c.GetId(), err)
		}
	}
	return nil
}
