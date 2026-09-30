package allocation_batch

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	agreementterm "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/agreement_line_term"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	allocationbatchpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_batch"
	allocationsharepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_share"
	costsourcecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
	agreementlinetermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/agreement_line_term"
)

// Allocation write behaviour (20260927-usage-and-pass-through-charges, build-spec §6.3): shared
// helpers of create_allocation_batch.go, update_allocation_batch_shares.go and
// publish_allocation_batch.go. Every write use case takes and returns esqyma messages (C1).

type writeCore struct {
	r Repositories
	s Services
}

func (c *writeCore) refuse(ctx context.Context, code string, detail ...string) error {
	return refuse(ctx, c.s.Translator, code, detail...)
}

func (c *writeCore) gate(ctx context.Context, action string) error {
	// Strict (C3): a would-be deny stays a deny even in shadow mode (fail closed on writes).
	return c.s.ActionGatekeeper.CheckStrict(ctx, &actiongate.CheckActionRequest{Entity: entityid.AllocationBatch, Action: action})
}

func (c *writeCore) inTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	if c.s.Transactor == nil || !c.s.Transactor.SupportsTransactions() {
		return c.refuse(ctx, codeTransactionRequired)
	}
	return c.s.Transactor.ExecuteInTransaction(ctx, fn)
}

func (c *writeCore) ready() error {
	if c.r.AllocationBatch == nil || c.r.AllocationShare == nil || c.r.CostSourceComponent == nil {
		return fmt.Errorf("allocation_batch: repositories unavailable")
	}
	if c.s.IDGenerator == nil {
		return fmt.Errorf("allocation_batch: id generator unavailable")
	}
	return nil
}

func strp(s string) *string { return &s }
func i64p(i int64) *int64   { return &i }

func stamp() (int64, string) {
	now := time.Now()
	return now.UnixMilli(), now.Format(time.RFC3339)
}

func strFilter(field, value string) *commonpb.FilterRequest {
	return &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
		Field: field,
		FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
			Value: value, Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true,
		}},
	}}}
}

// lockErr classifies a locker error (C9): the adapter's ErrLockedRowNotFound is a not_found
// refusal; anything else is an infrastructure error, logged with operation + ids and returned
// wrapped, never reported as not_found.
func (c *writeCore) lockErr(ctx context.Context, op, id string, err error) error {
	if errors.Is(err, domainports.ErrLockedRowNotFound) {
		return c.refuse(ctx, codeNotFound)
	}
	log.Printf("allocation_batch: %s id=%s: %v", op, id, err)
	return fmt.Errorf("allocation_batch: %s: %w", op, err)
}

// lockComponent takes the component FOR UPDATE through the adapter's locker. There is no
// unlocked-read fallback (C4): a repository without the locker capability fails closed.
func (c *writeCore) lockComponent(ctx context.Context, id string) (*costsourcecomponentpb.CostSourceComponent, error) {
	if id == "" {
		return nil, c.refuse(ctx, codeValidation, "cost source component id is required")
	}
	l, ok := c.r.CostSourceComponent.(domainports.CostSourceComponentLocker)
	if !ok {
		return nil, c.refuse(ctx, codeLockUnavailable)
	}
	v, err := l.LockCostSourceComponentForUpdate(ctx, id)
	if err != nil {
		return nil, c.lockErr(ctx, "lock component", id, err)
	}
	if v == nil {
		return nil, c.refuse(ctx, codeNotFound)
	}
	return v, nil
}

// lockBatch takes one batch FOR UPDATE; fail-closed without the locker (C4).
func (c *writeCore) lockBatch(ctx context.Context, id string) (*allocationbatchpb.AllocationBatch, error) {
	l, ok := c.r.AllocationBatch.(domainports.AllocationBatchLocker)
	if !ok {
		return nil, c.refuse(ctx, codeLockUnavailable)
	}
	v, err := l.LockAllocationBatchForUpdate(ctx, id)
	if err != nil {
		return nil, c.lockErr(ctx, "lock batch", id, err)
	}
	if v == nil {
		return nil, c.refuse(ctx, codeNotFound)
	}
	return v, nil
}

