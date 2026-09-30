package billable_charge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
)

// Charge creation from a published allocation (20260927-usage-and-pass-through-charges, build-spec
// §6.3 PublishAllocationBatch). This file is the ONLY creation-side entry into billable_charge
// owned by the allocation flow (W3-A); issuing/adjusting live in sibling files.

// AllocationChargeRepos are the repositories the constructor writes through.
type AllocationChargeRepos struct {
	BillableCharge  billablechargepb.BillableChargeDomainServiceServer
	ChargeComponent chargecomponentpb.ChargeComponentDomainServiceServer
	// Translator translates the named refusals (billable_charge.errors.<code>); nil = English defaults.
	Translator ports.Translator
}

// The Allocation* types below (input, component, result) are intra-transaction DOMAIN HELPER types
// of CreateFromAllocation, called only by allocation_batch's publish inside ITS transaction. They
// are not an Execute contract: presentation layers and consumers never construct them, so the
// "proto request/response for every verb" rule (C1) does not apply to them (R4 m10).

// AllocationChargeComponent is one component line of the charge to create.
type AllocationChargeComponent struct {
	Role                  enumspb.ChargeComponentRole
	DocumentKind          enumspb.ChargeDocumentKind
	BookPresentation      enumspb.BookPresentation
	TaxPosition           enumspb.TaxPosition
	Amount                int64
	CostSourceComponentID string
}

// AllocationChargeInput describes an ORIGINAL charge for one recoverable share.
type AllocationChargeInput struct {
	SubscriptionID        string
	ClientID              string
	AgreementLineTermID   string
	ChargePolicyVersionID string
	AllocationShareID     string
	Amount                int64 // signed centavos; > 0 for an original
	Currency              string
	ServiceFrom           string
	ServiceTo             string
	AccountingDate        string
	SourceComponentID     string // used for the obligation key
	Components            []AllocationChargeComponent
	NewID                 func() string
}

// AllocationObligationKey is the frozen obligation key: SRC:<component>:SUB:<subscription>:SVC:<from>/<to>.
func AllocationObligationKey(componentID, subscriptionID, from, to string) string {
	return fmt.Sprintf("SRC:%s:SUB:%s:SVC:%s/%s", componentID, subscriptionID, from, to)
}

