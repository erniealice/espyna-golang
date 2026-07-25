package subscription_group_product_plan

// ACTIVE->EXCLUDED in-use guard on the class UPDATE path — the go-unit half of
// coverage-audit gap A-G1 (docs/plan/20260724-section-assignment-merged/
// coverage-audit-20260725.md).
//
// plan.md §2 requires ONE rule shared by both write paths: "Delete/Exclude
// guards: refuse when active assignments exist". Delete has owned that rule
// under test since delete_subscription_group_product_plan_test.go; Exclude had
// no *_test.go at all, so update_subscription_group_product_plan.go:74-81 —
// isExcludeTransition + guardNotInUse — was reachable only through a live-DB
// browser case (C1 Case 5c). Deleting the whole `if isExcludeTransition(...)`
// block failed nothing.
//
// These cases mirror the Delete file exactly (same stubDeleteChecker, same
// "the write must not run" assertion) and add the direction Delete cannot
// express: the guard must fire ONLY on the ACTIVE->EXCLUDED transition. A guard
// that fired on every update of an in-use class would make the 372 live in-use
// classes uneditable, which is why the not-a-transition cases below are part of
// the contract, not filler.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	appcontext "github.com/erniealice/espyna-golang/internal/application/shared/context"
	espynaports "github.com/erniealice/espyna-golang/ports"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan"
)

const (
	classStatusActive   = pb.SubscriptionGroupProductPlanStatus_SUBSCRIPTION_GROUP_PRODUCT_PLAN_STATUS_ACTIVE
	classStatusExcluded = pb.SubscriptionGroupProductPlanStatus_SUBSCRIPTION_GROUP_PRODUCT_PLAN_STATUS_EXCLUDED
)

// ----- mocks ------------------------------------------------------------------

// excludeClassRepo serves the persisted class to effectiveClass's Read and to
// checkDuplicateClass's List, and counts the write. The Update use case reads
// and lists before it writes, so "the guard blocked the write" has to be
// asserted on updateCalls specifically — a total-call assertion would be
// satisfied by the read alone and would pass for the wrong reason.
type excludeClassRepo struct {
	pb.UnimplementedSubscriptionGroupProductPlanDomainServiceServer
	row         *pb.SubscriptionGroupProductPlan
	updateCalls int
	updated     *pb.SubscriptionGroupProductPlan
}

func (m *excludeClassRepo) ReadSubscriptionGroupProductPlan(_ context.Context, _ *pb.ReadSubscriptionGroupProductPlanRequest) (*pb.ReadSubscriptionGroupProductPlanResponse, error) {
	if m.row == nil {
		return &pb.ReadSubscriptionGroupProductPlanResponse{Success: true}, nil
	}
	return &pb.ReadSubscriptionGroupProductPlanResponse{Success: true, Data: []*pb.SubscriptionGroupProductPlan{m.row}}, nil
}

// ListSubscriptionGroupProductPlans returns the row under update as the only
// class on the section. checkDuplicateClass skips it via excludeID, so this
// deliberately proves the update does not read as its own duplicate.
func (m *excludeClassRepo) ListSubscriptionGroupProductPlans(_ context.Context, _ *pb.ListSubscriptionGroupProductPlansRequest) (*pb.ListSubscriptionGroupProductPlansResponse, error) {
	if m.row == nil {
		return &pb.ListSubscriptionGroupProductPlansResponse{Success: true}, nil
	}
	return &pb.ListSubscriptionGroupProductPlansResponse{Success: true, Data: []*pb.SubscriptionGroupProductPlan{m.row}}, nil
}

func (m *excludeClassRepo) UpdateSubscriptionGroupProductPlan(_ context.Context, req *pb.UpdateSubscriptionGroupProductPlanRequest) (*pb.UpdateSubscriptionGroupProductPlanResponse, error) {
	m.updateCalls++
	m.updated = req.GetData()
	return &pb.UpdateSubscriptionGroupProductPlanResponse{Success: true}, nil
}

// ----- fixture ----------------------------------------------------------------

type excludeFixture struct {
	repo    *excludeClassRepo
	checker *stubDeleteChecker
}

