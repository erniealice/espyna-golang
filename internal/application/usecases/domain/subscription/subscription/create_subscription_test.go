package subscription

// Q-GSE-8 tests for require_spawn_success on CreateSubscription. Uses the same
// lightweight in-package mocks as subscription_plan_client_mismatch_test.go and
// materialize_jobs_for_subscription_test.go (no mock_db / mock_auth build tags).

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"

	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// fakeInstantiator implements both the base JobTemplateInstantiator port and the
// detailed extension, so the create path exercises InstantiateJobsFromPlanDetailed.
type fakeInstantiator struct {
	ids           []string
	skip          string
	err           error
	detailedCalls int
	baseCalls     int
}

func (f *fakeInstantiator) InstantiateJobsFromPlan(_ context.Context, _, _, _, _ string, _ bool) error {
	f.baseCalls++
	return f.err
}

func (f *fakeInstantiator) InstantiateJobsFromPlanDetailed(_ context.Context, _, _, _, _ string, _ bool) (jobSpawnOutcome, error) {
	f.detailedCalls++
	if f.err != nil {
		return jobSpawnOutcome{}, f.err
	}
	return jobSpawnOutcome{SpawnedJobIDs: f.ids, SkipReason: f.skip}, nil
}

func boolPtr(b bool) *bool { return &b }

// newCreateSubUCForSpawn wires a create use case with a Plan repo (so
// planDeclaresRootTemplate resolves) + a fake instantiator + a chosen Transactor.
// rootTemplateID "" => the plan declares no root template.
func newCreateSubUCForSpawn(t *testing.T, rootTemplateID string, inst JobTemplateInstantiator, txn ports.Transactor) *CreateSubscriptionUseCase {
	t.Helper()

	planID := "plan-1"
	plan := &planpb.Plan{Id: &planID}
	if rootTemplateID != "" {
		v := rootTemplateID
		plan.JobTemplateId = &v
	}
	planRepo := &stubPlanRepo{rows: map[string]*planpb.Plan{planID: plan}}

	ppRepo := &mockPricePlanRepoSub{pp: &priceplanpb.PricePlan{Id: "pp-1", Active: true, PlanId: planID}}
	clientRepo := &mockClientRepoSub{c: makeClient("client-cruz")}

	return NewCreateSubscriptionUseCase(
		CreateSubscriptionRepositories{
			Subscription: &mockSubRepo{},
			Client:       clientRepo,
			PricePlan:    ppRepo,
			Plan:         planRepo,
		},
		CreateSubscriptionServices{
			Authorizer:              ports.NewNoOpAuthorizer(),
			ActionGatekeeper:        actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
			Transactor:              txn,
			Translator:              ports.NewNoOpTranslator(),
			IDGenerator:             stubIDForSub{},
			JobTemplateInstantiator: inst,
		},
	)
}

func spawnReq(require bool) *subscriptionpb.CreateSubscriptionRequest {
	sub := newSubscription("New Engagement", "client-cruz", "pp-1")
	code := "SUB-CODE-1" // supply a code so generateSubscriptionCode is skipped
	sub.Code = &code
	return &subscriptionpb.CreateSubscriptionRequest{
		Data:                sub,
		RequireSpawnSuccess: boolPtr(require),
	}
}

// require=true + plan declares a root template + empty spawn => hard failure.
func TestCreateSubscription_RequireSpawn_EmptyResult_Errors(t *testing.T) {
	inst := &fakeInstantiator{ids: nil} // no jobs spawned
	uc := newCreateSubUCForSpawn(t, "tpl-root", inst, stubTxService{})

	_, err := uc.Execute(context.Background(), spawnReq(true))
	if err == nil {
		t.Fatalf("expected error when require_spawn_success and spawn produced no jobs")
	}
	if inst.detailedCalls != 1 {
		t.Errorf("strict path must use the detailed instantiator once, got %d", inst.detailedCalls)
	}
}

