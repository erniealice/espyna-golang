package grade_compute

import (
	"context"
	"testing"

	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	joboutcomesummarypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
)

// fakeJobOutcomeSummaryRepo is a hand-rolled JobOutcomeSummaryDomainServiceServer
// stub. It lets the write-boundary guard test control the existing (GetByJob)
// row and observe whether an overwrite (Update) or a fresh insert (Create) was
// attempted. Embedding the generated Unimplemented server satisfies the rest of
// the interface without stubbing every RPC.
type fakeJobOutcomeSummaryRepo struct {
	joboutcomesummarypb.UnimplementedJobOutcomeSummaryDomainServiceServer
	existing     *joboutcomesummarypb.JobOutcomeSummary
	updateCalled bool
	createCalled bool
}

func (f *fakeJobOutcomeSummaryRepo) GetByJob(
	_ context.Context, _ *joboutcomesummarypb.GetJobOutcomeSummaryByJobRequest,
) (*joboutcomesummarypb.GetJobOutcomeSummaryByJobResponse, error) {
	return &joboutcomesummarypb.GetJobOutcomeSummaryByJobResponse{
		Success:           true,
		JobOutcomeSummary: f.existing,
	}, nil
}

func (f *fakeJobOutcomeSummaryRepo) UpdateJobOutcomeSummary(
	_ context.Context, req *joboutcomesummarypb.UpdateJobOutcomeSummaryRequest,
) (*joboutcomesummarypb.UpdateJobOutcomeSummaryResponse, error) {
	f.updateCalled = true
	return &joboutcomesummarypb.UpdateJobOutcomeSummaryResponse{
		Success: true,
		Data:    []*joboutcomesummarypb.JobOutcomeSummary{req.Data},
	}, nil
}

func (f *fakeJobOutcomeSummaryRepo) CreateJobOutcomeSummary(
	_ context.Context, req *joboutcomesummarypb.CreateJobOutcomeSummaryRequest,
) (*joboutcomesummarypb.CreateJobOutcomeSummaryResponse, error) {
	f.createCalled = true
	return &joboutcomesummarypb.CreateJobOutcomeSummaryResponse{
		Success: true,
		Data:    []*joboutcomesummarypb.JobOutcomeSummary{req.Data},
	}, nil
}

// TestUpsertJobSummary_RefusesToOverwriteAuthoritative locks the B2 write-boundary
// freeze guard: when the existing job_outcome_summary is is_authoritative=true
// (a frozen imported prod final), the year-final recompute must abort WITHOUT
// issuing an Update or a Create. This is the deep enforcement line that holds for
// every caller of ComputeJobOutcome, not only cmd/year-final-compute's filter.
func TestUpsertJobSummary_RefusesToOverwriteAuthoritative(t *testing.T) {
	repo := &fakeJobOutcomeSummaryRepo{
		existing: &joboutcomesummarypb.JobOutcomeSummary{
			Id:              "jos-frozen-1",
			JobId:           "job-1",
			IsAuthoritative: true,
		},
	}
	uc := &ComputeJobOutcomeUseCase{
		repositories: Repositories{JobOutcomeSummary: repo},
		// nil Translator -> uc.msg returns the fallback message (nil-safe).
		services: Services{},
	}

	_, err := uc.upsertJobSummary(context.Background(), &jobpb.Job{Id: "job-1"}, "scheme-1", 6.0, 7.0, "A")
	if err == nil {
		t.Fatal("expected an error refusing to overwrite an authoritative (frozen) summary, got nil")
	}
	if repo.updateCalled {
		t.Error("guard breached: UpdateJobOutcomeSummary was called on a frozen authoritative row")
	}
	if repo.createCalled {
		t.Error("guard breached: CreateJobOutcomeSummary was called instead of refusing")
	}
}

// TestUpsertJobSummary_OverwritesNonAuthoritative is the control: a non-frozen
// existing summary updates cleanly (idempotent recompute), proving the guard
// gates only on the authoritative flag and does not block ordinary re-runs.
func TestUpsertJobSummary_OverwritesNonAuthoritative(t *testing.T) {
	repo := &fakeJobOutcomeSummaryRepo{
		existing: &joboutcomesummarypb.JobOutcomeSummary{
			Id:              "jos-open-1",
			JobId:           "job-1",
			IsAuthoritative: false,
		},
	}
	uc := &ComputeJobOutcomeUseCase{
		repositories: Repositories{JobOutcomeSummary: repo},
		services:     Services{},
	}

	if _, err := uc.upsertJobSummary(context.Background(), &jobpb.Job{Id: "job-1"}, "scheme-1", 6.0, 7.0, "A"); err != nil {
		t.Fatalf("non-authoritative summary should update cleanly, got: %v", err)
	}
	if !repo.updateCalled {
		t.Error("expected UpdateJobOutcomeSummary to be called for a non-frozen row")
	}
}
