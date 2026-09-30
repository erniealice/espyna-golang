package subscription

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	chargepolicyuc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/ledger/charge_policy"
	chargepolicypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	chargepolicyversionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
	agreementlinetermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/agreement_line_term"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeLines struct {
	productpriceplanpb.UnimplementedProductPricePlanDomainServiceServer
	rows  []*productpriceplanpb.ProductPricePlan
	calls int
}

func (f *fakeLines) ListProductPricePlans(context.Context, *productpriceplanpb.ListProductPricePlansRequest) (*productpriceplanpb.ListProductPricePlansResponse, error) {
	f.calls++
	return &productpriceplanpb.ListProductPricePlansResponse{Data: f.rows, Success: true}, nil
}

type fakeTermRepo struct {
	agreementlinetermpb.UnimplementedAgreementLineTermDomainServiceServer
	created []*agreementlinetermpb.AgreementLineTerm
	failOn  int
}

func (f *fakeTermRepo) CreateAgreementLineTerm(_ context.Context, r *agreementlinetermpb.CreateAgreementLineTermRequest) (*agreementlinetermpb.CreateAgreementLineTermResponse, error) {
	if f.failOn > 0 && len(f.created)+1 == f.failOn {
		return nil, errors.New("term insert boom")
	}
	f.created = append(f.created, r.Data)
	return &agreementlinetermpb.CreateAgreementLineTermResponse{Success: true}, nil
}

type fakePolicies struct {
	chargepolicypb.UnimplementedChargePolicyDomainServiceServer
	rows map[string]*chargepolicypb.ChargePolicy
}

func (f *fakePolicies) ReadChargePolicy(_ context.Context, r *chargepolicypb.ReadChargePolicyRequest) (*chargepolicypb.ReadChargePolicyResponse, error) {
	if p, ok := f.rows[r.Data.Id]; ok {
		return &chargepolicypb.ReadChargePolicyResponse{Data: []*chargepolicypb.ChargePolicy{p}}, nil
	}
	return nil, errors.New("record not found")
}

type fakePolicyVersions struct {
	chargepolicyversionpb.UnimplementedChargePolicyVersionDomainServiceServer
	rows []*chargepolicyversionpb.ChargePolicyVersion
}

func (f *fakePolicyVersions) ListChargePolicyVersions(context.Context, *chargepolicyversionpb.ListChargePolicyVersionsRequest) (*chargepolicyversionpb.ListChargePolicyVersionsResponse, error) {
	return &chargepolicyversionpb.ListChargePolicyVersionsResponse{Data: f.rows, Success: true}, nil
}

type termFixture struct {
	uc       *CreateSubscriptionUseCase
	sub      *mockSubRepo
	lines    *fakeLines
	terms    *fakeTermRepo
	policies *fakePolicies
	versions *fakePolicyVersions
}

func newTermFixture(txn ports.Transactor, wired bool) *termFixture {
	f := &termFixture{
		sub:   &mockSubRepo{},
		lines: &fakeLines{},
		terms: &fakeTermRepo{},
		policies: &fakePolicies{rows: map[string]*chargepolicypb.ChargePolicy{
			"pol-1": {Id: "pol-1", Status: enumspb.ChargePolicyStatus_CHARGE_POLICY_STATUS_ACTIVE},
		}},
		versions: &fakePolicyVersions{rows: []*chargepolicyversionpb.ChargePolicyVersion{
			{Id: "ver-1", ChargePolicyId: "pol-1", VersionNumber: 1, Status: enumspb.ChargePolicyVersionStatus_CHARGE_POLICY_VERSION_STATUS_SUPERSEDED},
			{Id: "ver-2", ChargePolicyId: "pol-1", VersionNumber: 2, Status: enumspb.ChargePolicyVersionStatus_CHARGE_POLICY_VERSION_STATUS_APPROVED},
			{Id: "ver-3", ChargePolicyId: "pol-1", VersionNumber: 3, Status: enumspb.ChargePolicyVersionStatus_CHARGE_POLICY_VERSION_STATUS_DRAFT},
		}},
	}
	repos := CreateSubscriptionRepositories{
		Subscription: f.sub,
		Client:       &mockClientRepoSub{c: makeClient("client-cruz")},
		PricePlan:    &mockPricePlanRepoSub{pp: &priceplanpb.PricePlan{Id: "pp-1", Active: true, PlanId: "plan-1", BillingKind: priceplanpb.BillingKind_BILLING_KIND_RECURRING}},
		Plan:         &stubPlanRepo{rows: map[string]*planpb.Plan{}},
	}
	svc := CreateSubscriptionServices{
		Authorizer: ports.NewNoOpAuthorizer(), Transactor: txn, Translator: ports.NewNoOpTranslator(), IDGenerator: stubIDForSub{},
		ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
	}
	if wired {
		repos.ProductPricePlan, repos.AgreementLineTerm = f.lines, f.terms
		// The REAL charge_policy resolver (R1: subscription create resolves through ResolveChargePolicy,
		// it does not re-implement it), over the fake policy repositories.
		svc.ChargePolicyResolver = chargepolicyuc.NewResolveChargePolicyUseCase(
			chargepolicyuc.ResolveChargePolicyRepositories{ChargePolicy: f.policies, ChargePolicyVersion: f.versions},
			chargepolicyuc.ResolveChargePolicyServices{Authorizer: svc.Authorizer, Translator: svc.Translator, ActionGatekeeper: svc.ActionGatekeeper},
		)
	}
	f.uc = NewCreateSubscriptionUseCase(repos, svc)
	return f
}

