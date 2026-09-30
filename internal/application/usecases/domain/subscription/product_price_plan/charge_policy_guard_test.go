package product_price_plan

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	chargepolicypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	chargepolicyversionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	"google.golang.org/protobuf/proto"
)

type guardPolicyRepo struct {
	chargepolicypb.UnimplementedChargePolicyDomainServiceServer
	rows map[string]*chargepolicypb.ChargePolicy // only the caller's workspace: foreign ids are absent
}

func (r *guardPolicyRepo) ReadChargePolicy(_ context.Context, req *chargepolicypb.ReadChargePolicyRequest) (*chargepolicypb.ReadChargePolicyResponse, error) {
	p, ok := r.rows[req.Data.Id]
	if !ok {
		return nil, fmt.Errorf("record not found")
	}
	return &chargepolicypb.ReadChargePolicyResponse{Data: []*chargepolicypb.ChargePolicy{p}}, nil
}

type guardVersionRepo struct {
	chargepolicyversionpb.UnimplementedChargePolicyVersionDomainServiceServer
	rows []*chargepolicyversionpb.ChargePolicyVersion
}

func (r *guardVersionRepo) ListChargePolicyVersions(_ context.Context, req *chargepolicyversionpb.ListChargePolicyVersionsRequest) (*chargepolicyversionpb.ListChargePolicyVersionsResponse, error) {
	want := req.GetFilters().GetFilters()[0].GetStringFilter().GetValue()
	out := &chargepolicyversionpb.ListChargePolicyVersionsResponse{}
	for _, v := range r.rows {
		if v.GetChargePolicyId() == want {
			out.Data = append(out.Data, v)
		}
	}
	return out, nil
}

type guardPricePlanRepo struct {
	priceplanpb.UnimplementedPricePlanDomainServiceServer
	rows  map[string]*priceplanpb.PricePlan
	locks []string
}

// LockPricePlanForUpdate satisfies domainports.PricePlanLocker.
func (r *guardPricePlanRepo) LockPricePlanForUpdate(_ context.Context, id string) error {
	if _, ok := r.rows[id]; !ok {
		return fmt.Errorf("price_plan lock: %w", domainports.ErrLockedRowNotFound)
	}
	r.locks = append(r.locks, id)
	return nil
}

// unlockablePricePlanRepo hides the locker: the guard must refuse (fail closed) instead of
// reading unlocked.
type unlockablePricePlanRepo struct {
	priceplanpb.PricePlanDomainServiceServer
}

func (r *guardPricePlanRepo) ReadPricePlan(_ context.Context, req *priceplanpb.ReadPricePlanRequest) (*priceplanpb.ReadPricePlanResponse, error) {
	p, ok := r.rows[req.Data.Id]
	if !ok {
		return nil, fmt.Errorf("record not found")
	}
	return &priceplanpb.ReadPricePlanResponse{Data: []*priceplanpb.PricePlan{p}}, nil
}

type guardLineRepo struct {
	productpriceplanpb.UnimplementedProductPricePlanDomainServiceServer
	rows    map[string]*productpriceplanpb.ProductPricePlan
	created []*productpriceplanpb.ProductPricePlan
	updated []*productpriceplanpb.ProductPricePlan
}

func (r *guardLineRepo) CreateProductPricePlan(_ context.Context, req *productpriceplanpb.CreateProductPricePlanRequest) (*productpriceplanpb.CreateProductPricePlanResponse, error) {
	r.created = append(r.created, req.Data)
	return &productpriceplanpb.CreateProductPricePlanResponse{Data: []*productpriceplanpb.ProductPricePlan{req.Data}, Success: true}, nil
}
func (r *guardLineRepo) ReadProductPricePlan(_ context.Context, req *productpriceplanpb.ReadProductPricePlanRequest) (*productpriceplanpb.ReadProductPricePlanResponse, error) {
	l, ok := r.rows[req.Data.Id]
	if !ok {
		return nil, fmt.Errorf("record not found")
	}
	return &productpriceplanpb.ReadProductPricePlanResponse{Data: []*productpriceplanpb.ProductPricePlan{proto.Clone(l).(*productpriceplanpb.ProductPricePlan)}}, nil
}
func (r *guardLineRepo) UpdateProductPricePlan(_ context.Context, req *productpriceplanpb.UpdateProductPricePlanRequest) (*productpriceplanpb.UpdateProductPricePlanResponse, error) {
	r.updated = append(r.updated, req.Data)
	return &productpriceplanpb.UpdateProductPricePlanResponse{Data: []*productpriceplanpb.ProductPricePlan{req.Data}, Success: true}, nil
}

