//go:build postgresql

package subscription

import (
	"context"
	"errors"
	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core/scopetest"
	"testing"

	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
	agreementlinetermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/agreement_line_term"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
)

// AgreementLineTerm: every by-id and list operation carries a trusted, non-empty workspace predicate. A
// foreign workspace id, or a missing workspace, sees "not found"; a caller-supplied workspace_id
// never wins over the trusted one. Needs migration 20260930110000 on the leasing_usage1 clone.
func TestAgreementLineTermWorkspaceScope(t *testing.T) {
	h := scopetest.New(t, "agreement_line_term")
	repo := NewPostgresAgreementLineTermRepository(h.Ops, entityid.AgreementLineTerm).(*PostgresAgreementLineTermRepository)
	const id = "s1scope-agreement_line_term"
	h.Run(t, func(base, ctxA, ctxB, ctxNone context.Context) {
		if _, err := repo.CreateAgreementLineTerm(ctxA, &agreementlinetermpb.CreateAgreementLineTermRequest{Data: &agreementlinetermpb.AgreementLineTerm{
			Id: id, WorkspaceId: scopetest.WsB, // must be ignored in favour of the trusted workspace
			Active: true, SubscriptionId: "s1", ProductPricePlanId: "p1", ClientId: "cl1", ChargePolicyVersionId: "v1", EffectiveFrom: "2026-01-01", Origin: agreementlinetermpb.AgreementLineTermOrigin_AGREEMENT_LINE_TERM_ORIGIN_COPIED, AcceptedBy: scopetest.Str("orig")}}); err != nil {
			t.Fatalf("create in own workspace: %v", err)
		}
		got, err := repo.ReadAgreementLineTerm(ctxA, &agreementlinetermpb.ReadAgreementLineTermRequest{Data: &agreementlinetermpb.AgreementLineTerm{Id: id}})
		if err != nil || len(got.Data) != 1 || got.Data[0].GetWorkspaceId() != scopetest.WsA {
			t.Fatalf("own read must succeed with the trusted workspace stamped: %v %+v", err, got)
		}
		for name, ctx := range map[string]context.Context{"foreign workspace": ctxB, "no workspace": ctxNone} {
			if _, err := repo.ReadAgreementLineTerm(ctx, &agreementlinetermpb.ReadAgreementLineTermRequest{Data: &agreementlinetermpb.AgreementLineTerm{Id: id}}); err == nil {
				t.Errorf("%s: read must fail (not found / fail closed)", name)
			}
			if _, err := repo.GetAgreementLineTermItemPageData(ctx, &agreementlinetermpb.GetAgreementLineTermItemPageDataRequest{AgreementLineTermId: id}); err == nil {
				t.Errorf("%s: item page data must fail", name)
			}
			if _, err := repo.UpdateAgreementLineTerm(ctx, &agreementlinetermpb.UpdateAgreementLineTermRequest{Data: &agreementlinetermpb.AgreementLineTerm{Id: id, AcceptedBy: scopetest.Str("hijacked")}}); err == nil {
				t.Errorf("%s: update must fail", name)
			}
			if _, err := repo.DeleteAgreementLineTerm(ctx, &agreementlinetermpb.DeleteAgreementLineTermRequest{Data: &agreementlinetermpb.AgreementLineTerm{Id: id}}); err == nil {
				t.Errorf("%s: delete must fail", name)
			}
			if l, err := repo.ListAgreementLineTerms(ctx, &agreementlinetermpb.ListAgreementLineTermsRequest{}); err == nil {
				for _, r := range l.Data {
					if r.GetId() == id {
						t.Errorf("%s: list leaked a foreign row", name)
					}
				}
			}
			if l, err := repo.GetAgreementLineTermListPageData(ctx, &agreementlinetermpb.GetAgreementLineTermListPageDataRequest{}); err == nil {
				for _, r := range l.AgreementLineTermList {
					if r.GetId() == id {
						t.Errorf("%s: list page data leaked a foreign row", name)
					}
				}
			}
		}
		// The stored row is untouched by the refused writes.
		still, err := repo.ReadAgreementLineTerm(ctxA, &agreementlinetermpb.ReadAgreementLineTermRequest{Data: &agreementlinetermpb.AgreementLineTerm{Id: id}})
		if err != nil || len(still.Data) != 1 || still.Data[0].GetAcceptedBy() != "orig" {
			t.Errorf("foreign write changed the row: %v %+v", err, still)
		}
		// An OR filter would let the tenant predicate be OR-ed away: refused outright.
		if _, err := repo.ListAgreementLineTerms(ctxA, &agreementlinetermpb.ListAgreementLineTermsRequest{Filters: &commonpb.FilterRequest{Logic: commonpb.FilterLogic_OR, Filters: []*commonpb.TypedFilter{{Field: "id"}}}}); err == nil {
			t.Error("OR filter logic must be refused")
		}
	})
}