func termReq() *subscriptionpb.CreateSubscriptionRequest {
	s := newSubscription("Unit 4A lease", "client-cruz", "pp-1")
	code := "SUB-T-1"
	s.Code = &code
	s.DateTimeStart = timestamppb.New(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))
	return &subscriptionpb.CreateSubscriptionRequest{Data: s}
}

func optedInLine(id, policy string, markup *int32) *productpriceplanpb.ProductPricePlan {
	return &productpriceplanpb.ProductPricePlan{Id: id, PricePlanId: "pp-1", Active: true, ChargePolicyId: &policy, MarkupBps: markup}
}

func TestSubscriptionCreatePinsAgreementLineTerms(t *testing.T) {
	f := newTermFixture(stubTxService{}, true)
	zero := int32(0)
	f.lines.rows = []*productpriceplanpb.ProductPricePlan{
		optedInLine("ppl-elec", "pol-1", &zero),
		{Id: "ppl-rent", PricePlanId: "pp-1", Active: true}, // not opted in: no term
		optedInLine("ppl-water", "pol-1", nil),
		{Id: "ppl-other-plan", PricePlanId: "pp-other", Active: true, ChargePolicyId: strPtr("pol-1")}, // defensive: foreign plan line ignored
	}
	if _, err := f.uc.Execute(withUser(), termReq()); err != nil {
		t.Fatal(err)
	}
	if f.sub.createCalls != 1 || len(f.terms.created) != 2 {
		t.Fatalf("subscription=%d terms=%d", f.sub.createCalls, len(f.terms.created))
	}
	for _, tm := range f.terms.created {
		if tm.SubscriptionId != "sub-new" || tm.ClientId != "client-cruz" || tm.ChargePolicyVersionId != "ver-2" || tm.EffectiveFrom != "2026-03-01" ||
			tm.EffectiveTo != nil || tm.Origin != agreementlinetermpb.AgreementLineTermOrigin_AGREEMENT_LINE_TERM_ORIGIN_COPIED || !tm.Active || tm.GetAcceptedAt() == 0 {
			t.Fatalf("term wrong (must pin the APPROVED version ver-2, not the draft): %+v", tm)
		}
	}
	if f.terms.created[0].ProductPricePlanId != "ppl-elec" || f.terms.created[0].MarkupBps == nil || *f.terms.created[0].MarkupBps != 0 || f.terms.created[1].ProductPricePlanId != "ppl-water" {
		t.Fatalf("lines wrong: %+v %+v", f.terms.created[0], f.terms.created[1])
	}
}

func TestSubscriptionCreateRefusesResolverFailure(t *testing.T) {
	cases := map[string]func(f *termFixture){
		"retired": func(f *termFixture) {
			f.policies.rows["pol-1"].Status = enumspb.ChargePolicyStatus_CHARGE_POLICY_STATUS_RETIRED
		},
		"no_approved_version": func(f *termFixture) { f.versions.rows = f.versions.rows[:1] },
		"not_found":           func(f *termFixture) { delete(f.policies.rows, "pol-1") },
	}
	for want, mut := range cases {
		f := newTermFixture(stubTxService{}, true)
		f.lines.rows = []*productpriceplanpb.ProductPricePlan{optedInLine("ppl-1", "pol-1", nil)}
		mut(f)
		_, err := f.uc.Execute(withUser(), termReq())
		var ce interface{ ErrorCode() string }
		if !errors.As(err, &ce) || ce.ErrorCode() != want {
			t.Fatalf("%s: got %v", want, err)
		}
		if f.sub.createCalls != 0 || len(f.terms.created) != 0 {
			t.Fatalf("%s: a resolver refusal must create nothing (sub=%d terms=%d)", want, f.sub.createCalls, len(f.terms.created))
		}
	}
	// opted-in lines but no transaction: refused before any insert
	f := newTermFixture(noTxnSub{}, true)
	f.lines.rows = []*productpriceplanpb.ProductPricePlan{optedInLine("ppl-1", "pol-1", nil)}
	if _, err := f.uc.Execute(withUser(), termReq()); !usecaseerr.IsCode(err, "transaction_required") || f.sub.createCalls != 0 {
		t.Fatalf("no-tx: %v calls=%d", err, f.sub.createCalls)
	}
	// a term insert failure rolls the subscription back with it (error returned from the tx body)
	f = newTermFixture(stubTxService{}, true)
	f.lines.rows = []*productpriceplanpb.ProductPricePlan{optedInLine("ppl-1", "pol-1", nil)}
	f.terms.failOn = 1
	if _, err := f.uc.Execute(withUser(), termReq()); err == nil {
		t.Fatal("a failed term insert must fail the create")
	}
}