type allowAll struct{}

func (allowAll) IsEnabled() bool                                                   { return true }
func (allowAll) HasPermission(context.Context, string, string) (bool, error)       { return true, nil }
func (allowAll) HasGlobalPermission(context.Context, string, string) (bool, error) { return true, nil }

type guardIDs struct{}

func (guardIDs) GenerateID() string                   { return "ppp-new" }
func (guardIDs) GenerateIDWithPrefix(p string) string { return p + "-ppp-new" }
func (guardIDs) IsEnabled() bool                      { return true }
func (guardIDs) GetProviderInfo() string              { return "test" }

type guardEnv struct {
	create   *CreateProductPricePlanUseCase
	update   *UpdateProductPricePlanUseCase
	lines    *guardLineRepo
	policies *guardPolicyRepo
	versions *guardVersionRepo
	plans    *guardPricePlanRepo
}

func newGuardEnv(wired bool) *guardEnv {
	e := &guardEnv{
		lines: &guardLineRepo{rows: map[string]*productpriceplanpb.ProductPricePlan{}},
		policies: &guardPolicyRepo{rows: map[string]*chargepolicypb.ChargePolicy{
			"pol-ok":      {Id: "pol-ok", Status: enumspb.ChargePolicyStatus_CHARGE_POLICY_STATUS_ACTIVE},
			"pol-retired": {Id: "pol-retired", Status: enumspb.ChargePolicyStatus_CHARGE_POLICY_STATUS_RETIRED},
			"pol-draft":   {Id: "pol-draft", Status: enumspb.ChargePolicyStatus_CHARGE_POLICY_STATUS_ACTIVE},
			// "pol-foreign" belongs to another workspace: the workspace-scoped read returns not found
		}},
		versions: &guardVersionRepo{rows: []*chargepolicyversionpb.ChargePolicyVersion{
			{Id: "v-ok", ChargePolicyId: "pol-ok", Status: enumspb.ChargePolicyVersionStatus_CHARGE_POLICY_VERSION_STATUS_APPROVED},
			{Id: "v-retired", ChargePolicyId: "pol-retired", Status: enumspb.ChargePolicyVersionStatus_CHARGE_POLICY_VERSION_STATUS_APPROVED},
			{Id: "v-draft", ChargePolicyId: "pol-draft", Status: enumspb.ChargePolicyVersionStatus_CHARGE_POLICY_VERSION_STATUS_DRAFT},
		}},
		plans: &guardPricePlanRepo{rows: map[string]*priceplanpb.PricePlan{
			"pp-recurring": {Id: "pp-recurring", BillingKind: priceplanpb.BillingKind_BILLING_KIND_RECURRING},
			"pp-contract":  {Id: "pp-contract", BillingKind: priceplanpb.BillingKind_BILLING_KIND_CONTRACT},
			"pp-onetime":   {Id: "pp-onetime", BillingKind: priceplanpb.BillingKind_BILLING_KIND_ONE_TIME},
			"pp-milestone": {Id: "pp-milestone", BillingKind: priceplanpb.BillingKind_BILLING_KIND_MILESTONE},
			"pp-adhoc":     {Id: "pp-adhoc", BillingKind: priceplanpb.BillingKind_BILLING_KIND_AD_HOC},
			"pp-total":     {Id: "pp-total", BillingKind: priceplanpb.BillingKind_BILLING_KIND_RECURRING, AmountBasis: priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE},
		}},
	}
	repos := CreateProductPricePlanRepositories{ProductPricePlan: e.lines, PricePlan: e.plans}
	urepos := UpdateProductPricePlanRepositories{ProductPricePlan: e.lines, PricePlan: e.plans}
	if wired {
		repos.ChargePolicy, repos.ChargePolicyVersion = e.policies, e.versions
		urepos.ChargePolicy, urepos.ChargePolicyVersion = e.policies, e.versions
	}
	gate := actiongate.NewActionGatekeeper(allowAll{}, ports.NewNoOpTranslator())
	e.create = NewCreateProductPricePlanUseCase(repos, CreateProductPricePlanServices{Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), ActionGatekeeper: gate, IDGenerator: guardIDs{}})
	e.update = NewUpdateProductPricePlanUseCase(urepos, UpdateProductPricePlanServices{Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), ActionGatekeeper: gate})
	return e
}

