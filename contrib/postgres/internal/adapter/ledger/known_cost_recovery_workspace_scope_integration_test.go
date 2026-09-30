//go:build postgresql

package ledger

import (
	"context"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core/scopetest"
	"testing"

	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	chargeeffectpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_effect"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
)

// ChargeEffect: every by-id and list operation carries a trusted, non-empty workspace predicate. A
// foreign workspace id, or a missing workspace, sees "not found"; a caller-supplied workspace_id
// never wins over the trusted one. Needs migration 20260930110000 on the leasing_usage1 clone.
func TestChargeEffectWorkspaceScope(t *testing.T) {
	h := scopetest.New(t, "charge_effect")
	repo := NewPostgresChargeEffectRepository(h.Ops, entityid.ChargeEffect).(*PostgresChargeEffectRepository)
	const id = "s1scope-charge_effect"
	h.Run(t, func(base, ctxA, ctxB, ctxNone context.Context) {
		if _, err := repo.CreateChargeEffect(ctxA, &chargeeffectpb.CreateChargeEffectRequest{Data: &chargeeffectpb.ChargeEffect{
			Id: id, WorkspaceId: scopetest.WsB, // must be ignored in favour of the trusted workspace
			Active: true, EventKind: enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_ISSUE, EventId: "ev1", PostingRole: enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_RECEIVABLE, AccountId: "a1", Direction: chargeeffectpb.EffectDirection_EFFECT_DIRECTION_DEBIT, Amount: 1, Currency: "PHP", SourceRef: scopetest.Str("orig")}}); err != nil {
			t.Fatalf("create in own workspace: %v", err)
		}
		got, err := repo.ReadChargeEffect(ctxA, &chargeeffectpb.ReadChargeEffectRequest{Data: &chargeeffectpb.ChargeEffect{Id: id}})
		if err != nil || len(got.Data) != 1 || got.Data[0].GetWorkspaceId() != scopetest.WsA {
			t.Fatalf("own read must succeed with the trusted workspace stamped: %v %+v", err, got)
		}
		for name, ctx := range map[string]context.Context{"foreign workspace": ctxB, "no workspace": ctxNone} {
			if _, err := repo.ReadChargeEffect(ctx, &chargeeffectpb.ReadChargeEffectRequest{Data: &chargeeffectpb.ChargeEffect{Id: id}}); err == nil {
				t.Errorf("%s: read must fail (not found / fail closed)", name)
			}
			if _, err := repo.GetChargeEffectItemPageData(ctx, &chargeeffectpb.GetChargeEffectItemPageDataRequest{ChargeEffectId: id}); err == nil {
				t.Errorf("%s: item page data must fail", name)
			}
			if _, err := repo.UpdateChargeEffect(ctx, &chargeeffectpb.UpdateChargeEffectRequest{Data: &chargeeffectpb.ChargeEffect{Id: id, SourceRef: scopetest.Str("hijacked")}}); err == nil {
				t.Errorf("%s: update must fail", name)
			}
			if _, err := repo.DeleteChargeEffect(ctx, &chargeeffectpb.DeleteChargeEffectRequest{Data: &chargeeffectpb.ChargeEffect{Id: id}}); err == nil {
				t.Errorf("%s: delete must fail", name)
			}
			if l, err := repo.ListChargeEffects(ctx, &chargeeffectpb.ListChargeEffectsRequest{}); err == nil {
				for _, r := range l.Data {
					if r.GetId() == id {
						t.Errorf("%s: list leaked a foreign row", name)
					}
				}
			}
			if l, err := repo.GetChargeEffectListPageData(ctx, &chargeeffectpb.GetChargeEffectListPageDataRequest{}); err == nil {
				for _, r := range l.ChargeEffectList {
					if r.GetId() == id {
						t.Errorf("%s: list page data leaked a foreign row", name)
					}
				}
			}
		}
		// C6: the row is an immutable financial record - even the owning workspace cannot delete it.
		if _, err := repo.DeleteChargeEffect(ctxA, &chargeeffectpb.DeleteChargeEffectRequest{Data: &chargeeffectpb.ChargeEffect{Id: id}}); err == nil {
			t.Error("own-workspace delete must be refused (immutable)")
		} else if !postgresCore.IsImmutableRecord(err) {
			t.Errorf("own-workspace delete must be an immutable_record refusal, got %v", err)
		}
		// The stored row is untouched by the refused writes.
		still, err := repo.ReadChargeEffect(ctxA, &chargeeffectpb.ReadChargeEffectRequest{Data: &chargeeffectpb.ChargeEffect{Id: id}})
		if err != nil || len(still.Data) != 1 || still.Data[0].GetSourceRef() != "orig" {
			t.Errorf("foreign write changed the row: %v %+v", err, still)
		}
		// An OR filter would let the tenant predicate be OR-ed away: refused outright.
		if _, err := repo.ListChargeEffects(ctxA, &chargeeffectpb.ListChargeEffectsRequest{Filters: &commonpb.FilterRequest{Logic: commonpb.FilterLogic_OR, Filters: []*commonpb.TypedFilter{{Field: "id"}}}}); err == nil {
			t.Error("OR filter logic must be refused")
		}
	})
}
