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
	"errors"
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

// ── Narrow composition adapters (W2 inline recompute) ───────────────────────
//
// The fayna outcome_matrix record action (the grade-sheet edit-mode save path)
// recomputes the affected phase then job roll-up inline after a successful
// ACADEMIC cell write, so a report card is fresh the moment a grade is entered
// (Q-GSE-5, inline recompute). These two factories wrap the ComputePhaseOutcome
// / ComputeJobOutcome pass-throughs above into the narrow bare-func closures the
// app composition stores on AppContext.ComputePhaseOutcome / .ComputeJobOutcome
// (the GenerateDoc injection precedent) — the fayna EngineBlock type-asserts the
// exact signature `func(context.Context, string) (bool, error)`, so no fayna
// dependency on this package is created (identical to how GenerateDoc is
// asserted as a bare `func([]byte, map[string]any) ([]byte, error)`).
//
// Return contract (mapped to ratingFresh by the record action):
//   (true,  nil) → a recompute actually ran; the summary is fresh.
//   (false, nil) → deliberately SKIPPED because the target is authoritative
//                  (frozen imported finals — ErrSummaryFrozen). The existing
//                  grade stands and is NOT stale, so the save reports the rating
//                  as fresh (ratingNotRecomputed reason), never as failed.
//   (false, err) → a genuine compute failure. The grade persisted but the
//                  rating is now stale + retryable (ratingFresh:false). NEVER a
//                  reason to fail the cell save.
//
// A nil container / unwired use-case surfaces as (false, err); the record action
// degrades that to ratingFresh:false (fail-safe) exactly like a compute failure.

// NewComputePhaseOutcomeAdapter returns the phase-level inline-recompute closure
// bound to this container. Phase roll-up only ever writes
// SUMMARY_TYPE_ACADEMIC_RECORD summaries (score_scale transmutation), so it
// structurally never touches a frozen/non-scaled deportment summary — a
// non-recomputable phase (no scoring scheme resolves) surfaces as (false, err)
// and the save still succeeds with a stale rating.
func NewComputePhaseOutcomeAdapter(container *core.Container) func(ctx context.Context, jobPhaseID string) (bool, error) {
	return func(ctx context.Context, jobPhaseID string) (bool, error) {
		if _, err := ComputePhaseOutcome(ctx, container, jobPhaseID, ""); err != nil {
			return false, err
		}
		return true, nil
	}
}

// NewComputeJobOutcomeAdapter returns the job-level (year-final) inline-recompute
// closure bound to this container. It preserves the authoritative-summary freeze
// guard: an is_authoritative job_outcome_summary yields ErrSummaryFrozen from the
// use-case, which this adapter classifies as an expected skip (false, nil) — the
// pinned grade is never clobbered and the save is never failed for it.
func NewComputeJobOutcomeAdapter(container *core.Container) func(ctx context.Context, jobID string) (bool, error) {
	return func(ctx context.Context, jobID string) (bool, error) {
		if _, err := ComputeJobOutcome(ctx, container, jobID); err != nil {
			if errors.Is(err, grade_compute.ErrSummaryFrozen) {
				return false, nil // frozen/authoritative → not recomputed, not stale
			}
			return false, err
		}
		return true, nil
	}
}
