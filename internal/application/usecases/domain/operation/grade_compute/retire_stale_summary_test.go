package grade_compute

import (
	"context"
	"errors"
	"testing"

	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	joboutcomelinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_line"
	joboutcomesummarypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
	phaseoutcomesummarypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
)

// fakeRetirePhaseSummaryRepo serves GetByJobPhase from a stack of active rows and
// pops one per soft delete, mirroring the adapter's active-only lookup.
type fakeRetirePhaseSummaryRepo struct {
	phaseoutcomesummarypb.UnimplementedPhaseOutcomeSummaryDomainServiceServer
	active  []string
	deleted []string
}

func (f *fakeRetirePhaseSummaryRepo) GetByJobPhase(_ context.Context, _ *phaseoutcomesummarypb.GetPhaseOutcomeSummaryByJobPhaseRequest) (*phaseoutcomesummarypb.GetPhaseOutcomeSummaryByJobPhaseResponse, error) {
	if len(f.active) == 0 {
		return &phaseoutcomesummarypb.GetPhaseOutcomeSummaryByJobPhaseResponse{Success: true}, nil
	}
	return &phaseoutcomesummarypb.GetPhaseOutcomeSummaryByJobPhaseResponse{
		Success:             true,
		PhaseOutcomeSummary: &phaseoutcomesummarypb.PhaseOutcomeSummary{Id: f.active[0], Active: true},
	}, nil
}

func (f *fakeRetirePhaseSummaryRepo) DeletePhaseOutcomeSummary(_ context.Context, req *phaseoutcomesummarypb.DeletePhaseOutcomeSummaryRequest) (*phaseoutcomesummarypb.DeletePhaseOutcomeSummaryResponse, error) {
	f.deleted = append(f.deleted, req.Data.Id)
	f.active = f.active[1:]
	return &phaseoutcomesummarypb.DeletePhaseOutcomeSummaryResponse{Success: true}, nil
}

// TestRetireStaleSummary_RetiresEveryActiveRow proves a phase whose last value was
// cleared loses its stale composite/grade (the Criteria Total + Rating go blank).
func TestRetireStaleSummary_RetiresEveryActiveRow(t *testing.T) {
	repo := &fakeRetirePhaseSummaryRepo{active: []string{"s2", "s1"}}
	uc := &ComputePhaseOutcomeUseCase{repositories: Repositories{PhaseOutcomeSummary: repo}}
	if err := uc.retireStaleSummary(context.Background(), "p1"); err != nil {
		t.Fatalf("retire: %v", err)
	}
	if len(repo.active) != 0 || len(repo.deleted) != 2 {
		t.Fatalf("want both rows retired, got deleted=%v remaining=%v", repo.deleted, repo.active)
	}
}

// TestRetireStaleSummary_NoRowIsNoop proves a blank phase with no summary writes nothing.
func TestRetireStaleSummary_NoRowIsNoop(t *testing.T) {
	repo := &fakeRetirePhaseSummaryRepo{}
	uc := &ComputePhaseOutcomeUseCase{repositories: Repositories{PhaseOutcomeSummary: repo}}
	if err := uc.retireStaleSummary(context.Background(), "p1"); err != nil {
		t.Fatalf("retire: %v", err)
	}
	if len(repo.deleted) != 0 {
		t.Fatalf("want no deletes, got %v", repo.deleted)
	}
}

type fakeRetireJobSummaryRepo struct {
	fakeJobOutcomeSummaryRepo
	deleted []string
}

func (f *fakeRetireJobSummaryRepo) DeleteJobOutcomeSummary(_ context.Context, req *joboutcomesummarypb.DeleteJobOutcomeSummaryRequest) (*joboutcomesummarypb.DeleteJobOutcomeSummaryResponse, error) {
	f.deleted = append(f.deleted, req.Data.Id)
	return &joboutcomesummarypb.DeleteJobOutcomeSummaryResponse{Success: true}, nil
}

