package subscription

// Item #4b — enforce-style authz test for the item #5 (option a) split.
//
// Before item #5, one operator action ("create enrollment") traversed TWO verbs:
// subscription:create at the CreateSubscription front door, then
// subscription:update on the intrinsic materialize side effect. Under
// AUTHZ_ENFORCE=true a least-privilege registrar role holding subscription:create
// but NOT subscription:update would hit the second gate mid-flow — silently
// spawning zero jobs (legacy path) or rolling the enrollment back (strict path).
//
// Item #5 (option a) split an ungated materializeCore out of the gated public
// Execute: the create side effect calls the ungated core (its subscription:create
// ancestor already authorized it), while the manual/direct proto boundary keeps
// its subscription:update gate. These tests are the enforce-style proof of that
// split — they wire the REAL Create -> Materialize chain (not the isolating
// fakeInstantiator) with an authorizer that ALLOWS subscription:create and DENIES
// subscription:update, and assert:
//
//   1. the create-only principal STILL gets jobs spawned via the ungated core
//      (both the legacy best-effort and the strict require_spawn_success paths), and
//   2. the DIRECT/manual materialize command (public Execute) is STILL denied.
//
// If the create side effect is ever re-pointed back at the gated public Execute
// (undoing item #5), tests (1) fail — which is the point of an enforce-style test.

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"

	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// splitAuthorizer is a partial-deny RBAC port mirroring the omnisearch
// fakeAuthorizer (search_entities_test.go). IsEnabled() is true so the gate
// actually consults HasPermission; allowed holds the permission codes the
// principal carries (here: only subscription:create).
type splitAuthorizer struct{ allowed map[string]bool }

func (f *splitAuthorizer) IsEnabled() bool { return true }
func (f *splitAuthorizer) HasPermission(_ context.Context, _ string, permission string) (bool, error) {
	return f.allowed[permission], nil
}

// createOnlyAuthorizer grants subscription:create and (by omission) DENIES
// subscription:update — the exact least-privilege registrar posture item #5 is
// about.
func createOnlyAuthorizer() *splitAuthorizer {
	return &splitAuthorizer{allowed: map[string]bool{
		entityid.EntityPermission(entityid.Subscription, entityid.ActionCreate): true,
		// entityid.Subscription:update intentionally absent -> denied.
	}}
}

// registrarCtx carries a user id — ActionGatekeeper.Check requires one before it
// consults HasPermission (an empty user id fails on a different "authorization
// failed" path, which would make the test ambiguous).
func registrarCtx() context.Context {
	return contextutil.WithUserID(context.Background(), "registrar-1")
}

