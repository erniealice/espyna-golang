package subscription_group_product_plan

// Delete/Exclude in-use guard (plan.md §2 "Delete/Exclude guards: refuse when
// active assignments exist" — the section-delete-guard precedent).

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	espynaports "github.com/erniealice/espyna-golang/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan"
)

// stubDeleteChecker is a minimal espynaports.Checker stand-in: the embedded
// (nil) interface satisfies every OTHER method in the 20+-method contract —
// calling one would panic — while the ONE method this guard actually calls is
// explicitly overridden below. Standard partial-mock-via-embedding idiom.
type stubDeleteChecker struct {
	espynaports.Checker
	inUse map[string]bool
	err   error
}

func (s *stubDeleteChecker) GetSubscriptionGroupProductPlanInUseIDs(_ context.Context, ids []string) (map[string]bool, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := map[string]bool{}
	for _, id := range ids {
		if s.inUse[id] {
			out[id] = true
		}
	}
	return out, nil
}

type mockDeleteClassRepo struct {
	pb.UnimplementedSubscriptionGroupProductPlanDomainServiceServer
	deleteCalls int
}

func (m *mockDeleteClassRepo) DeleteSubscriptionGroupProductPlan(_ context.Context, _ *pb.DeleteSubscriptionGroupProductPlanRequest) (*pb.DeleteSubscriptionGroupProductPlanResponse, error) {
	m.deleteCalls++
	return &pb.DeleteSubscriptionGroupProductPlanResponse{Success: true}, nil
}

func newDeleteUC(repo *mockDeleteClassRepo, checker *stubDeleteChecker) *DeleteSubscriptionGroupProductPlanUseCase {
	var c espynaports.Checker
	if checker != nil {
		c = checker
	}
	return NewDeleteSubscriptionGroupProductPlanUseCase(
		DeleteSubscriptionGroupProductPlanRepositories{SubscriptionGroupProductPlan: repo},
		DeleteSubscriptionGroupProductPlanServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Translator:       ports.NewNoOpTranslator(),
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
			ReferenceChecker: c,
		},
	)
}

func TestDeleteClass_NoActiveAssignments_Succeeds(t *testing.T) {
	repo := &mockDeleteClassRepo{}
	checker := &stubDeleteChecker{inUse: map[string]bool{}}
	uc := newDeleteUC(repo, checker)

	_, err := uc.Execute(context.Background(), &pb.DeleteSubscriptionGroupProductPlanRequest{
		Data: &pb.SubscriptionGroupProductPlan{Id: "class-1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.deleteCalls != 1 {
		t.Fatalf("expected 1 delete call, got %d", repo.deleteCalls)
	}
}

func TestDeleteClass_ActiveAssignments_Rejects(t *testing.T) {
	repo := &mockDeleteClassRepo{}
	checker := &stubDeleteChecker{inUse: map[string]bool{"class-1": true}}
	uc := newDeleteUC(repo, checker)

	_, err := uc.Execute(context.Background(), &pb.DeleteSubscriptionGroupProductPlanRequest{
		Data: &pb.SubscriptionGroupProductPlan{Id: "class-1"},
	})
	if err == nil {
		t.Fatalf("expected rejection: class has active assignments")
	}
	if !strings.Contains(err.Error(), "active assignments") {
		t.Errorf("expected an active-assignments error, got %q", err.Error())
	}
	if repo.deleteCalls != 0 {
		t.Errorf("delete must not run on rejection, got %d calls", repo.deleteCalls)
	}
}

func TestDeleteClass_NilChecker_DegradesUnguarded(t *testing.T) {
	repo := &mockDeleteClassRepo{}
	uc := newDeleteUC(repo, nil)

	_, err := uc.Execute(context.Background(), &pb.DeleteSubscriptionGroupProductPlanRequest{
		Data: &pb.SubscriptionGroupProductPlan{Id: "class-1"},
	})
	if err != nil {
		t.Fatalf("unexpected error with a nil ReferenceChecker (composition gap must degrade, not brick): %v", err)
	}
	if repo.deleteCalls != 1 {
		t.Fatalf("expected 1 delete call, got %d", repo.deleteCalls)
	}
}