func guardCtx() context.Context { return contextutil.WithUserID(context.Background(), "u1") }

func line(pricePlan string, tr productpriceplanpb.BillingTreatment, policy string, markup *int32) *productpriceplanpb.ProductPricePlan {
	l := &productpriceplanpb.ProductPricePlan{PricePlanId: pricePlan, ProductPlanId: "pp-x", BillingTreatment: tr, BillingCurrency: "PHP"}
	if policy != "" {
		l.ChargePolicyId = &policy
	}
	l.MarkupBps = markup
	return l
}

// AC-CP-06 / AC-UC-35 / AC-UC-11
func TestProductPricePlanChargePolicyOptInGuard(t *testing.T) {
	usage := productpriceplanpb.BillingTreatment_BILLING_TREATMENT_USAGE_BASED
	recurringT := productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING
	oneTimeT := productpriceplanpb.BillingTreatment_BILLING_TREATMENT_ONE_TIME_INITIAL
	five := int32(5)
	zero := int32(0)

	refusals := []struct {
		name string
		line *productpriceplanpb.ProductPricePlan
		want string
	}{
		{"billing_treatment RECURRING", line("pp-recurring", recurringT, "pol-ok", nil), guardCodeNotUsageBased},
		{"billing_treatment ONE_TIME_INITIAL", line("pp-recurring", oneTimeT, "pol-ok", nil), guardCodeNotUsageBased},
		{"price plan ONE_TIME", line("pp-onetime", usage, "pol-ok", nil), guardCodeBillingKind},
		{"price plan MILESTONE", line("pp-milestone", usage, "pol-ok", nil), guardCodeBillingKind},
		{"price plan AD_HOC", line("pp-adhoc", usage, "pol-ok", nil), guardCodeBillingKind},
		{"price plan TOTAL_PACKAGE", line("pp-total", usage, "pol-ok", nil), guardCodeBillingKind},
		{"retired policy", line("pp-recurring", usage, "pol-retired", nil), guardCodeUnavailable},
		{"unapproved policy (draft only)", line("pp-recurring", usage, "pol-draft", nil), guardCodeUnavailable},
		{"foreign/unknown policy", line("pp-recurring", usage, "pol-foreign", nil), guardCodeNotFound},
		{"markup_bps > 0 with policy", line("pp-recurring", usage, "pol-ok", &five), guardCodeMarkupNotAllowed},
		{"markup_bps > 0 without policy", line("pp-recurring", usage, "", &five), guardCodeMarkupNotAllowed},
	}
	for _, tc := range refusals {
		t.Run("create refuses "+tc.name, func(t *testing.T) {
			e := newGuardEnv(true)
			if _, err := e.create.Execute(guardCtx(), &productpriceplanpb.CreateProductPricePlanRequest{Data: tc.line}); !IsGuardCode(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
			if len(e.lines.created) != 0 {
				t.Fatal("a refused line must not be written")
			}
		})
	}

	t.Run("create accepts USAGE_BASED on RECURRING and CONTRACT with an approved active policy", func(t *testing.T) {
		for _, pp := range []string{"pp-recurring", "pp-contract"} {
			e := newGuardEnv(true)
			if _, err := e.create.Execute(guardCtx(), &productpriceplanpb.CreateProductPricePlanRequest{Data: line(pp, usage, "pol-ok", &zero)}); err != nil {
				t.Fatalf("%s: %v", pp, err)
			}
		}
	})
	t.Run("fails closed when the charge policy repositories are not wired", func(t *testing.T) {
		e := newGuardEnv(false)
		if _, err := e.create.Execute(guardCtx(), &productpriceplanpb.CreateProductPricePlanRequest{Data: line("pp-recurring", usage, "pol-ok", nil)}); !IsGuardCode(err, guardCodeUnverifiable) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("legacy lines without a policy are unaffected (even when unwired, any treatment/kind)", func(t *testing.T) {
		e := newGuardEnv(false)
		for _, l := range []*productpriceplanpb.ProductPricePlan{
			line("pp-onetime", oneTimeT, "", nil),
			line("pp-recurring", recurringT, "", &zero),
			line("pp-milestone", usage, "", nil),
		} {
			if _, err := e.create.Execute(guardCtx(), &productpriceplanpb.CreateProductPricePlanRequest{Data: l}); err != nil {
				t.Fatalf("legacy create refused: %v", err)
			}
		}
	})

	t.Run("update guards the effective merged line", func(t *testing.T) {
		e := newGuardEnv(true)
		pol := "pol-ok"
		e.lines.rows["bound"] = &productpriceplanpb.ProductPricePlan{Id: "bound", PricePlanId: "pp-recurring", ProductPlanId: "pp-x", BillingTreatment: usage, ChargePolicyId: &pol}
		e.lines.rows["legacy"] = &productpriceplanpb.ProductPricePlan{Id: "legacy", PricePlanId: "pp-recurring", ProductPlanId: "pp-x", BillingTreatment: recurringT}
		ctx := guardCtx()
		// changing billing_treatment of a policy-bound line away from USAGE_BASED clears the policy
		// in the same write (C12: derived write in the use case); an explicit keep is refused
		if _, err := e.update.Execute(ctx, &productpriceplanpb.UpdateProductPricePlanRequest{Data: &productpriceplanpb.ProductPricePlan{Id: "bound", BillingTreatment: recurringT, ChargePolicyId: &pol}}); !IsGuardCode(err, guardCodeNotUsageBased) {
			t.Fatalf("bound->recurring keeping the policy: %v", err)
		}
		// setting a policy on a legacy RECURRING line is refused (needs USAGE_BASED)
		if _, err := e.update.Execute(ctx, &productpriceplanpb.UpdateProductPricePlanRequest{Data: &productpriceplanpb.ProductPricePlan{Id: "legacy", ChargePolicyId: &pol}}); !IsGuardCode(err, guardCodeNotUsageBased) {
			t.Fatalf("legacy+policy: %v", err)
		}
		// opting in together with USAGE_BASED succeeds
		if _, err := e.update.Execute(ctx, &productpriceplanpb.UpdateProductPricePlanRequest{Data: &productpriceplanpb.ProductPricePlan{Id: "legacy", ChargePolicyId: &pol, BillingTreatment: usage}}); err != nil {
			t.Fatalf("opt-in: %v", err)
		}
		// pointing at a retired/foreign policy is refused
		bad := "pol-retired"
		if _, err := e.update.Execute(ctx, &productpriceplanpb.UpdateProductPricePlanRequest{Data: &productpriceplanpb.ProductPricePlan{Id: "bound", ChargePolicyId: &bad}}); !IsGuardCode(err, guardCodeUnavailable) {
			t.Fatalf("retired: %v", err)
		}
		// markup > 0 refused
		if _, err := e.update.Execute(ctx, &productpriceplanpb.UpdateProductPricePlanRequest{Data: &productpriceplanpb.ProductPricePlan{Id: "bound", MarkupBps: &five}}); !IsGuardCode(err, guardCodeMarkupNotAllowed) {
			t.Fatalf("markup: %v", err)
		}
		// an ordinary edit of a bound line (price change only) still passes
		if _, err := e.update.Execute(ctx, &productpriceplanpb.UpdateProductPricePlanRequest{Data: &productpriceplanpb.ProductPricePlan{Id: "bound", BillingAmount: 500}}); err != nil {
			t.Fatalf("plain edit: %v", err)
		}
		// a legacy line's ordinary edits are unaffected
		if _, err := e.update.Execute(ctx, &productpriceplanpb.UpdateProductPricePlanRequest{Data: &productpriceplanpb.ProductPricePlan{Id: "legacy", BillingTreatment: oneTimeT}}); err != nil {
			t.Fatalf("legacy edit: %v", err)
		}
	})
}

// Typed clear of a package line's charge policy (W2-E). AC-CP-06 follow-up.
func TestProductPricePlanClearChargePolicy(t *testing.T) {
	usage := productpriceplanpb.BillingTreatment_BILLING_TREATMENT_USAGE_BASED
	recurringT := productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING
	pol := "pol-ok"
	empty := ""
	seed := func(e *guardEnv) {
		e.lines.rows["bound"] = &productpriceplanpb.ProductPricePlan{Id: "bound", PricePlanId: "pp-recurring", ProductPlanId: "pp-x", BillingTreatment: usage, ChargePolicyId: &pol}
	}

	t.Run("clear forwards the typed intent (present and empty) to the adapter", func(t *testing.T) {
		e := newGuardEnv(true)
		seed(e)
		if _, err := e.update.Execute(guardCtx(), &productpriceplanpb.UpdateProductPricePlanRequest{Data: &productpriceplanpb.ProductPricePlan{Id: "bound", ChargePolicyId: &empty}}); err != nil {
			t.Fatalf("clear: %v", err)
		}
		if len(e.lines.updated) != 1 || e.lines.updated[0].ChargePolicyId == nil || e.lines.updated[0].GetChargePolicyId() != "" {
			t.Fatalf("adapter must receive present-and-empty charge_policy_id, got %+v", e.lines.updated)
		}
	})
	t.Run("clear works even when the policy repositories are not wired", func(t *testing.T) {
		e := newGuardEnv(false)
		seed(e)
		if _, err := e.update.Execute(guardCtx(), &productpriceplanpb.UpdateProductPricePlanRequest{Data: &productpriceplanpb.ProductPricePlan{Id: "bound", ChargePolicyId: &empty}}); err != nil {
			t.Fatalf("clear unwired: %v", err)
		}
	})
	t.Run("clear together with switching away from USAGE_BASED succeeds", func(t *testing.T) {
		e := newGuardEnv(true)
		seed(e)
		if _, err := e.update.Execute(guardCtx(), &productpriceplanpb.UpdateProductPricePlanRequest{Data: &productpriceplanpb.ProductPricePlan{Id: "bound", ChargePolicyId: &empty, BillingTreatment: recurringT}}); err != nil {
			t.Fatalf("clear+switch: %v", err)
		}
	})
	// C12: the derived write "switching off USAGE_BASED clears the policy" lives in the use case,
	// so it holds for every edit surface (the view no longer has to send the clear).
	t.Run("switch away without clearing clears the policy in the use case", func(t *testing.T) {
		e := newGuardEnv(true)
		seed(e)
		if _, err := e.update.Execute(guardCtx(), &productpriceplanpb.UpdateProductPricePlanRequest{Data: &productpriceplanpb.ProductPricePlan{Id: "bound", BillingTreatment: recurringT}}); err != nil {
			t.Fatalf("switch away: %v", err)
		}
		if len(e.lines.updated) != 1 || e.lines.updated[0].ChargePolicyId == nil || e.lines.updated[0].GetChargePolicyId() != "" {
			t.Fatalf("adapter must receive the derived present-and-empty clear, got %+v", e.lines.updated)
		}
	})
	t.Run("switch away while explicitly keeping a policy is refused", func(t *testing.T) {
		e := newGuardEnv(true)
		seed(e)
		keep := "pol-ok"
		if _, err := e.update.Execute(guardCtx(), &productpriceplanpb.UpdateProductPricePlanRequest{Data: &productpriceplanpb.ProductPricePlan{Id: "bound", BillingTreatment: recurringT, ChargePolicyId: &keep}}); !IsGuardCode(err, guardCodeNotUsageBased) {
			t.Fatalf("got %v", err)
		}
		if len(e.lines.updated) != 0 {
			t.Fatal("nothing may be written")
		}
	})
	t.Run("clear also drops a non-zero markup in the same request (effective line has none)", func(t *testing.T) {
		e := newGuardEnv(true)
		seed(e)
		five := int32(5)
		if _, err := e.update.Execute(guardCtx(), &productpriceplanpb.UpdateProductPricePlanRequest{Data: &productpriceplanpb.ProductPricePlan{Id: "bound", ChargePolicyId: &empty, MarkupBps: &five}}); err != nil {
			t.Fatalf("clear+markup: %v", err)
		}
	})
	t.Run("forged clear on a foreign-workspace or unknown line is not found", func(t *testing.T) {
		e := newGuardEnv(true) // "foreign" is not visible through the workspace-scoped repo
		_, err := e.update.Execute(guardCtx(), &productpriceplanpb.UpdateProductPricePlanRequest{Data: &productpriceplanpb.ProductPricePlan{Id: "foreign", ChargePolicyId: &empty}})
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("want not found, got %v", err)
		}
		if len(e.lines.updated) != 0 {
			t.Fatal("nothing may be written")
		}
	})
	t.Run("guard refusals expose stable codes", func(t *testing.T) {
		e := newGuardEnv(true)
		_, err := e.create.Execute(guardCtx(), &productpriceplanpb.CreateProductPricePlanRequest{Data: line("pp-recurring", usage, "pol-foreign", nil)})
		var ce interface{ ErrorCode() string }
		if !errors.As(fmt.Errorf("wrap: %w", err), &ce) || ce.ErrorCode() != "charge_policy_not_found" {
			t.Fatalf("code: %v (%v)", ce, err)
		}
	})
}

type errLineRepo struct {
	*guardLineRepo
	readErr error
}

func (r *errLineRepo) ReadProductPricePlan(context.Context, *productpriceplanpb.ReadProductPricePlanRequest) (*productpriceplanpb.ReadProductPricePlanResponse, error) {
	return nil, r.readErr
}

// C12: the guard fails closed on a read error (never skips, never writes).
func TestGuardFailsClosedOnReadErrors(t *testing.T) {
	usage := productpriceplanpb.BillingTreatment_BILLING_TREATMENT_USAGE_BASED
	pol := "pol-ok"
	t.Run("stored line read error refuses the update", func(t *testing.T) {
		e := newGuardEnv(true)
		e.update.repositories.ProductPricePlan = &errLineRepo{guardLineRepo: e.lines, readErr: errors.New("connection reset")}
		_, err := e.update.Execute(guardCtx(), &productpriceplanpb.UpdateProductPricePlanRequest{Data: &productpriceplanpb.ProductPricePlan{Id: "bound", ChargePolicyId: &pol, BillingTreatment: usage}})
		if err == nil || !strings.Contains(err.Error(), "connection reset") {
			t.Fatalf("want the repo error, got %v", err)
		}
		if len(e.lines.updated) != 0 {
			t.Fatal("nothing may be written")
		}
	})
	t.Run("policy read error is not reported as not_found", func(t *testing.T) {
		e := newGuardEnv(true)
		e.create.repositories.ChargePolicy = &errPolicyRepo{err: errors.New("deadline exceeded")}
		_, err := e.create.Execute(guardCtx(), &productpriceplanpb.CreateProductPricePlanRequest{Data: line("pp-recurring", usage, "pol-ok", nil)})
		if err == nil || IsGuardCode(err, guardCodeNotFound) || !strings.Contains(err.Error(), "deadline exceeded") {
			t.Fatalf("want the repo error surfaced as itself, got %v", err)
		}
	})
	t.Run("price plan read error fails closed with the repo error", func(t *testing.T) {
		e := newGuardEnv(true)
		e.create.repositories.PricePlan = &errPricePlanRepo{rows: e.plans.rows, readErr: errors.New("connection reset")}
		_, err := e.create.Execute(guardCtx(), &productpriceplanpb.CreateProductPricePlanRequest{Data: line("pp-recurring", usage, "pol-ok", nil)})
		if err == nil {
			t.Fatal("must refuse")
		}
	})
}

type errPolicyRepo struct {
	chargepolicypb.UnimplementedChargePolicyDomainServiceServer
	err error
}

func (r *errPolicyRepo) ReadChargePolicy(context.Context, *chargepolicypb.ReadChargePolicyRequest) (*chargepolicypb.ReadChargePolicyResponse, error) {
	return nil, r.err
}

type errPricePlanRepo struct {
	priceplanpb.UnimplementedPricePlanDomainServiceServer
	rows    map[string]*priceplanpb.PricePlan
	readErr error
}

func (r *errPricePlanRepo) ReadPricePlan(_ context.Context, req *priceplanpb.ReadPricePlanRequest) (*priceplanpb.ReadPricePlanResponse, error) {
	if req.Data.Id == "pp-recurring" {
		return nil, r.readErr
	}
	return &priceplanpb.ReadPricePlanResponse{Data: []*priceplanpb.PricePlan{r.rows[req.Data.Id]}}, nil
}

type guardXlator struct{}

func (guardXlator) Get(_ context.Context, _, key string, _ ...any) string { return "XL[" + key + "]" }
func (guardXlator) GetWithDefault(_ context.Context, _, key, _ string, _ ...any) string {
	return "XL[" + key + "]"
}

// C2: guard refusals are translated from product_price_plan.errors.<code>.
func TestGuardRefusalsAreTranslated(t *testing.T) {
	e := newGuardEnv(true)
	e.create.services.Translator = guardXlator{}
	_, err := e.create.Execute(guardCtx(), &productpriceplanpb.CreateProductPricePlanRequest{Data: line("pp-onetime", productpriceplanpb.BillingTreatment_BILLING_TREATMENT_USAGE_BASED, "pol-ok", nil)})
	if !IsGuardCode(err, guardCodeBillingKind) || !strings.Contains(err.Error(), "XL[product_price_plan.errors.charge_policy_billing_kind]") {
		t.Fatalf("got %v", err)
	}
}

// R4 m3: the opt-in guard takes the price plan lock and refuses when the repository cannot lock.
func TestChargePolicyGuardTakesPricePlanLockAndFailsClosedWithoutLocker(t *testing.T) {
	usage := productpriceplanpb.BillingTreatment_BILLING_TREATMENT_USAGE_BASED
	e := newGuardEnv(true)
	if _, err := e.create.Execute(guardCtx(), &productpriceplanpb.CreateProductPricePlanRequest{Data: line("pp-recurring", usage, "pol-ok", nil)}); err != nil {
		t.Fatalf("opt-in: %v", err)
	}
	if len(e.plans.locks) != 1 || e.plans.locks[0] != "pp-recurring" {
		t.Fatalf("price plan lock not taken: %v", e.plans.locks)
	}
	legacy := newGuardEnv(true)
	if _, err := legacy.create.Execute(guardCtx(), &productpriceplanpb.CreateProductPricePlanRequest{Data: line("pp-recurring", usage, "", nil)}); err != nil || len(legacy.plans.locks) != 0 {
		t.Fatalf("legacy line must not lock: err=%v locks=%v", err, legacy.plans.locks)
	}
	err := validateChargePolicyOptIn(guardCtx(), chargePolicyDeps{ChargePolicy: e.policies, ChargePolicyVersion: e.versions, PricePlan: unlockablePricePlanRepo{e.plans}},
		line("pp-recurring", usage, "pol-ok", nil), ports.NewNoOpTranslator())
	if !IsGuardCode(err, guardCodeUnverifiable) {
		t.Fatalf("want %s, got %v", guardCodeUnverifiable, err)
	}
}