// newSplitAuthzChain wires the REAL Create -> Materialize chain, with BOTH
// gatekeepers backed by the SAME authorizer (mirroring production composition
// where one Authorizer is shared). The materialize UC is given a full stub repo
// set that spawns exactly one root Job for subscription "sub-new" (the id
// stubIDForSub mints for the created subscription).
func newSplitAuthzChain(t *testing.T, authz actiongate.Authorizer) (*CreateSubscriptionUseCase, *stubJobRepo) {
	t.Helper()

	const (
		subID       = "sub-new" // stubIDForSub.GenerateID()
		planID      = "plan-1"
		pricePlanID = "pp-1"
		rootTplID   = "tpl-root"
	)

	// --- Materialize UC repos: must resolve sub-new -> pp-1 -> plan-1 -> tpl-root
	//     and spawn one root Job. ---
	matSubRepo := &stubSubscriptionRepo{rows: map[string]*subscriptionpb.Subscription{
		subID: {Id: subID, Active: true, ClientId: "client-cruz", PricePlanId: pricePlanID, Name: "New Engagement"},
	}}
	matPPRepo := &stubPricePlanRepo{rows: map[string]*priceplanpb.PricePlan{
		pricePlanID: {Id: pricePlanID, Active: true, PlanId: planID},
	}}
	tpl := rootTplID
	matPlanRepo := &stubPlanRepo{rows: map[string]*planpb.Plan{
		planID: {Id: &[]string{planID}[0], JobTemplateId: &tpl},
	}}
	tplRepo := &stubJobTemplateRepo{rows: map[string]*jobtemplatepb.JobTemplate{
		rootTplID: makeTemplate(rootTplID, "Root", true),
	}}
	phaseRepo := &stubJobTemplatePhaseRepo{byTemplate: map[string][]*jobtemplatephasepb.JobTemplatePhase{
		rootTplID: {makePhase("p1", rootTplID, "Semester 1", 1, "")},
	}}
	taskRepo := &stubJobTemplateTaskRepo{byPhase: nil}
	jobRepo := &stubJobRepo{}

	materializeUC := NewMaterializeJobsForSubscriptionUseCase(
		MaterializeJobsForSubscriptionRepositories{
			Subscription:     matSubRepo,
			PricePlan:        matPPRepo,
			Plan:             matPlanRepo,
			JobTemplate:      tplRepo,
			JobTemplatePhase: phaseRepo,
			JobTemplateTask:  taskRepo,
			Job:              jobRepo,
			JobPhase:         &stubJobPhaseRepo{},
			JobTask:          &stubJobTaskRepo{},
		},
		MaterializeJobsForSubscriptionServices{
			ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
			Transactor:       stubTxService{},
			Translator:       ports.NewNoOpTranslator(),
			IDGenerator:      ports.NewNoOpIDGenerator(),
		},
	)

	inst := &MaterializeJobsForSubscriptionInstantiator{UseCase: materializeUC}

	// --- Create UC repos. Plan repo lets planDeclaresRootTemplate resolve for the
	//     strict path. ---
	createPlanRepo := &stubPlanRepo{rows: map[string]*planpb.Plan{
		planID: {Id: &[]string{planID}[0], JobTemplateId: &tpl},
	}}
	createUC := NewCreateSubscriptionUseCase(
		CreateSubscriptionRepositories{
			Subscription: &mockSubRepo{},
			Client:       &mockClientRepoSub{c: makeClient("client-cruz")},
			PricePlan:    &mockPricePlanRepoSub{pp: &priceplanpb.PricePlan{Id: pricePlanID, Active: true, PlanId: planID}},
			Plan:         createPlanRepo,
		},
		CreateSubscriptionServices{
			ActionGatekeeper:        actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
			Transactor:              stubTxService{},
			Translator:              ports.NewNoOpTranslator(),
			IDGenerator:             stubIDForSub{},
			JobTemplateInstantiator: inst,
		},
	)
	return createUC, jobRepo
}

func createReqCoded(require bool) *subscriptionpb.CreateSubscriptionRequest {
	sub := newSubscription("New Engagement", "client-cruz", "pp-1")
	code := "SUB-CODE-1" // supply a code so generateSubscriptionCode is skipped
	sub.Code = &code
	return &subscriptionpb.CreateSubscriptionRequest{
		Data:                sub,
		RequireSpawnSuccess: boolPtr(require),
	}
}

// TestCreateSubscription_CreateOnly_LegacyPath_SpawnsViaUngatedCore proves the
// item #5 split on the LEGACY best-effort path: a principal with
// subscription:create ALLOWED but subscription:update DENIED still spawns jobs,
// because the create side effect calls the ungated materializeCore rather than
// the subscription:update-gated public Execute.
func TestCreateSubscription_CreateOnly_LegacyPath_SpawnsViaUngatedCore(t *testing.T) {
	createUC, jobRepo := newSplitAuthzChain(t, createOnlyAuthorizer())

	resp, err := createUC.Execute(registrarCtx(), createReqCoded(false))
	if err != nil {
		t.Fatalf("create-only principal must not be denied on the create front door (subscription:create is granted): %v", err)
	}
	if resp == nil {
		t.Fatalf("expected a response")
	}
	if got := len(resp.GetSpawnedJobIds()); got != 1 {
		t.Errorf("item #5: create-only principal must still spawn jobs via the ungated core; want 1 spawned job id, got %d (%v)", got, resp.GetSpawnedJobIds())
	}
	if got := len(jobRepo.created); got != 1 {
		t.Errorf("materialize repo must have created the root Job (proves the ungated core actually ran); want 1 CreateJob, got %d", got)
	}
}