// peekBatch reads a batch WITHOUT locking, only to learn its component (lock order: component
// first, then batches). It lists by id so a missing/foreign batch is an empty result (not_found)
// while a repository failure is an error, never reported as not_found (C9).
func (c *writeCore) peekBatch(ctx context.Context, id string) (*allocationbatchpb.AllocationBatch, error) {
	if strings.TrimSpace(id) == "" {
		return nil, c.refuse(ctx, codeValidation, "allocation batch id is required")
	}
	resp, err := c.r.AllocationBatch.ListAllocationBatches(ctx, &allocationbatchpb.ListAllocationBatchesRequest{Filters: strFilter("id", id)})
	if err != nil {
		log.Printf("allocation_batch: peek batch id=%s: %v", id, err)
		return nil, fmt.Errorf("allocation_batch: read batch: %w", err)
	}
	for _, b := range resp.GetData() {
		if b.GetId() == id { // never trust an adapter that ignored the filter
			return b, nil
		}
	}
	return nil, c.refuse(ctx, codeNotFound)
}

// batchesOfComponent locks every batch of the component (revision ASC) through the locker (C4).
func (c *writeCore) batchesOfComponent(ctx context.Context, componentID string) ([]*allocationbatchpb.AllocationBatch, error) {
	l, ok := c.r.AllocationBatch.(domainports.AllocationBatchLocker)
	if !ok {
		return nil, c.refuse(ctx, codeLockUnavailable)
	}
	rows, err := l.LockAllocationBatchesByComponent(ctx, componentID)
	if err != nil {
		return nil, c.lockErr(ctx, "lock batches", componentID, err)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].GetRevision() < rows[j].GetRevision() })
	return rows, nil
}

func (c *writeCore) sharesOfBatch(ctx context.Context, batchID string) ([]*allocationsharepb.AllocationShare, error) {
	resp, err := c.r.AllocationShare.ListAllocationShares(ctx, &allocationsharepb.ListAllocationSharesRequest{Filters: strFilter("allocation_batch_id", batchID)})
	if err != nil {
		log.Printf("allocation_batch: list shares batch=%s: %v", batchID, err)
		return nil, fmt.Errorf("allocation_batch: list shares: %w", err)
	}
	var out []*allocationsharepb.AllocationShare
	for _, s := range resp.GetData() {
		if s.GetAllocationBatchId() == batchID {
			out = append(out, s)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].GetSequenceOrder() < out[j].GetSequenceOrder() })
	return out, nil
}

// requireUnclaimed refuses when the component's source is already claimed (both claim orders).
func (c *writeCore) requireUnclaimed(ctx context.Context, comp *costsourcecomponentpb.CostSourceComponent) error {
	if comp.ClaimKind == nil {
		return nil
	}
	switch comp.GetClaimKind() {
	case costsourcecomponentpb.SourceClaimKind_SOURCE_CLAIM_KIND_RECOGNITION:
		return c.refuse(ctx, codeSourceClaimedByRecognition)
	default:
		return c.refuse(ctx, codeSourceClaimedByAllocation)
	}
}

func (c *writeCore) requireDraft(ctx context.Context, b *allocationbatchpb.AllocationBatch) error {
	switch b.GetStatus() {
	case allocationbatchpb.AllocationBatchStatus_ALLOCATION_BATCH_STATUS_DRAFT:
		return nil
	case allocationbatchpb.AllocationBatchStatus_ALLOCATION_BATCH_STATUS_PUBLISHED:
		return c.refuse(ctx, codeAlreadyPublished)
	default:
		return c.refuse(ctx, codeNotDraft)
	}
}