// BillableCharge: every by-id and list operation carries a trusted, non-empty workspace predicate. A
// foreign workspace id, or a missing workspace, sees "not found"; a caller-supplied workspace_id
// never wins over the trusted one. Needs migration 20260930110000 on the leasing_usage1 clone.
func TestBillableChargeWorkspaceScope(t *testing.T) {
	h := scopetest.New(t, "billable_charge")
	repo := NewPostgresBillableChargeRepository(h.Ops, entityid.BillableCharge).(*PostgresBillableChargeRepository)
	const id = "s1scope-billable_charge"
	h.Run(t, func(base, ctxA, ctxB, ctxNone context.Context) {
		if _, err := repo.CreateBillableCharge(ctxA, &billablechargepb.CreateBillableChargeRequest{Data: &billablechargepb.BillableCharge{
			Id: id, WorkspaceId: scopetest.WsB, // must be ignored in favour of the trusted workspace
			Active: true, ObligationKey: "SCOPE:billable_charge", ContentHash: "h", ChargeKind: billablechargepb.BillableChargeKind_BILLABLE_CHARGE_KIND_ORIGINAL, Amount: 1, Currency: "PHP", Status: billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_OPEN, Reason: scopetest.Str("orig")}}); err != nil {
			t.Fatalf("create in own workspace: %v", err)
		}
		got, err := repo.ReadBillableCharge(ctxA, &billablechargepb.ReadBillableChargeRequest{Data: &billablechargepb.BillableCharge{Id: id}})
		if err != nil || len(got.Data) != 1 || got.Data[0].GetWorkspaceId() != scopetest.WsA {
			t.Fatalf("own read must succeed with the trusted workspace stamped: %v %+v", err, got)
		}
		for name, ctx := range map[string]context.Context{"foreign workspace": ctxB, "no workspace": ctxNone} {
			if _, err := repo.ReadBillableCharge(ctx, &billablechargepb.ReadBillableChargeRequest{Data: &billablechargepb.BillableCharge{Id: id}}); err == nil {
				t.Errorf("%s: read must fail (not found / fail closed)", name)
			}
			if _, err := repo.GetBillableChargeItemPageData(ctx, &billablechargepb.GetBillableChargeItemPageDataRequest{BillableChargeId: id}); err == nil {
				t.Errorf("%s: item page data must fail", name)
			}
			if _, err := repo.UpdateBillableCharge(ctx, &billablechargepb.UpdateBillableChargeRequest{Data: &billablechargepb.BillableCharge{Id: id, Reason: scopetest.Str("hijacked")}}); err == nil {
				t.Errorf("%s: update must fail", name)
			}
			if _, err := repo.DeleteBillableCharge(ctx, &billablechargepb.DeleteBillableChargeRequest{Data: &billablechargepb.BillableCharge{Id: id}}); err == nil {
				t.Errorf("%s: delete must fail", name)
			}
			if l, err := repo.ListBillableCharges(ctx, &billablechargepb.ListBillableChargesRequest{}); err == nil {
				for _, r := range l.Data {
					if r.GetId() == id {
						t.Errorf("%s: list leaked a foreign row", name)
					}
				}
			}
			if l, err := repo.GetBillableChargeListPageData(ctx, &billablechargepb.GetBillableChargeListPageDataRequest{}); err == nil {
				for _, r := range l.BillableChargeList {
					if r.GetId() == id {
						t.Errorf("%s: list page data leaked a foreign row", name)
					}
				}
			}
			if _, err := repo.LockBillableChargeForUpdate(ctx, id); err == nil {
				t.Errorf("%s: LockBillableChargeForUpdate must report not found", name)
			}
		}
		// The stored row is untouched by the refused writes.
		still, err := repo.ReadBillableCharge(ctxA, &billablechargepb.ReadBillableChargeRequest{Data: &billablechargepb.BillableCharge{Id: id}})
		if err != nil || len(still.Data) != 1 || still.Data[0].GetReason() != "orig" {
			t.Errorf("foreign write changed the row: %v %+v", err, still)
		}
		// C6: even the OWNING workspace cannot hard-delete a billable charge (immutable financial row).
		if _, err := repo.DeleteBillableCharge(ctxA, &billablechargepb.DeleteBillableChargeRequest{Data: &billablechargepb.BillableCharge{Id: id}}); !errors.Is(err, domainports.ErrImmutableRow) {
			t.Errorf("own delete must be refused as immutable, got %v", err)
		}
		// C9: a foreign row is reported through the not-found sentinel (never a raw lock error).
		if _, err := repo.LockBillableChargeForUpdate(ctxB, id); !errors.Is(err, domainports.ErrLockedRowNotFound) {
			t.Errorf("foreign lock must wrap ErrLockedRowNotFound, got %v", err)
		}
		if still2, err := repo.ReadBillableCharge(ctxA, &billablechargepb.ReadBillableChargeRequest{Data: &billablechargepb.BillableCharge{Id: id}}); err != nil || len(still2.Data) != 1 {
			t.Errorf("the refused delete must leave the row: %v", err)
		}
		// An OR filter would let the tenant predicate be OR-ed away: refused outright.
		if _, err := repo.ListBillableCharges(ctxA, &billablechargepb.ListBillableChargesRequest{Filters: &commonpb.FilterRequest{Logic: commonpb.FilterLogic_OR, Filters: []*commonpb.TypedFilter{{Field: "id"}}}}); err == nil {
			t.Error("OR filter logic must be refused")
		}
		if v, err := repo.LockBillableChargeForUpdate(ctxA, id); err != nil || v.GetId() != id {
			t.Errorf("own LockBillableChargeForUpdate: %v", err)
		}
	})
	// Outside a transaction the lock fails closed (SELECT ... FOR UPDATE needs a tx).
	if _, err := repo.LockBillableChargeForUpdate(identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: scopetest.WsA}), id); err == nil {
		t.Error("LockBillableChargeForUpdate outside a transaction must fail closed")
	}
}