// newExcludeFixture builds a class whose cross-domain graph is fully VALID
// (sg-1 on plan-1 in ws-1 / pp-1 on plan-1 delivering product-1 / tmpl-1
// outputting product-1), so validateClassInvariants and checkDuplicateClass
// both pass. Anything these tests observe therefore comes from the in-use guard
// and nothing else.
func newExcludeFixture(persistedStatus pb.SubscriptionGroupProductPlanStatus, checker *stubDeleteChecker) *excludeFixture {
	return &excludeFixture{
		repo: &excludeClassRepo{row: &pb.SubscriptionGroupProductPlan{
			Id:                  "class-1",
			Active:              true,
			WorkspaceId:         "ws-1",
			SubscriptionGroupId: "sg-1",
			ProductPlanId:       "pp-1",
			JobTemplateId:       "tmpl-1",
			Status:              persistedStatus,
		}},
		checker: checker,
	}
}

func (f *excludeFixture) updateUC() *UpdateSubscriptionGroupProductPlanUseCase {
	// A typed-nil *stubDeleteChecker stored in an interface is NOT nil, which
	// would defeat guardNotInUse's nil-checker branch — assign only when set.
	var c espynaports.Checker
	if f.checker != nil {
		c = f.checker
	}
	return NewUpdateSubscriptionGroupProductPlanUseCase(
		UpdateSubscriptionGroupProductPlanRepositories{
			SubscriptionGroupProductPlan: f.repo,
			ProductPlan:                  &mockPlanRepo{byID: map[string]*productplanpb.ProductPlan{"pp-1": classPlan("pp-1", "plan-1", "product-1")}},
			SubscriptionGroup:            &mockGroupRepo{group: classGroup("plan-1", "ws-1")},
			JobTemplate:                  &mockJobTemplateRepo{byID: map[string]*jobtemplatepb.JobTemplate{"tmpl-1": classTemplate("tmpl-1", "product-1")}},
		},
		UpdateSubscriptionGroupProductPlanServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Translator:       ports.NewNoOpTranslator(),
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
			ReferenceChecker: c,
		},
	)
}

// excludeReq is the S1 "Exclude" / S4 status-change body: id + the target
// status only. Every other field is omitted, which is exactly how the SetStatus
// route posts it and is what makes effectiveClass's merge load-bearing.
func excludeReq(target pb.SubscriptionGroupProductPlanStatus) *pb.UpdateSubscriptionGroupProductPlanRequest {
	return &pb.UpdateSubscriptionGroupProductPlanRequest{
		Data: &pb.SubscriptionGroupProductPlan{Id: "class-1", Status: target},
	}
}

func excludeCtx() context.Context {
	return appcontext.WithWorkspaceID(context.Background(), "ws-1")
}

// ----- tests ------------------------------------------------------------------

// TestUpdateClass_ExcludeTransition_InUse_Rejects is the gap A-G1 case: an
// ACTIVE class carrying active assignments must REFUSE the move to EXCLUDED,
// with the same error the Delete guard raises, and the persisted row must be
// left untouched.
func TestUpdateClass_ExcludeTransition_InUse_Rejects(t *testing.T) {
	f := newExcludeFixture(classStatusActive, &stubDeleteChecker{inUse: map[string]bool{"class-1": true}})

	_, err := f.updateUC().Execute(excludeCtx(), excludeReq(classStatusExcluded))
	if err == nil {
		t.Fatalf("expected rejection: ACTIVE->EXCLUDED while active assignments reference the class (plan.md §2)")
	}
	if !strings.Contains(err.Error(), "active assignments") {
		t.Errorf("expected the shared in-use error, got %q", err.Error())
	}
	if f.repo.updateCalls != 0 {
		t.Errorf("update must not run on rejection, got %d calls", f.repo.updateCalls)
	}
}

// TestUpdateClass_ExcludeTransition_NotInUse_Succeeds is the other half: with no
// active assignments the exclude goes through and actually writes EXCLUDED.
// Without this, a guard that rejected unconditionally would pass the test above.
func TestUpdateClass_ExcludeTransition_NotInUse_Succeeds(t *testing.T) {
	f := newExcludeFixture(classStatusActive, &stubDeleteChecker{inUse: map[string]bool{}})

	if _, err := f.updateUC().Execute(excludeCtx(), excludeReq(classStatusExcluded)); err != nil {
		t.Fatalf("unexpected error excluding a class with no active assignments: %v", err)
	}
	if f.repo.updateCalls != 1 {
		t.Fatalf("expected 1 update call, got %d", f.repo.updateCalls)
	}
	if got := f.repo.updated.GetStatus(); got != classStatusExcluded {
		t.Errorf("expected the persisted status to be EXCLUDED, got %v", got)
	}
}