// validateShares checks the draft rows (esqyma AllocationShare messages: share_kind, subscription,
// client, basis weights) and returns the exact amounts (largest remainder). A zero weight is a
// legal share; its amount is 0 and it creates no charge.
func (c *writeCore) validateShares(ctx context.Context, total int64, in []*allocationsharepb.AllocationShare) (amounts []int64, err error) {
	if len(in) == 0 {
		return nil, c.refuse(ctx, codeValidation, "at least one share is required")
	}
	denominator := in[0].GetBasisDenominator()
	if denominator <= 0 {
		return nil, c.refuse(ctx, codeDenominatorInvalid)
	}
	seenSub := map[string]bool{}
	split := make([]SplitInput, len(in))
	for i, s := range in {
		if s.GetBasisDenominator() != denominator {
			return nil, c.refuse(ctx, codeSharesTotalMismatch, "all shares must carry the same total weight")
		}
		if s.GetBasisNumerator() < 0 {
			return nil, c.refuse(ctx, codeValidation, "weight must not be negative")
		}
		switch s.GetShareKind() {
		case allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_RECOVERABLE:
			if s.GetSubscriptionId() == "" || s.GetClientId() == "" {
				return nil, c.refuse(ctx, codeValidation, "a recoverable share needs a subscription and a client")
			}
			if seenSub[s.GetSubscriptionId()] {
				return nil, c.refuse(ctx, codeValidation, "a subscription appears in more than one recoverable share")
			}
			seenSub[s.GetSubscriptionId()] = true
		case allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_OWN_USE,
			allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_VACANCY,
			allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_COMMON_LOSS:
			if s.GetSubscriptionId() != "" || s.GetClientId() != "" {
				return nil, c.refuse(ctx, codeValidation, "only a recoverable share may reference a subscription or client")
			}
		default:
			return nil, c.refuse(ctx, codeValidation, "share_kind is required")
		}
		split[i] = SplitInput{SequenceOrder: int32(i + 1), Numerator: s.GetBasisNumerator()}
	}
	amounts, ok := LargestRemainderSplit(total, denominator, split)
	if !ok {
		return nil, c.refuse(ctx, codeSharesTotalMismatch, "weights must add up to the total weight")
	}
	return amounts, nil
}

// verifyRecoverableTerms proves (workspace-scoped) that every recoverable share's subscription has
// a charge term for that client; a foreign or unknown subscription finds none and is refused.
func (c *writeCore) verifyRecoverableTerms(ctx context.Context, in []*allocationsharepb.AllocationShare) error {
	for _, s := range in {
		if s.GetShareKind() != allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_RECOVERABLE {
			continue
		}
		terms, err := c.listTerms(ctx, s.GetSubscriptionId())
		if err != nil {
			return err
		}
		found := false
		for _, t := range terms {
			if t.GetActive() && t.GetClientId() == s.GetClientId() {
				found = true
				break
			}
		}
		if !found {
			return c.refuse(ctx, codeAgreementTermMissing)
		}
	}
	return nil
}

func (c *writeCore) listTerms(ctx context.Context, subscriptionID string) ([]*agreementlinetermpb.AgreementLineTerm, error) {
	if c.r.AgreementLineTerm == nil {
		return nil, c.refuse(ctx, codeValidation, "agreement terms repository unavailable (fail closed)")
	}
	terms, err := agreementterm.ListBySubscription(ctx, c.r.AgreementLineTerm, subscriptionID)
	if err != nil {
		log.Printf("allocation_batch: list agreement terms subscription=%s: %v", subscriptionID, err)
		return nil, fmt.Errorf("allocation_batch: list agreement terms: %w", err)
	}
	return terms, nil
}

// writeShares persists draft share rows (sequence_order = input order, 1-based). Zero-valued
// weights and amounts are stored as 0 through the column DEFAULT 0 (C7).
func (c *writeCore) writeShares(ctx context.Context, batchID string, in []*allocationsharepb.AllocationShare, amounts []int64) ([]*allocationsharepb.AllocationShare, error) {
	ms, ts := stamp()
	out := make([]*allocationsharepb.AllocationShare, 0, len(in))
	for i, s := range in {
		row := &allocationsharepb.AllocationShare{
			Id:                 c.s.IDGenerator.GenerateID(),
			AllocationBatchId:  batchID,
			ShareKind:          s.GetShareKind(),
			BasisNumerator:     s.GetBasisNumerator(),
			BasisDenominator:   s.GetBasisDenominator(),
			Amount:             amounts[i],
			SequenceOrder:      int32(i + 1),
			Active:             true,
			DateCreated:        i64p(ms),
			DateCreatedString:  strp(ts),
			DateModified:       i64p(ms),
			DateModifiedString: strp(ts),
		}
		if s.GetSubscriptionId() != "" {
			row.SubscriptionId = strp(s.GetSubscriptionId())
		}
		if s.GetClientId() != "" {
			row.ClientId = strp(s.GetClientId())
		}
		resp, err := c.r.AllocationShare.CreateAllocationShare(ctx, &allocationsharepb.CreateAllocationShareRequest{Data: row})
		if err != nil {
			return nil, fmt.Errorf("allocation_batch: create share: %w", err)
		}
		if d := resp.GetData(); len(d) > 0 && d[0] != nil {
			row = d[0]
		}
		out = append(out, row)
	}
	return out, nil
}
