package allocation_batch

import (
	"context"
	"fmt"
	"log"
	"time"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	agreementterm "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/agreement_line_term"
	billablecharge "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/billable_charge"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	allocationbatchpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_batch"
	allocationsharepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_share"
	costsourcecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
	chargepolicycomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_component"
	chargepolicyversionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	agreementlinetermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/agreement_line_term"
)

// PublishAllocationBatchUseCase publishes a DRAFT allocation in ONE transaction
// (allocation_batch:publish, CheckStrict): locks the component (then the batch), refuses a source
// claimed by RECOGNITION or another PUBLISHED batch, checks every recoverable share against a charge
// term that covers the whole service interval (term_boundary_crossed), splits the amount by largest
// remainder, writes the ALLOCATION claim, and creates one OPEN charge + component per recoverable
// share with a positive amount. Request: AllocationBatchId; response: Data = the published batch,
// Shares (final amounts), Charges, ReplayedCount (charges that already existed with an identical
// content hash).
type PublishAllocationBatchUseCase struct{ c *writeCore }

func (uc *PublishAllocationBatchUseCase) Execute(ctx context.Context, req *allocationbatchpb.PublishAllocationBatchRequest) (*allocationbatchpb.PublishAllocationBatchResponse, error) {
	if err := uc.c.gate(ctx, entityid.ActionPublish); err != nil {
		return nil, err
	}
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		return nil, uc.c.refuse(ctx, codeValidation, "authenticated user required")
	}
	if req == nil || req.GetAllocationBatchId() == "" {
		return nil, uc.c.refuse(ctx, codeValidation, "allocation batch id is required")
	}
	if err := uc.c.ready(); err != nil {
		return nil, err
	}
	if uc.c.r.BillableCharge == nil || uc.c.r.ChargeComponent == nil || uc.c.r.ChargePolicyVersion == nil || uc.c.r.ChargePolicyComponent == nil {
		return nil, uc.c.refuse(ctx, codeValidation, "charge repositories unavailable (fail closed)")
	}
	var out *allocationbatchpb.PublishAllocationBatchResponse
	err = uc.c.inTx(ctx, func(txCtx context.Context) error {
		peek, err := uc.c.peekBatch(txCtx, req.GetAllocationBatchId())
		if err != nil {
			return err
		}
		if peek.GetStatus() == allocationbatchpb.AllocationBatchStatus_ALLOCATION_BATCH_STATUS_PUBLISHED {
			return uc.c.refuse(txCtx, codeAlreadyPublished)
		}
		comp, err := uc.c.lockComponent(txCtx, peek.GetCostSourceComponentId())
		if err != nil {
			return err
		}
		if err := uc.c.requireUnclaimed(txCtx, comp); err != nil { // both claim orders, under the lock
			return err
		}
		all, err := uc.c.batchesOfComponent(txCtx, comp.GetId()) // locks batches revision ASC
		if err != nil {
			return err
		}
		var batch *allocationbatchpb.AllocationBatch
		for _, b := range all {
			if b.GetId() == req.GetAllocationBatchId() {
				batch = b
			}
		}
		if batch == nil {
			return uc.c.refuse(txCtx, codeNotFound)
		}
		if err := uc.c.requireDraft(txCtx, batch); err != nil {
			return err
		}
		for _, b := range all {
			if b.GetId() != batch.GetId() && b.GetStatus() == allocationbatchpb.AllocationBatchStatus_ALLOCATION_BATCH_STATUS_PUBLISHED {
				return uc.c.refuse(txCtx, codeSourceClaimedByAllocation)
			}
		}
		from, to := comp.GetServiceFrom(), comp.GetServiceTo()
		if comp.ServiceFrom == nil || comp.ServiceTo == nil || from >= to {
			return uc.c.refuse(txCtx, codeServicePeriodInvalid)
		}
		shares, err := uc.c.sharesOfBatch(txCtx, batch.GetId())
		if err != nil {
			return err
		}
		recoverable := 0
		for _, s := range shares {
			if s.GetShareKind() == allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_RECOVERABLE {
				recoverable++
			}
		}
		if recoverable == 0 {
			return uc.c.refuse(txCtx, codeNoRecoverableShare)
		}
		amounts, err := uc.c.validateShares(txCtx, comp.GetAmount(), shares)
		if err != nil {
			return err
		}

		// Pass 1 (reads only): resolve the covering term and pinned version of every recoverable
		// share, so every refusal (term_boundary_crossed, policy_component_invalid, ...) happens
		// before the first write.
		type plan struct {
			term *agreementlinetermpb.AgreementLineTerm
			ver  *versionInfo
		}
		plans := make([]*plan, len(shares))
		versions := map[string]*versionInfo{}
		for i, s := range shares {
			if s.GetShareKind() != allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_RECOVERABLE {
				continue
			}
			terms, err := uc.c.listTerms(txCtx, s.GetSubscriptionId())
			if err != nil {
				return err
			}
			term, err := agreementterm.FindCoveringTerm(terms, s.GetClientId(), from, to)
			if err != nil {
				return reissue(txCtx, uc.c.s.Translator, err)
			}
			if amounts[i] <= 0 {
				continue // a zero-weight recoverable share produces no charge
			}
			vi, ok := versions[term.GetChargePolicyVersionId()]
			if !ok {
				if vi, err = uc.c.loadVersion(txCtx, term.GetChargePolicyVersionId()); err != nil {
					return err
				}
				versions[term.GetChargePolicyVersionId()] = vi
			}
			plans[i] = &plan{term: term, ver: vi}
		}

		// Pass 2: writes.
		ms, ts := stamp()
		accountingDate := time.UnixMilli(ms).UTC().Format("2006-01-02")
		resp := &allocationbatchpb.PublishAllocationBatchResponse{Success: true}
		for i, s := range shares {
			if s.GetAmount() != amounts[i] {
				if _, err := uc.c.r.AllocationShare.UpdateAllocationShare(txCtx, &allocationsharepb.UpdateAllocationShareRequest{Data: &allocationsharepb.AllocationShare{
					Id: s.GetId(), Amount: amounts[i], DateModified: i64p(ms), DateModifiedString: strp(ts)}}); err != nil {
					return err
				}
				s.Amount = amounts[i]
			}
			pl := plans[i]
			if pl == nil {
				continue
			}
			res, err := billablecharge.CreateChargeFromAllocation(txCtx,
				billablecharge.AllocationChargeRepos{BillableCharge: uc.c.r.BillableCharge, ChargeComponent: uc.c.r.ChargeComponent, Translator: uc.c.s.Translator},
				&billablecharge.AllocationChargeInput{
					SubscriptionID:        s.GetSubscriptionId(),
					ClientID:              s.GetClientId(),
					AgreementLineTermID:   pl.term.GetId(),
					ChargePolicyVersionID: pl.term.GetChargePolicyVersionId(),
					AllocationShareID:     s.GetId(),
					Amount:                amounts[i],
					Currency:              comp.GetCurrency(),
					ServiceFrom:           from,
					ServiceTo:             to,
					AccountingDate:        accountingDate,
					SourceComponentID:     comp.GetId(),
					NewID:                 uc.c.s.IDGenerator.GenerateID,
					Components: []billablecharge.AllocationChargeComponent{{
						Role: pl.ver.component.GetComponentRole(), DocumentKind: pl.ver.component.GetDocumentKind(),
						BookPresentation: pl.ver.component.GetBookPresentation(), TaxPosition: pl.ver.version.GetTaxPosition(),
						Amount: amounts[i], CostSourceComponentID: comp.GetId(),
					}},
				})
			if err != nil {
				return err
			}
			if res.Replayed {
				resp.ReplayedCount++
			}
			resp.Charges = append(resp.Charges, res.Charge)
		}

		if _, err := uc.c.r.AllocationBatch.UpdateAllocationBatch(txCtx, &allocationbatchpb.UpdateAllocationBatchRequest{Data: &allocationbatchpb.AllocationBatch{
			Id: batch.GetId(), Status: allocationbatchpb.AllocationBatchStatus_ALLOCATION_BATCH_STATUS_PUBLISHED,
			TotalAmount: i64p(comp.GetAmount()), Currency: strp(comp.GetCurrency()),
			PublishedAt: i64p(ms), PublishedBy: strp(userID), DateModified: i64p(ms), DateModifiedString: strp(ts),
		}}); err != nil {
			return err
		}
		batch.Status = allocationbatchpb.AllocationBatchStatus_ALLOCATION_BATCH_STATUS_PUBLISHED
		batch.PublishedAt, batch.PublishedBy = i64p(ms), strp(userID)
		batch.TotalAmount, batch.Currency = i64p(comp.GetAmount()), strp(comp.GetCurrency())

		claim := costsourcecomponentpb.SourceClaimKind_SOURCE_CLAIM_KIND_ALLOCATION
		if _, err := uc.c.r.CostSourceComponent.UpdateCostSourceComponent(txCtx, &costsourcecomponentpb.UpdateCostSourceComponentRequest{Data: &costsourcecomponentpb.CostSourceComponent{
			Id: comp.GetId(), ClaimKind: &claim, ClaimRefId: strp(batch.GetId()), ClaimedAt: i64p(ms),
			DateModified: i64p(ms), DateModifiedString: strp(ts),
		}}); err != nil {
			return err
		}
		resp.Data, resp.Shares = batch, shares
		out = resp
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// versionInfo is the pinned policy version plus its single component.
type versionInfo struct {
	version   *chargepolicyversionpb.ChargePolicyVersion
	component *chargepolicycomponentpb.ChargePolicyComponent
}

// loadVersion reads the pinned version and its single component; S1 recovery charges carry one
// component (build-spec §6.3), so zero or several, or an unset enum, is policy_component_invalid.
// A repository failure is logged and returned as an infrastructure error, not as a policy refusal (C9).
func (c *writeCore) loadVersion(ctx context.Context, versionID string) (*versionInfo, error) {
	vr, err := c.r.ChargePolicyVersion.ReadChargePolicyVersion(ctx, &chargepolicyversionpb.ReadChargePolicyVersionRequest{Data: &chargepolicyversionpb.ChargePolicyVersion{Id: versionID}})
	if err != nil {
		log.Printf("allocation_batch: read charge_policy_version=%s: %v", versionID, err)
		return nil, fmt.Errorf("allocation_batch: read charge policy version: %w", err)
	}
	if vr == nil || len(vr.Data) == 0 {
		return nil, c.refuse(ctx, codePolicyComponentInvalid, "charge policy version not found")
	}
	cr, err := c.r.ChargePolicyComponent.ListChargePolicyComponents(ctx, &chargepolicycomponentpb.ListChargePolicyComponentsRequest{
		Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
			Field: "charge_policy_version_id",
			FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
				Value: versionID, Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true}},
		}}},
	})
	if err != nil {
		log.Printf("allocation_batch: list charge_policy_component version=%s: %v", versionID, err)
		return nil, fmt.Errorf("allocation_batch: list charge policy components: %w", err)
	}
	var comps []*chargepolicycomponentpb.ChargePolicyComponent
	for _, x := range cr.GetData() {
		if x.GetChargePolicyVersionId() == versionID && x.GetActive() {
			comps = append(comps, x)
		}
	}
	if len(comps) != 1 {
		return nil, c.refuse(ctx, codePolicyComponentInvalid, fmt.Sprintf("version has %d components, exactly one is required", len(comps)))
	}
	v, k := vr.Data[0], comps[0]
	if k.GetComponentRole() == 0 || k.GetDocumentKind() == 0 || k.GetBookPresentation() == 0 || v.GetTaxPosition() == 0 {
		return nil, c.refuse(ctx, codePolicyComponentInvalid, "incomplete component or tax position")
	}
	return &versionInfo{version: v, component: k}, nil
}
