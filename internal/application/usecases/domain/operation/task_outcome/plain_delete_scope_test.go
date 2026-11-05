package task_outcome

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/reviewedit"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// decisionSpyRepo records the review-edit decision and clear-read mark every
// repository call sees.
type decisionSpyRepo struct {
	delRepo
	readDecisions []reviewedit.Decision
	readClearMark []bool
	plainDeletes  int
	deleteDecs    []reviewedit.Decision
}

func (s *decisionSpyRepo) ReadTaskOutcome(ctx context.Context, req *pb.ReadTaskOutcomeRequest) (*pb.ReadTaskOutcomeResponse, error) {
	d, _ := reviewedit.DecisionFromContext(ctx)
	s.readDecisions = append(s.readDecisions, d)
	s.readClearMark = append(s.readClearMark, reviewedit.IsClearRead(ctx))
	return s.delRepo.ReadTaskOutcome(ctx, req)
}

func (s *decisionSpyRepo) DeleteTaskOutcome(ctx context.Context, _ *pb.DeleteTaskOutcomeRequest) (*pb.DeleteTaskOutcomeResponse, error) {
	d, _ := reviewedit.DecisionFromContext(ctx)
	s.deleteDecs = append(s.deleteDecs, d)
	s.plainDeletes++
	return &pb.DeleteTaskOutcomeResponse{Success: true}, nil
}

var allOverrideCodes = map[string]bool{
	reviewedit.PermissionCode: true, reviewedit.CreateCode: true,
	reviewedit.DeleteCode: true, reviewedit.ClassOverrideCode: true,
}

// Review R2 blocker: the plain delete never carries a review/clear/class
// decision into the repository, whoever holds which code and even when the
// incoming ctx already carries one, so the adapter's read stays recorder-only
// (b388d68f behaviour). Each holder variant is checked.
func TestDeleteTaskOutcome_PlainDeleteNeverWidens(t *testing.T) {
	holders := map[string]map[string]bool{
		"clear_only":  {reviewedit.DeleteCode: true},
		"update_only": {reviewedit.PermissionCode: true},
		"class_only":  {reviewedit.ClassOverrideCode: true},
		"all_codes":   allOverrideCodes,
		"no_code":     {},
	}
	for name, perms := range holders {
		t.Run(name, func(t *testing.T) {
			repo := &decisionSpyRepo{delRepo: delRepo{condRepo: condRepo{plainRepo: plainRepo{stored: storedOutcome()}}}}
			svc := delSvcs(true)
			svc.Authorizer = strictAuthz{perms: perms}
			ctx := reviewedit.WithClearRead(reviewedit.WithDecision(staffCtx(7, "staff-1", true),
				reviewedit.Decision{Allowed: true, Fill: true, Clear: true, ClassOverride: true, WorkspaceScope: true}))
			if _, err := NewDeleteTaskOutcomeUseCase(DeleteTaskOutcomeRepositories{TaskOutcome: repo}, svc).Execute(ctx, delReq()); err != nil {
				t.Fatalf("delete: %v", err)
			}
			for i, d := range append(append([]reviewedit.Decision{}, repo.readDecisions...), repo.deleteDecs...) {
				if d != (reviewedit.Decision{}) {
					t.Fatalf("call %d saw decision %+v, want zero", i, d)
				}
			}
			if len(repo.readDecisions) == 0 || repo.plainDeletes != 1 {
				t.Fatalf("reads=%d deletes=%d", len(repo.readDecisions), repo.plainDeletes)
			}
		})
	}
}

// The override clear's own read is the only read carrying the clear-read mark.
func TestDeleteTaskOutcomeIfUnchanged_ReadCarriesClearMark(t *testing.T) {
	repo := &decisionSpyRepo{delRepo: delRepo{condRepo: condRepo{plainRepo: plainRepo{stored: storedOutcome()}}}}
	svc := delSvcs(true)
	svc.Authorizer = strictAuthz{perms: map[string]bool{reviewedit.DeleteCode: true}}
	if _, _, err := NewDeleteTaskOutcomeIfUnchangedUseCase(DeleteTaskOutcomeRepositories{TaskOutcome: repo}, svc).
		Execute(staffCtx(7, "staff-1", true), delReq(), storedOutcome()); err != nil {
		t.Fatalf("conditional delete: %v", err)
	}
	if len(repo.readClearMark) != 1 || !repo.readClearMark[0] || !repo.readDecisions[0].Clear {
		t.Fatalf("read marks=%v decisions=%+v", repo.readClearMark, repo.readDecisions)
	}
}

// Plain update never carries the clear-read mark, so Clear cannot widen it.
func TestUpdateTaskOutcome_NoClearMark(t *testing.T) {
	repo := &decisionSpyRepo{delRepo: delRepo{condRepo: condRepo{plainRepo: plainRepo{stored: storedOutcome()}}}}
	us, _ := svcs(true)
	us.Authorizer = strictAuthz{perms: map[string]bool{reviewedit.DeleteCode: true}}
	_, _ = NewUpdateTaskOutcomeUseCase(UpdateTaskOutcomeRepositories{TaskOutcome: repo}, us).
		Execute(staffCtx(7, "staff-1", true), &pb.UpdateTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: "o1", NumericValue: fp(4)}})
	for i, m := range repo.readClearMark {
		if m {
			t.Fatalf("update read %d carried the clear-read mark", i)
		}
	}
}
