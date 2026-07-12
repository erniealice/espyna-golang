package consumer

// grade_compute.go — public entrypoint for the education grade roll-up.
//
// The ComputePhaseOutcome use-case (the within-criterion MAX -> SUM composite
// -> score_scale transmutation that upserts phase_outcome_summary) lives in
// internal/application/usecases/domain/operation/grade_compute and is reachable
// on the container as GetUseCases().Operation.GradeCompute.ComputePhaseOutcome.
// Consumer apps (separate Go modules) cannot import that internal package to
// construct its request struct, so this thin pass-through exposes the call with
// no logic of its own — it just builds the request and delegates to the REAL
// use-case Execute. All math, repository plumbing, and the action-gate run
// inside the use-case exactly as in the running server.

import (
	"context"
	"fmt"

	core "github.com/erniealice/espyna-golang/internal/composition/core"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/grade_compute"
)

// GradeComputeResult is the public, internal-free shape returned by
// ComputePhaseOutcome: the resolved scheme id and the raw composite, plus the
// upserted summary's id and transmuted grade.
type GradeComputeResult struct {
	JobPhaseID      string
	SummaryID       string
	ScoringSchemeID string
	Composite       float64
	ScaledScore     float64
	ScaledLabel     string
}

// ComputePhaseOutcome runs the real grade roll-up for one job phase against the
// container's wired operation use-cases, upserting phase_outcome_summary. The
// ctx must carry an authorized principal (the use-case action-gates a
// PhaseOutcomeSummary:create). reportingCheckpointID may be empty.
func ComputePhaseOutcome(ctx context.Context, container *core.Container, jobPhaseID, reportingCheckpointID string) (*GradeComputeResult, error) {
	if container == nil {
		return nil, fmt.Errorf("grade compute: nil container")
	}
	uc := container.GetUseCases()
	if uc == nil || uc.Operation == nil || uc.Operation.GradeCompute == nil || uc.Operation.GradeCompute.ComputePhaseOutcome == nil {
		return nil, fmt.Errorf("grade compute: ComputePhaseOutcome use-case not wired on the operation rollup")
	}
	resp, err := uc.Operation.GradeCompute.ComputePhaseOutcome.Execute(ctx, &grade_compute.ComputePhaseOutcomeRequest{
		JobPhaseId:            jobPhaseID,
		ReportingCheckpointId: reportingCheckpointID,
	})
	if err != nil {
		return nil, err
	}
	out := &GradeComputeResult{
		JobPhaseID:      jobPhaseID,
		ScoringSchemeID: resp.ScoringSchemeId,
		Composite:       resp.Composite,
	}
	if s := resp.Summary; s != nil {
		out.SummaryID = s.Id
		if s.ScaledScore != nil {
			out.ScaledScore = *s.ScaledScore
		}
		if s.ScaledLabel != nil {
			out.ScaledLabel = *s.ScaledLabel
		}
	}
	return out, nil
}

// JobGradeComputeResult is the public shape returned by ComputeJobOutcome: the
// upserted job_outcome_summary id + line id and the year-final grade drawn from
// the terminal (Semester-2) phase.
type JobGradeComputeResult struct {
	JobID              string
	SummaryID          string
	LineID             string
	ScoringSchemeID    string
	Composite          float64
	ScaledScore        float64
	ScaledLabel        string
	TerminalJobPhaseID string
}

// ComputeJobOutcome runs the real year-final job roll-up for one job against the
// container's wired operation use-cases, upserting job_outcome_summary +
// job_outcome_line. The ctx must carry an authorized principal (the use-case
// action-gates a JobOutcomeSummary:create). It reads the job's already-computed
// phase_outcome_summary rows and passes the terminal phase's grade through.
func ComputeJobOutcome(ctx context.Context, container *core.Container, jobID string) (*JobGradeComputeResult, error) {
	if container == nil {
		return nil, fmt.Errorf("grade compute: nil container")
	}
	uc := container.GetUseCases()
	if uc == nil || uc.Operation == nil || uc.Operation.GradeCompute == nil || uc.Operation.GradeCompute.ComputeJobOutcome == nil {
		return nil, fmt.Errorf("grade compute: ComputeJobOutcome use-case not wired on the operation rollup")
	}
	resp, err := uc.Operation.GradeCompute.ComputeJobOutcome.Execute(ctx, &grade_compute.ComputeJobOutcomeRequest{
		JobId: jobID,
	})
	if err != nil {
		return nil, err
	}
	out := &JobGradeComputeResult{
		JobID:              jobID,
		ScoringSchemeID:    resp.ScoringSchemeId,
		Composite:          resp.Composite,
		ScaledScore:        resp.ScaledScore,
		ScaledLabel:        resp.ScaledLabel,
		TerminalJobPhaseID: resp.TerminalJobPhaseId,
	}
	if resp.Summary != nil {
		out.SummaryID = resp.Summary.Id
	}
	if resp.Line != nil {
		out.LineID = resp.Line.Id
	}
	return out, nil
}