// AC-UC-35 / PRES: without an opted-in line (or without the S1 collaborators) the term path writes
// nothing and the create result is identical.
func TestSubscriptionCreateUnchangedWithoutOptIn(t *testing.T) {
	plain := newTermFixture(stubTxService{}, false)
	respPlain, err := plain.uc.Execute(withUser(), termReq())
	if err != nil {
		t.Fatal(err)
	}
	f := newTermFixture(stubTxService{}, true)
	f.lines.rows = []*productpriceplanpb.ProductPricePlan{{Id: "ppl-rent", PricePlanId: "pp-1", Active: true}}
	resp, err := f.uc.Execute(withUser(), termReq())
	if err != nil {
		t.Fatal(err)
	}
	if len(f.terms.created) != 0 || f.sub.createCalls != 1 {
		t.Fatalf("no opt-in: terms=%d subs=%d", len(f.terms.created), f.sub.createCalls)
	}
	a, b := respPlain.Data[0], resp.Data[0]
	if a.GetName() != b.GetName() || a.GetCode() != b.GetCode() || a.GetClientId() != b.GetClientId() || a.GetPricePlanId() != b.GetPricePlanId() || a.GetId() != b.GetId() || !resp.GetSuccess() {
		t.Fatalf("create result differs: %+v vs %+v", a, b)
	}
	if plain.terms.created != nil || f.policies.rows == nil {
		t.Fatal("unwired path must not touch terms")
	}
}

func withUser() context.Context { return context.Background() }

func strPtr(s string) *string { return &s }

// R1: a price plan that cannot carry policy lines (ONE_TIME/AD_HOC/MILESTONE/TOTAL_PACKAGE) never
// triggers the package-line read, so a failing list cannot fail those creates.
func TestSubscriptionCreateDoesNotReadLinesForPlansWithoutPolicyLines(t *testing.T) {
	f := newTermFixture(stubTxService{}, true)
	f.uc.repositories.PricePlan = &mockPricePlanRepoSub{pp: &priceplanpb.PricePlan{Id: "pp-1", Active: true, PlanId: "plan-1", BillingKind: priceplanpb.BillingKind_BILLING_KIND_ONE_TIME}}
	f.lines.rows = []*productpriceplanpb.ProductPricePlan{optedInLine("ppl-1", "pol-1", nil)}
	if _, err := f.uc.Execute(withUser(), termReq()); err != nil {
		t.Fatal(err)
	}
	if f.lines.calls != 0 || len(f.terms.created) != 0 {
		t.Fatalf("no line read / term expected for a ONE_TIME plan (calls=%d terms=%d)", f.lines.calls, len(f.terms.created))
	}
}

// Opted-in lines but no resolver wired: refused before any write (fail closed).
func TestSubscriptionCreateRefusesUnverifiableWithoutResolver(t *testing.T) {
	f := newTermFixture(stubTxService{}, true)
	f.uc.services.ChargePolicyResolver = nil
	f.lines.rows = []*productpriceplanpb.ProductPricePlan{optedInLine("ppl-1", "pol-1", nil)}
	if _, err := f.uc.Execute(withUser(), termReq()); !usecaseerr.IsCode(err, "charge_policy_unverifiable") || f.sub.createCalls != 0 {
		t.Fatalf("got %v calls=%d", err, f.sub.createCalls)
	}
}

// R4 m9: an actor with subscription:create but WITHOUT charge_policy:read can create a subscription
// on an opted-in plan: create resolves the policy through the ungated helper under its own gate,
// and the gated Execute of the same use case still refuses that actor.
func TestSubscriptionCreateDoesNotInheritChargePolicyReadGate(t *testing.T) {
	f := newTermFixture(stubTxService{}, true)
	authz := createOnlyAuthorizer() // subscription:create only; charge_policy:read is absent
	gate := actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator())
	f.uc.services.ActionGatekeeper = gate
	resolver := chargepolicyuc.NewResolveChargePolicyUseCase(
		chargepolicyuc.ResolveChargePolicyRepositories{ChargePolicy: f.policies, ChargePolicyVersion: f.versions},
		chargepolicyuc.ResolveChargePolicyServices{Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), ActionGatekeeper: gate},
	)
	f.uc.services.ChargePolicyResolver = resolver
	f.lines.rows = []*productpriceplanpb.ProductPricePlan{optedInLine("ppl-1", "pol-1", nil)}

	if _, err := resolver.Execute(registrarCtx(), &chargepolicypb.ResolveChargePolicyRequest{PackageLineChargePolicyId: strPtr("pol-1")}); err == nil {
		t.Fatal("the gated ResolveChargePolicy.Execute must deny an actor without charge_policy:read")
	}
	if _, err := f.uc.Execute(registrarCtx(), termReq()); err != nil {
		t.Fatalf("create with subscription:create only must succeed on an opted-in plan: %v", err)
	}
	if f.sub.createCalls != 1 || len(f.terms.created) != 1 || f.terms.created[0].GetChargePolicyVersionId() != "ver-2" {
		t.Fatalf("subscription=%d terms=%v", f.sub.createCalls, f.terms.created)
	}
}