// ChargeComponent: every by-id and list operation carries a trusted, non-empty workspace predicate. A
// foreign workspace id, or a missing workspace, sees "not found"; a caller-supplied workspace_id
// never wins over the trusted one. Needs migration 20260930110000 on the leasing_usage1 clone.
func TestChargeComponentWorkspaceScope(t *testing.T) {
	h := scopetest.New(t, "charge_component")
	repo := NewPostgresChargeComponentRepository(h.Ops, entityid.ChargeComponent).(*PostgresChargeComponentRepository)
	const id = "s1scope-charge_component"
	h.Run(t, func(base, ctxA, ctxB, ctxNone context.Context) {
		if _, err := repo.CreateChargeComponent(ctxA, &chargecomponentpb.CreateChargeComponentRequest{Data: &chargecomponentpb.ChargeComponent{
			Id: id, WorkspaceId: scopetest.WsB, // must be ignored in favour of the trusted workspace
			Active: true, BillableChargeId: "bc1", ComponentRole: enumspb.ChargeComponentRole_CHARGE_COMPONENT_ROLE_RECOVERY_COST, DocumentKind: enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT, BookPresentation: enumspb.BookPresentation_BOOK_PRESENTATION_EXCLUDED_REIMBURSEMENT, TaxPosition: enumspb.TaxPosition_TAX_POSITION_EXCLUDED_REIMBURSEMENT, Amount: 10, Currency: "PHP"}}); err != nil {
			t.Fatalf("create in own workspace: %v", err)
		}
		got, err := repo.ReadChargeComponent(ctxA, &chargecomponentpb.ReadChargeComponentRequest{Data: &chargecomponentpb.ChargeComponent{Id: id}})
		if err != nil || len(got.Data) != 1 || got.Data[0].GetWorkspaceId() != scopetest.WsA {
			t.Fatalf("own read must succeed with the trusted workspace stamped: %v %+v", err, got)
		}
		for name, ctx := range map[string]context.Context{"foreign workspace": ctxB, "no workspace": ctxNone} {
			if _, err := repo.ReadChargeComponent(ctx, &chargecomponentpb.ReadChargeComponentRequest{Data: &chargecomponentpb.ChargeComponent{Id: id}}); err == nil {
				t.Errorf("%s: read must fail (not found / fail closed)", name)
			}
			if _, err := repo.GetChargeComponentItemPageData(ctx, &chargecomponentpb.GetChargeComponentItemPageDataRequest{ChargeComponentId: id}); err == nil {
				t.Errorf("%s: item page data must fail", name)
			}
			if _, err := repo.UpdateChargeComponent(ctx, &chargecomponentpb.UpdateChargeComponentRequest{Data: &chargecomponentpb.ChargeComponent{Id: id, Amount: 999}}); err == nil {
				t.Errorf("%s: update must fail", name)
			}
			if _, err := repo.DeleteChargeComponent(ctx, &chargecomponentpb.DeleteChargeComponentRequest{Data: &chargecomponentpb.ChargeComponent{Id: id}}); err == nil {
				t.Errorf("%s: delete must fail", name)
			}
			if l, err := repo.ListChargeComponents(ctx, &chargecomponentpb.ListChargeComponentsRequest{}); err == nil {
				for _, r := range l.Data {
					if r.GetId() == id {
						t.Errorf("%s: list leaked a foreign row", name)
					}
				}
			}
			if l, err := repo.GetChargeComponentListPageData(ctx, &chargecomponentpb.GetChargeComponentListPageDataRequest{}); err == nil {
				for _, r := range l.ChargeComponentList {
					if r.GetId() == id {
						t.Errorf("%s: list page data leaked a foreign row", name)
					}
				}
			}
		}
		// The stored row is untouched by the refused writes.
		still, err := repo.ReadChargeComponent(ctxA, &chargecomponentpb.ReadChargeComponentRequest{Data: &chargecomponentpb.ChargeComponent{Id: id}})
		if err != nil || len(still.Data) != 1 || still.Data[0].GetAmount() != int64(10) {
			t.Errorf("foreign write changed the row: %v %+v", err, still)
		}
		// C6: even the OWNING workspace cannot hard-delete a charge component (part of an immutable charge).
		if _, err := repo.DeleteChargeComponent(ctxA, &chargecomponentpb.DeleteChargeComponentRequest{Data: &chargecomponentpb.ChargeComponent{Id: id}}); !errors.Is(err, domainports.ErrImmutableRow) {
			t.Errorf("own delete must be refused as immutable, got %v", err)
		}
		// An OR filter would let the tenant predicate be OR-ed away: refused outright.
		if _, err := repo.ListChargeComponents(ctxA, &chargecomponentpb.ListChargeComponentsRequest{Filters: &commonpb.FilterRequest{Logic: commonpb.FilterLogic_OR, Filters: []*commonpb.TypedFilter{{Field: "id"}}}}); err == nil {
			t.Error("OR filter logic must be refused")
		}
	})
}