// TestCreateSubscription_CreateOnly_StrictPath_SpawnsViaUngatedCore proves the
// same split on the STRICT require_spawn_success path: create + spawn are atomic,
// yet the create-only principal succeeds (no rollback) because the ungated core
// runs. Before item #5 this path rolled the enrollment back under enforce.
func TestCreateSubscription_CreateOnly_StrictPath_SpawnsViaUngatedCore(t *testing.T) {
	createUC, jobRepo := newSplitAuthzChain(t, createOnlyAuthorizer())

	resp, err := createUC.Execute(registrarCtx(), createReqCoded(true))
	if err != nil {
		t.Fatalf("item #5: strict path must NOT roll back a legitimate create-only enrollment; got error: %v", err)
	}
	if resp == nil {
		t.Fatalf("expected a response")
	}
	if got := len(resp.GetSpawnedJobIds()); got != 1 {
		t.Errorf("strict path: want 1 spawned job id, got %d (%v)", got, resp.GetSpawnedJobIds())
	}
	if got := len(jobRepo.created); got != 1 {
		t.Errorf("strict path: materialize repo must have created the root Job; want 1, got %d", got)
	}
}

// TestMaterializeForSubscription_DirectExecute_CreateOnly_IsDenied proves the
// OTHER half of the split is intact: the DIRECT/manual materialize command
// (public Execute, reached by the "Spawn Jobs" drawer and any API/route caller)
// STILL gates subscription:update, so the same create-only principal is denied
// here — the ungated core did not open a bypass door on the direct boundary.
func TestMaterializeForSubscription_DirectExecute_CreateOnly_IsDenied(t *testing.T) {
	authz := createOnlyAuthorizer()

	// Build the materialize UC standalone with the same partial-deny authorizer.
	// Repos need not resolve — the subscription:update gate fires BEFORE any read.
	jobRepo := &stubJobRepo{}
	materializeUC := NewMaterializeJobsForSubscriptionUseCase(
		MaterializeJobsForSubscriptionRepositories{
			Subscription:     &stubSubscriptionRepo{rows: map[string]*subscriptionpb.Subscription{}},
			PricePlan:        &stubPricePlanRepo{rows: map[string]*priceplanpb.PricePlan{}},
			Plan:             &stubPlanRepo{rows: map[string]*planpb.Plan{}},
			JobTemplate:      &stubJobTemplateRepo{rows: map[string]*jobtemplatepb.JobTemplate{}},
			JobTemplatePhase: &stubJobTemplatePhaseRepo{byTemplate: map[string][]*jobtemplatephasepb.JobTemplatePhase{}},
			JobTemplateTask:  &stubJobTemplateTaskRepo{byPhase: nil},
			Job:              jobRepo,
			JobPhase:         &stubJobPhaseRepo{},
			JobTask:          &stubJobTaskRepo{},
		},
		MaterializeJobsForSubscriptionServices{
			ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
			Transactor:       stubTxService{},
			Translator:       ports.NewNoOpTranslator(),
			IDGenerator:      ports.NewNoOpIDGenerator(),
		},
	)

	_, err := materializeUC.Execute(registrarCtx(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{
		SubscriptionId: "sub-new",
		SpawnJobs:      true,
	})
	if err == nil {
		t.Fatalf("direct/manual materialize Execute must STILL be denied for a create-only principal (subscription:update gate)")
	}
	// NoOpTranslator returns the default message verbatim; actiongate's deny branch
	// uses "Permission denied".
	if !strings.Contains(err.Error(), "Permission denied") {
		t.Errorf("want a permission-denied error from the subscription:update gate, got %q", err.Error())
	}
	if got := len(jobRepo.created); got != 0 {
		t.Errorf("denied direct Execute must not spawn any Job; got %d CreateJob calls", got)
	}
}
