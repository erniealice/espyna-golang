package grade_compute

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/approvalctx"

	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

// inTxTransactor runs the operation on a child ctx, like the postgres
// Transactor that stamps the ambient *sql.Tx onto ctx.
type inTxTransactor struct{}

func (inTxTransactor) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
func (inTxTransactor) SupportsTransactions() bool               { return true }
func (inTxTransactor) IsTransactionActive(context.Context) bool { return true }

var errStopAfterFirstRead = errors.New("stop after first read")

// trustProbeJobPhase records whether the roll-up's first read ran as the
// trusted recompute, then stops the roll-up.
type trustProbeJobPhase struct {
	jobphasepb.JobPhaseDomainServiceServer
	trusted *bool
}

func (p trustProbeJobPhase) ReadJobPhase(ctx context.Context, _ *jobphasepb.ReadJobPhaseRequest) (*jobphasepb.ReadJobPhaseResponse, error) {
	*p.trusted = approvalctx.IsTrustedRecompute(ctx)
	return nil, errStopAfterFirstRead
}

type trustProbeJob struct {
	jobpb.JobDomainServiceServer
	trusted *bool
}

func (p trustProbeJob) ReadJob(ctx context.Context, _ *jobpb.ReadJobRequest) (*jobpb.ReadJobResponse, error) {
	*p.trusted = approvalctx.IsTrustedRecompute(ctx)
	return nil, errStopAfterFirstRead
}

func trustProbeServices(perms ...string) Services {
	svc := servicesWith(perms...)
	svc.Transactor = inTxTransactor{}
	return svc
}

// The grade-sheet save seam must recompute as the trusted system seam so a
// staff-scoped teacher finds (and updates) the existing summary instead of
// inserting a duplicate on every save; Execute (tools) stays request-scoped.
func TestExecuteAfterRecordRunsAsTrustedRecompute(t *testing.T) {
	var phaseTrusted, jobTrusted bool
	repos := Repositories{
		JobPhase: trustProbeJobPhase{trusted: &phaseTrusted},
		Job:      trustProbeJob{trusted: &jobTrusted},
	}
	svc := trustProbeServices("task_outcome:update")

	if _, err := NewComputePhaseOutcomeUseCase(repos, svc).ExecuteAfterRecord(recordGrantCtx(), &ComputePhaseOutcomeRequest{JobPhaseId: "jp-1"}); !errors.Is(err, errStopAfterFirstRead) {
		t.Fatalf("phase err = %v, want the probe stop", err)
	}
	if !phaseTrusted {
		t.Fatal("phase ExecuteAfterRecord: roll-up reads did not run as the trusted recompute")
	}
	if _, err := NewComputeJobOutcomeUseCase(repos, svc).ExecuteAfterRecord(recordGrantCtx(), &ComputeJobOutcomeRequest{JobId: "job-1"}); !errors.Is(err, errStopAfterFirstRead) {
		t.Fatalf("job err = %v, want the probe stop", err)
	}
	if !jobTrusted {
		t.Fatal("job ExecuteAfterRecord: roll-up reads did not run as the trusted recompute")
	}
}

func TestExecuteDoesNotRunAsTrustedRecompute(t *testing.T) {
	phaseTrusted, jobTrusted := true, true
	repos := Repositories{
		JobPhase: trustProbeJobPhase{trusted: &phaseTrusted},
		Job:      trustProbeJob{trusted: &jobTrusted},
	}
	svc := trustProbeServices("phase_outcome_summary:create", "job_outcome_summary:create")

	if _, err := NewComputePhaseOutcomeUseCase(repos, svc).Execute(recordGrantCtx(), &ComputePhaseOutcomeRequest{JobPhaseId: "jp-1"}); !errors.Is(err, errStopAfterFirstRead) {
		t.Fatalf("phase err = %v, want the probe stop", err)
	}
	if _, err := NewComputeJobOutcomeUseCase(repos, svc).Execute(recordGrantCtx(), &ComputeJobOutcomeRequest{JobId: "job-1"}); !errors.Is(err, errStopAfterFirstRead) {
		t.Fatalf("job err = %v, want the probe stop", err)
	}
	if phaseTrusted || jobTrusted {
		t.Fatalf("Execute ran as the trusted recompute (phase=%v job=%v)", phaseTrusted, jobTrusted)
	}
}

// Without a transaction the marker is never minted: the adapter would fail
// closed on a pool executor, so the non-tx path stays request-scoped.
func TestExecuteAfterRecordWithoutTransactorIsNotTrusted(t *testing.T) {
	phaseTrusted := true
	repos := Repositories{JobPhase: trustProbeJobPhase{trusted: &phaseTrusted}}
	if _, err := NewComputePhaseOutcomeUseCase(repos, servicesWith("task_outcome:update")).ExecuteAfterRecord(recordGrantCtx(), &ComputePhaseOutcomeRequest{JobPhaseId: "jp-1"}); !errors.Is(err, errStopAfterFirstRead) {
		t.Fatalf("err = %v, want the probe stop", err)
	}
	if phaseTrusted {
		t.Fatal("non-tx ExecuteAfterRecord minted the trusted recompute marker")
	}
}