type fakeRetireJobLineRepo struct {
	joboutcomelinepb.UnimplementedJobOutcomeLineDomainServiceServer
	lines   []*joboutcomelinepb.JobOutcomeLine
	deleted []string
}

func (f *fakeRetireJobLineRepo) ListJobOutcomeLines(_ context.Context, _ *joboutcomelinepb.ListJobOutcomeLinesRequest) (*joboutcomelinepb.ListJobOutcomeLinesResponse, error) {
	return &joboutcomelinepb.ListJobOutcomeLinesResponse{Success: true, Data: f.lines}, nil
}

func (f *fakeRetireJobLineRepo) DeleteJobOutcomeLine(_ context.Context, req *joboutcomelinepb.DeleteJobOutcomeLineRequest) (*joboutcomelinepb.DeleteJobOutcomeLineResponse, error) {
	f.deleted = append(f.deleted, req.Data.Id)
	return &joboutcomelinepb.DeleteJobOutcomeLineResponse{Success: true}, nil
}

func newBlankJobUC(summary *joboutcomesummarypb.JobOutcomeSummary) (*ComputeJobOutcomeUseCase, *fakeRetireJobSummaryRepo, *fakeRetireJobLineRepo) {
	sumRepo := &fakeRetireJobSummaryRepo{fakeJobOutcomeSummaryRepo: fakeJobOutcomeSummaryRepo{existing: summary}}
	lineRepo := &fakeRetireJobLineRepo{lines: []*joboutcomelinepb.JobOutcomeLine{
		{Id: "l1", JobOutcomeSummaryId: "js1", Active: true},
	}}
	uc := &ComputeJobOutcomeUseCase{repositories: Repositories{
		Job:                 &fakeJobRepoGC{job: &jobpb.Job{Id: "j1"}},
		PhaseOutcomeSummary: &fakePhaseSummaryRepoGC{byJob: nil}, // every phase cleared
		JobOutcomeSummary:   sumRepo,
		JobOutcomeLine:      lineRepo,
	}}
	return uc, sumRepo, lineRepo
}

// TestExecuteJobCore_NoGradedPhasesRetiresStaleYearFinal proves a job whose every
// phase was cleared retires its stale year-final summary + transcript line.
func TestExecuteJobCore_NoGradedPhasesRetiresStaleYearFinal(t *testing.T) {
	uc, sumRepo, lineRepo := newBlankJobUC(&joboutcomesummarypb.JobOutcomeSummary{Id: "js1", Active: true})
	_, err := uc.executeJobCore(context.Background(), &ComputeJobOutcomeRequest{JobId: "j1"})
	if !errors.Is(err, ErrNoGradedPhases) {
		t.Fatalf("want ErrNoGradedPhases, got %v", err)
	}
	if len(sumRepo.deleted) != 1 || sumRepo.deleted[0] != "js1" {
		t.Fatalf("want summary js1 retired, got %v", sumRepo.deleted)
	}
	if len(lineRepo.deleted) != 1 || lineRepo.deleted[0] != "l1" {
		t.Fatalf("want line l1 retired, got %v", lineRepo.deleted)
	}
}

// TestExecuteJobCore_NoGradedPhasesKeepsAuthoritative proves the freeze guard holds:
// an imported (is_authoritative) year-final is never retired.
func TestExecuteJobCore_NoGradedPhasesKeepsAuthoritative(t *testing.T) {
	uc, sumRepo, lineRepo := newBlankJobUC(&joboutcomesummarypb.JobOutcomeSummary{Id: "js1", Active: true, IsAuthoritative: true})
	if _, err := uc.executeJobCore(context.Background(), &ComputeJobOutcomeRequest{JobId: "j1"}); !errors.Is(err, ErrNoGradedPhases) {
		t.Fatalf("want ErrNoGradedPhases, got %v", err)
	}
	if len(sumRepo.deleted) != 0 || len(lineRepo.deleted) != 0 {
		t.Fatalf("authoritative summary must stay pinned, deleted summary=%v line=%v", sumRepo.deleted, lineRepo.deleted)
	}
}