// require=true + plan declares a root template + spawn error => error propagates.
func TestCreateSubscription_RequireSpawn_SpawnError_Errors(t *testing.T) {
	inst := &fakeInstantiator{err: errors.New("materialize boom")}
	uc := newCreateSubUCForSpawn(t, "tpl-root", inst, stubTxService{})

	if _, err := uc.Execute(context.Background(), spawnReq(true)); err == nil {
		t.Fatalf("expected spawn error to fail the create under require_spawn_success")
	}
}

// require=true + no Transactor + empty spawn => still errors (now rejected
// pre-insert; see TestCreateSubscription_RequireSpawn_TemplateDeclared_NoTransactor_RejectsBeforeInsert
// for the fail-closed no-insert assertion).
func TestCreateSubscription_RequireSpawn_NoTransactor_EmptyResult_Errors(t *testing.T) {
	inst := &fakeInstantiator{ids: nil}
	uc := newCreateSubUCForSpawn(t, "tpl-root", inst, noTxnSub{})

	if _, err := uc.Execute(context.Background(), spawnReq(true)); err == nil {
		t.Fatalf("expected error even without a Transactor (best-effort fallback)")
	}
}

// errPlanRepo always fails the plan read, forcing the tri-state
// planDeclaresRootTemplate to report "cannot verify".
type errPlanRepo struct {
	planpb.UnimplementedPlanDomainServiceServer
}

func (errPlanRepo) ReadPlan(context.Context, *planpb.ReadPlanRequest) (*planpb.ReadPlanResponse, error) {
	return nil, errors.New("plan read boom")
}

// newCreateSubUCWith wires a create use case with explicit sub/plan repos so a
// test can assert whether the subscription create port was invoked.
func newCreateSubUCWith(subRepo *mockSubRepo, planRepo planpb.PlanDomainServiceServer, inst JobTemplateInstantiator, txn ports.Transactor) *CreateSubscriptionUseCase {
	return NewCreateSubscriptionUseCase(
		CreateSubscriptionRepositories{
			Subscription: subRepo,
			Client:       &mockClientRepoSub{c: makeClient("client-cruz")},
			PricePlan:    &mockPricePlanRepoSub{pp: &priceplanpb.PricePlan{Id: "pp-1", Active: true, PlanId: "plan-1"}},
			Plan:         planRepo,
		},
		CreateSubscriptionServices{
			Authorizer:              ports.NewNoOpAuthorizer(),
			ActionGatekeeper:        actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
			Transactor:              txn,
			Translator:              ports.NewNoOpTranslator(),
			IDGenerator:             stubIDForSub{},
			JobTemplateInstantiator: inst,
		},
	)
}

// (a) require=true + plan repo read error => fail closed: error AND no
// subscription inserted, no instantiator invoked.
func TestCreateSubscription_RequireSpawn_PlanReadError_FailsClosed_NoInsert(t *testing.T) {
	subRepo := &mockSubRepo{}
	inst := &fakeInstantiator{ids: []string{"job-1"}}
	uc := newCreateSubUCWith(subRepo, errPlanRepo{}, inst, stubTxService{})

	if _, err := uc.Execute(context.Background(), spawnReq(true)); err == nil {
		t.Fatalf("expected fail-closed error when the plan read fails under require_spawn_success")
	}
	if subRepo.createCalls != 0 {
		t.Errorf("no subscription must be created when the spawn requirement cannot be verified, got %d creates", subRepo.createCalls)
	}
	if inst.detailedCalls != 0 || inst.baseCalls != 0 {
		t.Errorf("instantiator must not run on fail-closed, got detailed=%d base=%d", inst.detailedCalls, inst.baseCalls)
	}
}

