package charge_effect

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	chargeeffectpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_effect"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
)

type fakeEffects struct {
	chargeeffectpb.UnimplementedChargeEffectDomainServiceServer
	rows []*chargeeffectpb.ChargeEffect
}

func (f *fakeEffects) CreateChargeEffect(_ context.Context, r *chargeeffectpb.CreateChargeEffectRequest) (*chargeeffectpb.CreateChargeEffectResponse, error) {
	f.rows = append(f.rows, r.Data)
	return &chargeeffectpb.CreateChargeEffectResponse{Success: true, Data: []*chargeeffectpb.ChargeEffect{r.Data}}, nil
}
func (f *fakeEffects) ListChargeEffects(_ context.Context, r *chargeeffectpb.ListChargeEffectsRequest) (*chargeeffectpb.ListChargeEffectsResponse, error) {
	want := r.GetFilters().GetFilters()[0].GetStringFilter().GetValue()
	out := &chargeeffectpb.ListChargeEffectsResponse{Success: true}
	for _, e := range f.rows {
		if e.GetEventId() == want {
			out.Data = append(out.Data, e)
		}
	}
	return out, nil
}

type fakePostings struct {
	postingpb.UnimplementedChargePolicyPostingDomainServiceServer
	rows []*postingpb.ChargePolicyPosting
}

func (f *fakePostings) ListChargePolicyPostings(_ context.Context, r *postingpb.ListChargePolicyPostingsRequest) (*postingpb.ListChargePolicyPostingsResponse, error) {
	want := r.GetFilters().GetFilters()[0].GetStringFilter().GetValue()
	out := &postingpb.ListChargePolicyPostingsResponse{Success: true}
	for _, p := range f.rows {
		if p.GetChargePolicyVersionId() == want {
			out.Data = append(out.Data, p)
		}
	}
	return out, nil
}

var _ = commonpb.FilterRequest{}

const (
	evIssue = enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_ISSUE
	evApp   = enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_APPLICATION
	evRev   = enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_REVERSAL
	rRecv   = enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_RECEIVABLE
	rClear  = enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_CLEARING
	rCash   = enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_CASH
)

func posting(id string, ev enumspb.ChargePostingEvent, role enumspb.ChargePostingRole, acct string) *postingpb.ChargePolicyPosting {
	return &postingpb.ChargePolicyPosting{Id: id, ChargePolicyVersionId: "v1", Event: ev, PostingRole: role, AccountId: acct, Active: true}
}

func fixture() (Repos, *fakeEffects) {
	fe := &fakeEffects{}
	fp := &fakePostings{rows: []*postingpb.ChargePolicyPosting{
		posting("p1", evIssue, rRecv, "acc-recv"), posting("p2", evIssue, rClear, "acc-clear"),
		posting("p3", evApp, rCash, "acc-cash"), posting("p4", evApp, rRecv, "acc-recv"),
	}}
	return Repos{ChargeEffect: fe, ChargePolicyPosting: fp}, fe
}

func sums(rows []*chargeeffectpb.ChargeEffect) (dr, cr int64) {
	for _, r := range rows {
		if r.GetDirection() == chargeeffectpb.EffectDirection_EFFECT_DIRECTION_DEBIT {
			dr += r.GetAmount()
		} else {
			cr += r.GetAmount()
		}
	}
	return
}

func find(rows []*chargeeffectpb.ChargeEffect, role enumspb.ChargePostingRole) *chargeeffectpb.ChargeEffect {
	for _, r := range rows {
		if r.GetPostingRole() == role {
			return r
		}
	}
	return nil
}