// AllocationContentHash hashes everything that defines the obligation's content (not its
// identity): a replay with the same key must carry the same hash.
func AllocationContentHash(in *AllocationChargeInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "v1|sub=%s|client=%s|term=%s|ver=%s|amt=%d|cur=%s|from=%s|to=%s",
		in.SubscriptionID, in.ClientID, in.AgreementLineTermID, in.ChargePolicyVersionID, in.Amount, in.Currency, in.ServiceFrom, in.ServiceTo)
	for _, c := range in.Components {
		fmt.Fprintf(&b, "|c=%d,%d,%d,%d,%d,%s", c.Role, c.DocumentKind, c.BookPresentation, c.TaxPosition, c.Amount, c.CostSourceComponentID)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// AllocationChargeResult is the outcome of CreateChargeFromAllocation.
type AllocationChargeResult struct {
	Charge   *billablechargepb.BillableCharge
	Replayed bool // an identical charge already existed; nothing was written
}

// CreateChargeFromAllocation creates (or replays) the ORIGINAL charge + its components for one
// recoverable share. The caller owns the transaction, the workspace/actor context and the locks.
// Same obligation key + same content hash returns the existing charge (Replayed); the same key
// with a different hash is refused with the named refusal obligation_conflict (ErrorCode()).
func CreateChargeFromAllocation(ctx context.Context, repos AllocationChargeRepos, in *AllocationChargeInput) (*AllocationChargeResult, error) {
	if repos.BillableCharge == nil || repos.ChargeComponent == nil {
		return nil, refuse(ctx, repos.Translator, codeValidation, "repositories unavailable")
	}
	if in == nil || in.NewID == nil || in.SubscriptionID == "" || in.ClientID == "" || in.SourceComponentID == "" ||
		in.Currency == "" || in.ServiceFrom == "" || in.ServiceTo == "" || in.Amount <= 0 || len(in.Components) == 0 {
		return nil, refuse(ctx, repos.Translator, codeValidation, "invalid allocation charge input")
	}
	var sum int64
	for _, c := range in.Components {
		sum += c.Amount
	}
	if sum != in.Amount {
		return nil, refuse(ctx, repos.Translator, codeValidation, "components do not add up to the charge amount")
	}
	key := AllocationObligationKey(in.SourceComponentID, in.SubscriptionID, in.ServiceFrom, in.ServiceTo)
	hash := AllocationContentHash(in)

	existing, err := repos.BillableCharge.ListBillableCharges(ctx, &billablechargepb.ListBillableChargesRequest{
		Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
			Field: "obligation_key",
			FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
				Value: key, Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true,
			}},
		}}},
	})
	if err != nil {
		log.Printf("billable_charge: obligation lookup key=%s: %v", key, err)
		return nil, fmt.Errorf("billable_charge: obligation lookup: %w", err)
	}
	for _, row := range existing.GetData() {
		if row.GetObligationKey() != key {
			continue // defensive: never trust an adapter that ignored the filter
		}
		if row.GetContentHash() != hash {
			return nil, refuse(ctx, repos.Translator, codeObligationConflict)
		}
		return &AllocationChargeResult{Charge: row, Replayed: true}, nil
	}

	now := time.Now()
	ms, ts := now.UnixMilli(), now.Format(time.RFC3339)
	charge := &billablechargepb.BillableCharge{
		Id:                 in.NewID(),
		ObligationKey:      key,
		ContentHash:        hash,
		SubscriptionId:     allocStrp(in.SubscriptionID),
		ClientId:           allocStrp(in.ClientID),
		ChargeKind:         billablechargepb.BillableChargeKind_BILLABLE_CHARGE_KIND_ORIGINAL,
		EvidenceRevision:   1,
		RatingRevision:     1,
		Amount:             in.Amount,
		Currency:           in.Currency,
		ServiceFrom:        allocStrp(in.ServiceFrom),
		ServiceTo:          allocStrp(in.ServiceTo),
		Status:             billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_OPEN,
		Active:             true,
		DateCreated:        &ms,
		DateCreatedString:  &ts,
		DateModified:       &ms,
		DateModifiedString: &ts,
	}
	if in.AgreementLineTermID != "" {
		charge.AgreementLineTermId = allocStrp(in.AgreementLineTermID)
	}
	if in.ChargePolicyVersionID != "" {
		charge.ChargePolicyVersionId = allocStrp(in.ChargePolicyVersionID)
	}
	if in.AllocationShareID != "" {
		charge.AllocationShareId = allocStrp(in.AllocationShareID)
	}
	if in.AccountingDate != "" {
		charge.AccountingDate = allocStrp(in.AccountingDate)
	}
	created, err := repos.BillableCharge.CreateBillableCharge(ctx, &billablechargepb.CreateBillableChargeRequest{Data: charge})
	if err != nil {
		return nil, fmt.Errorf("billable_charge: create: %w", err)
	}
	out := charge
	if d := created.GetData(); len(d) > 0 && d[0] != nil {
		out = d[0]
	}
	for _, c := range in.Components {
		comp := &chargecomponentpb.ChargeComponent{
			Id:                 in.NewID(),
			BillableChargeId:   out.GetId(),
			ComponentRole:      c.Role,
			DocumentKind:       c.DocumentKind,
			BookPresentation:   c.BookPresentation,
			TaxPosition:        c.TaxPosition,
			Amount:             c.Amount,
			Currency:           in.Currency,
			Active:             true,
			DateCreated:        &ms,
			DateCreatedString:  &ts,
			DateModified:       &ms,
			DateModifiedString: &ts,
		}
		if c.CostSourceComponentID != "" {
			comp.CostSourceComponentId = allocStrp(c.CostSourceComponentID)
		}
		if _, err := repos.ChargeComponent.CreateChargeComponent(ctx, &chargecomponentpb.CreateChargeComponentRequest{Data: comp}); err != nil {
			return nil, fmt.Errorf("billable_charge: create component: %w", err)
		}
	}
	return &AllocationChargeResult{Charge: out}, nil
}

func allocStrp(s string) *string { return &s }
