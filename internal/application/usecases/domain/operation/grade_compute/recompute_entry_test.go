package grade_compute

import (
	"context"
	"errors"
	"testing"

	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	phaseoutcomesummarypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
)

// fakeJobPhaseRepoGC is a minimal JobPhaseDomainServiceServer stub for the
// ambient-tx recompute entry tests.
type fakeJobPhaseRepoGC struct {
	jobphasepb.UnimplementedJobPhaseDomainServiceServer
	phase   *jobphasepb.JobPhase
	readErr error
}

func (f *fakeJobPhaseRepoGC) ReadJobPhase(_ context.Context, _ *jobphasepb.ReadJobPhaseRequest) (*jobphasepb.ReadJobPhaseResponse, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	return &jobphasepb.ReadJobPhaseResponse{Success: true, Data: []*jobphasepb.JobPhase{f.phase}}, nil
}

type fakeJobRepoGC struct {
	jobpb.UnimplementedJobDomainServiceServer
	job     *jobpb.Job
	readErr error
}

func (f *fakeJobRepoGC) ReadJob(_ context.Context, _ *jobpb.ReadJobRequest) (*jobpb.ReadJobResponse, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	return &jobpb.ReadJobResponse{Success: true, Data: []*jobpb.Job{f.job}}, nil
}

type fakePhaseSummaryRepoGC struct {
	phaseoutcomesummarypb.UnimplementedPhaseOutcomeSummaryDomainServiceServer
	byJob []*phaseoutcomesummarypb.PhaseOutcomeSummary
}

func (f *fakePhaseSummaryRepoGC) ListByJob(_ context.Context, _ *phaseoutcomesummarypb.ListPhaseOutcomeSummarysByJobRequest) (*phaseoutcomesummarypb.ListPhaseOutcomeSummarysByJobResponse, error) {
	return &phaseoutcomesummarypb.ListPhaseOutcomeSummarysByJobResponse{Success: true, PhaseOutcomeSummarys: f.byJob}, nil
}

// TestRecomputePhaseInAmbientTx_NotGradableSkips proves an unschemed (ledger) phase
// is an EXPECTED skip (false,nil), never a failure — so a blank/non-gradable phase
// does not roll the submit transition back.
func TestRecomputePhaseInAmbientTx_NotGradableSkips(t *testing.T) {
	// A phase with no scheme id and no template phase → ResolveScoringScheme fails →
	// CheckRecomputeEligibility returns (false,nil,nil) → skip.
	uc := &ComputePhaseOutcomeUseCase{
		repositories: Repositories{JobPhase: &fakeJobPhaseRepoGC{phase: &jobphasepb.JobPhase{Id: "p1"}}},
		services:     Services{},
	}
	ok, err := uc.RecomputePhaseInAmbientTx(context.Background(), "p1")
	if err != nil {
		t.Fatalf("non-gradable phase must be a skip, got err %v", err)
	}
	if ok {
		t.Fatal("non-gradable phase must report recomputed=false")
	}
}

// TestRecomputePhaseInAmbientTx_ReadErrorPropagates proves a genuine repository read
// failure propagates (false,err) → the submit transition rolls back.
func TestRecomputePhaseInAmbientTx_ReadErrorPropagates(t *testing.T) {
	boom := errors.New("db unavailable")
	uc := &ComputePhaseOutcomeUseCase{
		repositories: Repositories{JobPhase: &fakeJobPhaseRepoGC{readErr: boom}},
		services:     Services{},
	}
	if _, err := uc.RecomputePhaseInAmbientTx(context.Background(), "p1"); !errors.Is(err, boom) {
		t.Fatalf("genuine read failure must propagate, got %v", err)
	}
}

// TestRecomputeJobInAmbientTx_NoGradedPhasesSkips proves an all-blank job (no graded
// phase summaries) is an EXPECTED skip (false,nil) via the ErrNoGradedPhases
// sentinel — a fresh blank submit does not fail.
func TestRecomputeJobInAmbientTx_NoGradedPhasesSkips(t *testing.T) {
	uc := &ComputeJobOutcomeUseCase{
		repositories: Repositories{
			Job:                 &fakeJobRepoGC{job: &jobpb.Job{Id: "j1"}},
			PhaseOutcomeSummary: &fakePhaseSummaryRepoGC{byJob: nil}, // no graded phases
		},
		services: Services{},
	}
	ok, err := uc.RecomputeJobInAmbientTx(context.Background(), "j1")
	if err != nil {
		t.Fatalf("all-blank job must be a skip, got err %v", err)
	}
	if ok {
		t.Fatal("all-blank job must report recomputed=false")
	}
}

// TestRecomputeJobInAmbientTx_ReadErrorPropagates proves a genuine job read failure
// propagates → the submit transition rolls back.
func TestRecomputeJobInAmbientTx_ReadErrorPropagates(t *testing.T) {
	boom := errors.New("job read failed")
	uc := &ComputeJobOutcomeUseCase{
		repositories: Repositories{Job: &fakeJobRepoGC{readErr: boom}},
		services:     Services{},
	}
	if _, err := uc.RecomputeJobInAmbientTx(context.Background(), "j1"); !errors.Is(err, boom) {
		t.Fatalf("genuine job read failure must propagate, got %v", err)
	}
}