// (b) require=true + root template declared + NO usable Transactor => reject
// BEFORE inserting: error AND no subscription persisted, no instantiator invoked.
func TestCreateSubscription_RequireSpawn_TemplateDeclared_NoTransactor_RejectsBeforeInsert(t *testing.T) {
	subRepo := &mockSubRepo{}
	inst := &fakeInstantiator{ids: []string{"job-1"}}
	planID := "plan-1"
	tpl := "tpl-root"
	planRepo := &stubPlanRepo{rows: map[string]*planpb.Plan{planID: {Id: &planID, JobTemplateId: &tpl}}}
	// noTxnSub.SupportsTransactions() == false -> no usable Transactor.
	uc := newCreateSubUCWith(subRepo, planRepo, inst, noTxnSub{})

	if _, err := uc.Execute(context.Background(), spawnReq(true)); err == nil {
		t.Fatalf("expected rejection when a root template is declared but no usable Transactor is available")
	}
	if subRepo.createCalls != 0 {
		t.Errorf("strict mode with no usable Transactor must reject BEFORE inserting, got %d creates", subRepo.createCalls)
	}
	if inst.detailedCalls != 0 || inst.baseCalls != 0 {
		t.Errorf("instantiator must not run when the create is rejected pre-insert, got detailed=%d base=%d", inst.detailedCalls, inst.baseCalls)
	}
}

// require=true + plan declares NO root template => clean skip, subscription succeeds.
func TestCreateSubscription_RequireSpawn_NoTemplate_CleanSkip(t *testing.T) {
	inst := &fakeInstantiator{skip: SkipReasonNoTemplateFound}
	uc := newCreateSubUCForSpawn(t, "", inst, stubTxService{})

	resp, err := uc.Execute(context.Background(), spawnReq(true))
	if err != nil {
		t.Fatalf("no root template must be a clean skip, got error: %v", err)
	}
	if resp == nil {
		t.Fatalf("expected a response")
	}
	if resp.GetSpawnSkipReason() != SkipReasonNoTemplateFound {
		t.Errorf("want skip reason %q, got %q", SkipReasonNoTemplateFound, resp.GetSpawnSkipReason())
	}
}

// require=false (legacy default) => best-effort: an empty/failed spawn never
// fails the create.
func TestCreateSubscription_LegacyBestEffort_SpawnError_Succeeds(t *testing.T) {
	inst := &fakeInstantiator{err: errors.New("materialize boom")}
	uc := newCreateSubUCForSpawn(t, "tpl-root", inst, stubTxService{})

	resp, err := uc.Execute(context.Background(), spawnReq(false))
	if err != nil {
		t.Fatalf("legacy best-effort must not fail on spawn error, got: %v", err)
	}
	if resp == nil {
		t.Fatalf("expected a response")
	}
}

// spawned_job_ids populated on success (strict path).
func TestCreateSubscription_RequireSpawn_Success_PopulatesJobIDs(t *testing.T) {
	inst := &fakeInstantiator{ids: []string{"job-1", "job-2"}}
	uc := newCreateSubUCForSpawn(t, "tpl-root", inst, stubTxService{})

	resp, err := uc.Execute(context.Background(), spawnReq(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := resp.GetSpawnedJobIds()
	if len(got) != 2 || got[0] != "job-1" || got[1] != "job-2" {
		t.Errorf("want spawned_job_ids [job-1 job-2], got %v", got)
	}
}

// spawned_job_ids also populated additively on the legacy best-effort path.
func TestCreateSubscription_LegacyBestEffort_Success_PopulatesJobIDs(t *testing.T) {
	inst := &fakeInstantiator{ids: []string{"job-9"}}
	uc := newCreateSubUCForSpawn(t, "tpl-root", inst, stubTxService{})

	resp, err := uc.Execute(context.Background(), spawnReq(false))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := resp.GetSpawnedJobIds(); len(got) != 1 || got[0] != "job-9" {
		t.Errorf("want spawned_job_ids [job-9], got %v", got)
	}
	if inst.detailedCalls != 1 {
		t.Errorf("legacy path should prefer the detailed instantiator, got %d calls", inst.detailedCalls)
	}
}
