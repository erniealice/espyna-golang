package billable_charge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
)

// AdjustBillableChargeUseCase creates a CORRECTION charge (signed negative delta, predecessor =
// the original, key "<orig key>#r<n>") plus one component for the delta (billable_charge:adjust,
// CheckStrict). Request: BillableChargeId, NewAmount (the new EFFECTIVE amount in centavos,
// 0 <= x < current), Reason; response: Data = the correction. Only an ISSUED ORIGINAL can be
// adjusted; the original row is never rewritten (immutable). The correction is OPEN and is issued as
// a CREDIT_NOTE by IssueRecoveryDocuments. S1: the delta component copies the role / document kind /
// presentation / tax position of the original's first component.
type AdjustBillableChargeUseCase struct {
	repos Repositories
	svc   Services
}

func eq(field, value string) *commonpb.FilterRequest {
	return &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
		Field: field,
		FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
			Value: value, Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true,
		}},
	}}}
}

func (uc *AdjustBillableChargeUseCase) Execute(ctx context.Context, req *billablechargepb.AdjustBillableChargeRequest) (*billablechargepb.AdjustBillableChargeResponse, error) {
	if err := uc.svc.gateStrict(ctx, entityid.ActionAdjust); err != nil {
		return nil, err
	}
	if uc.repos.BillableCharge == nil || uc.repos.ChargeComponent == nil {
		return nil, fmt.Errorf("billable_charge: adjust repositories unavailable")
	}
	if req == nil || strings.TrimSpace(req.GetBillableChargeId()) == "" {
		return nil, uc.svc.refuse(ctx, codeNotFound)
	}
	if req.GetNewAmount() < 0 {
		return nil, uc.svc.refuse(ctx, codeAdjustNotDownward)
	}
	if uc.svc.Transactor == nil || !uc.svc.Transactor.SupportsTransactions() {
		return nil, uc.svc.refuse(ctx, codeTransactionRequired)
	}
	if uc.svc.IDGenerator == nil {
		return nil, fmt.Errorf("billable_charge: id generator unavailable")
	}
	var out *billablechargepb.BillableCharge
	err := uc.svc.Transactor.ExecuteInTransaction(ctx, func(tx context.Context) error {
		// Lock the original: concurrent adjustments serialize and see each other's corrections.
		// No unlocked-read fallback (C4); only the adapter's not-found sentinel is not_found (C9).
		l, ok := uc.repos.BillableCharge.(domainports.BillableChargeLocker)
		if !ok {
			return uc.svc.refuse(tx, codeLockUnavailable)
		}
		orig, err := l.LockBillableChargeForUpdate(tx, req.GetBillableChargeId())
		if err != nil {
			if errors.Is(err, domainports.ErrLockedRowNotFound) {
				return uc.svc.refuse(tx, codeNotFound)
			}
			log.Printf("billable_charge: adjust lock charge=%s: %v", req.GetBillableChargeId(), err)
			return fmt.Errorf("billable_charge: lock: %w", err)
		}
		if orig == nil {
			return uc.svc.refuse(tx, codeNotFound)
		}
		if orig.GetChargeKind() != billablechargepb.BillableChargeKind_BILLABLE_CHARGE_KIND_ORIGINAL ||
			orig.GetStatus() != billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_ISSUED {
			return uc.svc.refuse(tx, codeNotIssued)
		}
		prior, err := uc.repos.BillableCharge.ListBillableCharges(tx, &billablechargepb.ListBillableChargesRequest{Filters: eq("predecessor_id", orig.GetId())})
		if err != nil {
			log.Printf("billable_charge: adjust list corrections charge=%s: %v", orig.GetId(), err)
			return fmt.Errorf("billable_charge: list corrections: %w", err)
		}
		effective := orig.GetAmount()
		for _, p := range prior.GetData() {
			if p.GetStatus() != billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_CANCELLED {
				effective += p.GetAmount()
			}
		}
		if req.GetNewAmount() >= effective {
			return uc.svc.refuse(tx, codeAdjustNotDownward)
		}
		delta := req.GetNewAmount() - effective // negative

		comps, err := uc.repos.ChargeComponent.ListChargeComponents(tx, &chargecomponentpb.ListChargeComponentsRequest{Filters: eq("billable_charge_id", orig.GetId())})
		if err != nil {
			log.Printf("billable_charge: adjust list components charge=%s: %v", orig.GetId(), err)
			return fmt.Errorf("billable_charge: list components: %w", err)
		}
		if len(comps.GetData()) == 0 {
			return uc.svc.refuse(tx, codeValidation, "original has no components")
		}
		base := comps.Data[0]
		for _, c := range comps.Data { // deterministic: lowest component id
			if c.GetId() < base.GetId() {
				base = c
			}
		}

		key := fmt.Sprintf("%s#r%d", orig.GetObligationKey(), len(prior.GetData())+1)
		sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%s", key, delta, req.GetReason())))
		now := time.Now()
		id := uc.svc.IDGenerator.GenerateID()
		corr := &billablechargepb.BillableCharge{
			Id: id, ObligationKey: key, ContentHash: hex.EncodeToString(sum[:]),
			SubscriptionId: orig.SubscriptionId, ClientId: orig.ClientId, AgreementLineTermId: orig.AgreementLineTermId,
			ChargePolicyVersionId: orig.ChargePolicyVersionId, AllocationShareId: orig.AllocationShareId,
			ChargeKind:       billablechargepb.BillableChargeKind_BILLABLE_CHARGE_KIND_CORRECTION,
			PredecessorId:    strp(orig.GetId()),
			EvidenceRevision: orig.GetEvidenceRevision(), RatingRevision: orig.GetRatingRevision(),
			Amount: delta, Currency: orig.GetCurrency(),
			ServiceFrom: orig.ServiceFrom, ServiceTo: orig.ServiceTo, AccountingDate: orig.AccountingDate,
			Status: billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_OPEN,
			Active: true, DateCreated: &[]int64{now.UnixMilli()}[0], DateModified: &[]int64{now.UnixMilli()}[0],
		}
		if !blank(req.GetReason()) {
			corr.Reason = strp(strings.TrimSpace(req.GetReason()))
		}
		if _, err := uc.repos.BillableCharge.CreateBillableCharge(tx, &billablechargepb.CreateBillableChargeRequest{Data: corr}); err != nil {
			return fmt.Errorf("billable_charge: create correction: %w", err)
		}
		comp := &chargecomponentpb.ChargeComponent{
			Id: uc.svc.IDGenerator.GenerateID(), BillableChargeId: id,
			ComponentRole: base.GetComponentRole(), DocumentKind: base.GetDocumentKind(),
			BookPresentation: base.GetBookPresentation(), TaxPosition: base.GetTaxPosition(),
			Amount: delta, Currency: orig.GetCurrency(), CostSourceComponentId: base.CostSourceComponentId,
			Active: true, DateCreated: corr.DateCreated, DateModified: corr.DateModified,
		}
		if _, err := uc.repos.ChargeComponent.CreateChargeComponent(tx, &chargecomponentpb.CreateChargeComponentRequest{Data: comp}); err != nil {
			return fmt.Errorf("billable_charge: create correction component: %w", err)
		}
		out = corr
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &billablechargepb.AdjustBillableChargeResponse{Data: out, Success: true}, nil
}

func blank(s string) bool   { return strings.TrimSpace(s) == "" }
func strp(s string) *string { return &s }