// TestUpdateClass_NonExcludeUpdate_InUse_NotGuarded pins the guard's PRECISION.
// The class is in use, but the body omits status (the SetStatus-less edit the S4
// Info tab posts), so effectiveClass inherits ACTIVE and this is not an exclude
// transition. Blocking it would freeze every one of the 372 live in-use classes
// against ordinary edits.
func TestUpdateClass_NonExcludeUpdate_InUse_NotGuarded(t *testing.T) {
	f := newExcludeFixture(classStatusActive, &stubDeleteChecker{inUse: map[string]bool{"class-1": true}})

	// Status left UNSPECIFIED == omitted by the caller.
	req := excludeReq(pb.SubscriptionGroupProductPlanStatus_SUBSCRIPTION_GROUP_PRODUCT_PLAN_STATUS_UNSPECIFIED)
	if _, err := f.updateUC().Execute(excludeCtx(), req); err != nil {
		t.Fatalf("a non-exclude update of an in-use class must not be guarded: %v", err)
	}
	if f.repo.updateCalls != 1 {
		t.Fatalf("expected 1 update call, got %d", f.repo.updateCalls)
	}
}

// TestUpdateClass_AlreadyExcluded_InUse_NotATransition covers isExcludeTransition's
// "existing != EXCLUDED" leg. Re-saving a class that is ALREADY excluded is not a
// transition into the guarded state, so an in-use excluded class stays editable
// rather than becoming permanently read-only.
func TestUpdateClass_AlreadyExcluded_InUse_NotATransition(t *testing.T) {
	f := newExcludeFixture(classStatusExcluded, &stubDeleteChecker{inUse: map[string]bool{"class-1": true}})

	if _, err := f.updateUC().Execute(excludeCtx(), excludeReq(classStatusExcluded)); err != nil {
		t.Fatalf("re-saving an already-EXCLUDED class is not an ACTIVE->EXCLUDED transition: %v", err)
	}
	if f.repo.updateCalls != 1 {
		t.Fatalf("expected 1 update call, got %d", f.repo.updateCalls)
	}
}

// TestUpdateClass_ExcludeTransition_CheckerError_FailsClosed: guardNotInUse
// returns the checker's error verbatim, so a reference-check failure must abort
// the exclude rather than fall through to the write. An error swallowed here
// would silently un-guard the transition exactly when the guard cannot answer.
func TestUpdateClass_ExcludeTransition_CheckerError_FailsClosed(t *testing.T) {
	f := newExcludeFixture(classStatusActive, &stubDeleteChecker{err: errors.New("reference check unavailable")})

	_, err := f.updateUC().Execute(excludeCtx(), excludeReq(classStatusExcluded))
	if err == nil {
		t.Fatalf("expected the exclude to fail closed when the reference checker errors")
	}
	if f.repo.updateCalls != 0 {
		t.Errorf("update must not run when the guard could not be evaluated, got %d calls", f.repo.updateCalls)
	}
}

// TestUpdateClass_NilChecker_DegradesUnguarded mirrors
// TestDeleteClass_NilChecker_DegradesUnguarded: a composition gap (no
// ReferenceChecker wired) must degrade to un-guarded rather than brick every
// exclude. Documented tolerance in in_use_guard.go, pinned here so a future
// fail-closed decision is a deliberate change to this test, not a silent one.
func TestUpdateClass_NilChecker_DegradesUnguarded(t *testing.T) {
	f := newExcludeFixture(classStatusActive, nil)

	if _, err := f.updateUC().Execute(excludeCtx(), excludeReq(classStatusExcluded)); err != nil {
		t.Fatalf("unexpected error with a nil ReferenceChecker: %v", err)
	}
	if f.repo.updateCalls != 1 {
		t.Fatalf("expected 1 update call, got %d", f.repo.updateCalls)
	}
}