func TestRecordChargeEffectsBalancedAndIdempotent(t *testing.T) {
	ctx := context.Background()
	repos, fe := fixture()

	// ISSUE: DR RECEIVABLE / CR CLEARING, balanced, twice = still 2 rows.
	for i := 0; i < 2; i++ {
		if err := RecordChargeEffects(ctx, repos, "v1", evIssue, "doc1", "2026-09-30", "PHP", 12345, "ref"); err != nil {
			t.Fatal(err)
		}
	}
	if len(fe.rows) != 2 {
		t.Fatalf("rows=%d want 2", len(fe.rows))
	}
	if dr, cr := sums(fe.rows); dr != 12345 || cr != 12345 {
		t.Fatalf("unbalanced dr=%d cr=%d", dr, cr)
	}
	if r := find(fe.rows, rRecv); r.GetDirection() != chargeeffectpb.EffectDirection_EFFECT_DIRECTION_DEBIT || r.GetAccountId() != "acc-recv" || r.GetId() == "" {
		t.Fatalf("receivable leg wrong: %v", r)
	}
	if r := find(fe.rows, rClear); r.GetDirection() != chargeeffectpb.EffectDirection_EFFECT_DIRECTION_CREDIT || r.GetAccountId() != "acc-clear" {
		t.Fatalf("clearing leg wrong: %v", r)
	}

	// Partial retry (one leg already present) only writes the missing leg.
	fe.rows = fe.rows[:1]
	if err := RecordChargeEffects(ctx, repos, "v1", evIssue, "doc1", "", "PHP", 12345, ""); err != nil {
		t.Fatal(err)
	}
	if len(fe.rows) != 2 {
		t.Fatalf("rows=%d want 2 after partial retry", len(fe.rows))
	}

	// APPLICATION: DR CASH / CR RECEIVABLE.
	before := len(fe.rows)
	if err := RecordChargeEffects(ctx, repos, "v1", evApp, "app1", "", "PHP", 500, ""); err != nil {
		t.Fatal(err)
	}
	app := fe.rows[before:]
	if find(app, rCash).GetDirection() != chargeeffectpb.EffectDirection_EFFECT_DIRECTION_DEBIT || find(app, rRecv).GetDirection() != chargeeffectpb.EffectDirection_EFFECT_DIRECTION_CREDIT {
		t.Fatal("application legs wrong")
	}

	// REVERSAL mirrors the application: DR RECEIVABLE / CR CASH; kind REVERSAL.
	before = len(fe.rows)
	if err := RecordChargeEffects(ctx, repos, "v1", evRev, "rev1", "", "PHP", 500, ""); err != nil {
		t.Fatal(err)
	}
	rev := fe.rows[before:]
	if len(rev) != 2 || rev[0].GetEventKind() != evRev ||
		find(rev, rRecv).GetDirection() != chargeeffectpb.EffectDirection_EFFECT_DIRECTION_DEBIT ||
		find(rev, rCash).GetDirection() != chargeeffectpb.EffectDirection_EFFECT_DIRECTION_CREDIT {
		t.Fatalf("reversal legs wrong: %v", rev)
	}

	// Reversal of an ISSUE mirrors it.
	before = len(fe.rows)
	if err := RecordReversalEffects(ctx, repos, "v1", evIssue, "rev2", "", "PHP", 100, ""); err != nil {
		t.Fatal(err)
	}
	if find(fe.rows[before:], rRecv).GetDirection() != chargeeffectpb.EffectDirection_EFFECT_DIRECTION_CREDIT {
		t.Fatal("issue reversal should credit receivable")
	}

	// Credit-note issue (negative amount) swaps directions, posts magnitude.
	before = len(fe.rows)
	if err := RecordChargeEffects(ctx, repos, "v1", evIssue, "cn1", "", "PHP", -700, ""); err != nil {
		t.Fatal(err)
	}
	cn := fe.rows[before:]
	if find(cn, rRecv).GetDirection() != chargeeffectpb.EffectDirection_EFFECT_DIRECTION_CREDIT || find(cn, rRecv).GetAmount() != 700 {
		t.Fatal("credit note legs wrong")
	}

	// Zero amount writes nothing.
	before = len(fe.rows)
	if err := RecordChargeEffects(ctx, repos, "v1", evIssue, "z", "", "PHP", 0, ""); err != nil || len(fe.rows) != before {
		t.Fatal("zero amount must be a no-op")
	}
}

func TestRecordChargeEffectsMissingPosting(t *testing.T) {
	ctx := context.Background()
	repos, fe := fixture()
	// version without mapping
	err := RecordChargeEffects(ctx, repos, "other", evIssue, "doc9", "", "PHP", 100, "")
	var ec interface{ ErrorCode() string }
	if !errors.As(err, &ec) || ec.ErrorCode() != "missing_posting" || !usecaseerr.IsCode(err, "missing_posting") {
		t.Fatalf("want missing_posting, got %v", err)
	}
	// version with only one of the two roles
	repos.ChargePolicyPosting.(*fakePostings).rows = []*postingpb.ChargePolicyPosting{posting("x", evIssue, rRecv, "a")}
	if err := RecordChargeEffects(ctx, repos, "v1", evIssue, "doc9", "", "PHP", 100, ""); !usecaseerr.IsCode(err, "missing_posting") {
		t.Fatalf("want missing_posting for half mapping, got %v", err)
	}
	if len(fe.rows) != 0 {
		t.Fatal("no rows may be written when the mapping is missing")
	}
}
