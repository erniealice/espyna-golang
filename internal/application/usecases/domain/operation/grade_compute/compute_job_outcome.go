package grade_compute

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/gradecompute"
	"github.com/erniealice/espyna-golang/registry/entityid"

	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	joboutcomelinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_line"
	joboutcomesummarypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	phaseoutcomesummarypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
	scorescalepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
	scorescalebandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale_band"
	scoringschemepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_scheme"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// ComputeJobOutcomeRequest is the structured input for the JOB-level (year-final)
// grade roll-up: the job whose phase grades roll up into one job_outcome_summary
// (+ a per-subject job_outcome_line).
type ComputeJobOutcomeRequest struct {
	JobId string
}

// ComputeJobOutcomeResponse returns the upserted job summary + line carrying the
// year-final grade, plus the terminal phase it was drawn from (traceability).
type ComputeJobOutcomeResponse struct {
	Summary            *joboutcomesummarypb.JobOutcomeSummary
	Line               *joboutcomelinepb.JobOutcomeLine
	ScaledScore        float64
	ScaledLabel        string
	Composite          float64
	TerminalJobPhaseId string
	ScoringSchemeId    string
}

// ComputeJobOutcomeUseCase rolls a job's graded phases (phase_outcome_summary
// rows) up into the year-final job_outcome_summary + job_outcome_line.
//
// Policy (see gradecompute.SelectTerminalPhase): the year-final grade = the
// TERMINAL (highest phase_order) graded phase's transmuted grade — determined
// empirically vs the authoritative prod report-card `final` grades (terminal
// pass-through matched 96.2%, every averaging variant <66%; single-semester
// jobs pass their lone phase through). This is a pure PASS-THROUGH of the
// already-validated phase summary — it does NOT re-derive grades from
// task_outcomes (that is ComputePhaseOutcome's job); re-run the phase compute
// first after any grade mutation, then this.
type ComputeJobOutcomeUseCase struct {
	repositories Repositories
	services     Services
}

// NewComputeJobOutcomeUseCase constructs the use case.
func NewComputeJobOutcomeUseCase(repositories Repositories, services Services) *ComputeJobOutcomeUseCase {
	return &ComputeJobOutcomeUseCase{repositories: repositories, services: services}
}

func (uc *ComputeJobOutcomeUseCase) msg(ctx context.Context, key, def string) string {
	return contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, key, def)
}

// Execute runs the year-final roll-up for one job.
func (uc *ComputeJobOutcomeUseCase) Execute(ctx context.Context, req *ComputeJobOutcomeRequest) (*ComputeJobOutcomeResponse, error) {
	// Gate 1 (Action): writing a job_outcome_summary is a create action.
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.JobOutcomeSummary,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	if req == nil || req.JobId == "" {
		return nil, errors.New(uc.msg(ctx, "grade_compute.validation.job_id_required", "[ERR-DEFAULT] Job ID is required"))
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *ComputeJobOutcomeResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeJobCore(txCtx, req)
			if err != nil {
				return err
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.executeJobCore(ctx, req)
}

func (uc *ComputeJobOutcomeUseCase) executeJobCore(ctx context.Context, req *ComputeJobOutcomeRequest) (*ComputeJobOutcomeResponse, error) {
	// 1. Read the job for its denormalized client_id/workspace_id + subject label.
	job, err := uc.readJob(ctx, req.JobId)
	if err != nil {
		return nil, err
	}

	// 2. List the job's graded phase summaries (the phase-compute output).
	summaries, err := uc.listPhaseSummaries(ctx, req.JobId)
	if err != nil {
		return nil, err
	}

	// 3. Build the per-phase roll-up inputs; read each phase for its phase_order.
	//    A phase with no composite (summary_score) is not a graded phase and is
	//    skipped (no fabrication). NOTE: the phase_outcome_summary adapter's
	//    ListByJob does not project scaled_score/scaled_label, so we carry only
	//    the composite and RE-TRANSMUTE the terminal composite below (deterministic
	//    — same scheme/bands the phase compute used, so it reproduces the stored
	//    phase grade exactly).
	var rollups []gradecompute.PhaseRollup
	phaseByID := make(map[string]*jobphasepb.JobPhase, len(summaries))
	for _, s := range summaries {
		if s == nil || s.SummaryScore == nil {
			continue
		}
		phase, perr := uc.readJobPhase(ctx, s.JobPhaseId)
		if perr != nil {
			return nil, perr
		}
		phaseByID[s.JobPhaseId] = phase
		rollups = append(rollups, gradecompute.PhaseRollup{
			JobPhaseID: s.JobPhaseId,
			PhaseOrder: phase.PhaseOrder,
			Composite:  *s.SummaryScore,
		})
	}
	if len(rollups) == 0 {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.no_graded_phases",
			"[ERR-DEFAULT] job %s has no graded phase summaries to roll up"), req.JobId)
	}

	// 4. Year-final policy: the terminal (highest phase_order) phase carries.
	terminal, ok := gradecompute.SelectTerminalPhase(rollups)
	if !ok {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.no_terminal_phase",
			"[ERR-DEFAULT] could not select a terminal phase for job %s"), req.JobId)
	}

	// 5. Resolve the terminal phase's scoring scheme (for the snapshot) + derive
	//    the score_scale_band for the transcript line by re-transmuting the
	//    terminal composite (also a consistency check vs the stored scaled grade).
	termPhase := phaseByID[terminal.JobPhaseID]
	var templatePhaseSchemeID *string
	if termPhase.TemplatePhaseId != nil && *termPhase.TemplatePhaseId != "" {
		tplPhase, terr := uc.readJobTemplatePhase(ctx, *termPhase.TemplatePhaseId)
		if terr != nil {
			return nil, terr
		}
		if tplPhase != nil {
			templatePhaseSchemeID = tplPhase.ScoringSchemeId
		}
	}
	schemeID, err := gradecompute.ResolveScoringScheme(termPhase.ScoringSchemeId, templatePhaseSchemeID)
	if err != nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.scheme_unresolved",
			"[ERR-DEFAULT] no scoring scheme resolved for terminal job_phase %s (job %s): %w"),
			terminal.JobPhaseID, req.JobId, err)
	}
	band, err := uc.deriveTerminalBand(ctx, schemeID, terminal.Composite)
	if err != nil {
		return nil, err
	}
	// The year-final grade IS the terminal composite transmuted through the same
	// scheme/bands the phase compute used (reproduces the stored phase grade).
	scaledScore, scaledLabel := gradecompute.BandOutput(band)

	// 6. Upsert the job_outcome_summary carrying the year-final grade.
	summary, err := uc.upsertJobSummary(ctx, job, schemeID, terminal.Composite, scaledScore, scaledLabel)
	if err != nil {
		return nil, err
	}

	// 7. Upsert the per-subject transcript line under that summary.
	line, err := uc.upsertJobLine(ctx, job, summary, scaledScore, scaledLabel, band)
	if err != nil {
		return nil, err
	}

	return &ComputeJobOutcomeResponse{
		Summary:            summary,
		Line:               line,
		ScaledScore:        scaledScore,
		ScaledLabel:        scaledLabel,
		Composite:          terminal.Composite,
		TerminalJobPhaseId: terminal.JobPhaseID,
		ScoringSchemeId:    schemeID,
	}, nil
}

// deriveTerminalBand re-transmutes the terminal composite through the scheme's
// score_scale to recover the score_scale_band (for the transcript line's FK).
// It reuses the pure gradecompute.Transmute — a nil band is a config gap and is
// surfaced (fail-loud), never a silent 0.
func (uc *ComputeJobOutcomeUseCase) deriveTerminalBand(ctx context.Context, schemeID string, composite float64) (*scorescalebandpb.ScoreScaleBand, error) {
	scheme, err := uc.readScoringScheme(ctx, schemeID)
	if err != nil {
		return nil, err
	}
	if scheme.ScoreScaleId == nil || *scheme.ScoreScaleId == "" {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.scheme_no_scale",
			"[ERR-DEFAULT] scoring scheme %s has no score_scale_id"), schemeID)
	}
	scale, err := uc.readScoreScale(ctx, *scheme.ScoreScaleId)
	if err != nil {
		return nil, err
	}
	bands, err := uc.listBands(ctx, *scheme.ScoreScaleId)
	if err != nil {
		return nil, err
	}
	if len(bands) == 0 {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.no_bands",
			"[ERR-DEFAULT] score_scale %s has no bands"), *scheme.ScoreScaleId)
	}
	band, err := gradecompute.Transmute(scale, bands, composite)
	if err != nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.transmute_failed",
			"[ERR-DEFAULT] year-final transmute failed for scheme %s composite %v: %w"), schemeID, composite, err)
	}
	return band, nil
}

// --- repository reads ---

func (uc *ComputeJobOutcomeUseCase) readJob(ctx context.Context, id string) (*jobpb.Job, error) {
	resp, err := uc.repositories.Job.ReadJob(ctx, &jobpb.ReadJobRequest{Data: &jobpb.Job{Id: id}})
	if err != nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.read_job_failed",
			"[ERR-DEFAULT] failed to read job %s: %w"), id, err)
	}
	if resp == nil || len(resp.Data) == 0 || resp.Data[0] == nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.job_not_found",
			"[ERR-DEFAULT] job %s not found"), id)
	}
	return resp.Data[0], nil
}

func (uc *ComputeJobOutcomeUseCase) listPhaseSummaries(ctx context.Context, jobID string) ([]*phaseoutcomesummarypb.PhaseOutcomeSummary, error) {
	resp, err := uc.repositories.PhaseOutcomeSummary.ListByJob(ctx,
		&phaseoutcomesummarypb.ListPhaseOutcomeSummarysByJobRequest{JobId: jobID})
	if err != nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.list_phase_summaries_failed",
			"[ERR-DEFAULT] failed to list phase_outcome_summary for job %s: %w"), jobID, err)
	}
	if resp == nil {
		return nil, nil
	}
	return resp.PhaseOutcomeSummarys, nil
}

func (uc *ComputeJobOutcomeUseCase) readJobPhase(ctx context.Context, id string) (*jobphasepb.JobPhase, error) {
	resp, err := uc.repositories.JobPhase.ReadJobPhase(ctx, &jobphasepb.ReadJobPhaseRequest{
		Data: &jobphasepb.JobPhase{Id: id},
	})
	if err != nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.read_job_phase_failed",
			"[ERR-DEFAULT] failed to read job_phase %s: %w"), id, err)
	}
	if resp == nil || len(resp.Data) == 0 || resp.Data[0] == nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.job_phase_not_found",
			"[ERR-DEFAULT] job_phase %s not found"), id)
	}
	return resp.Data[0], nil
}

// upsertJobSummary writes the year-final grade onto job_outcome_summary,
// updating the existing summary for this job when one exists (idempotent
// re-compute) or creating a fresh one. A lookup failure MUST abort — swallowing
// it degrades every re-run into a duplicate insert (the phase-summary lesson).
func (uc *ComputeJobOutcomeUseCase) upsertJobSummary(
	ctx context.Context,
	job *jobpb.Job,
	schemeID string,
	composite float64,
	scaledScore float64,
	scaledLabel string,
) (*joboutcomesummarypb.JobOutcomeSummary, error) {
	now := time.Now()
	score := scaledScore
	label := scaledLabel
	scheme := schemeID

	data := &joboutcomesummarypb.JobOutcomeSummary{
		JobId:              job.Id,
		SummaryType:        enumspb.SummaryType_SUMMARY_TYPE_ACADEMIC_RECORD,
		ScoringMethod:      enumspb.ScoringMethod_SCORING_METHOD_SUM,
		SummaryScore:       &composite,
		ScaledScore:        &score,
		ScaledLabel:        &label,
		ScoringSchemeId:    &scheme,
		Active:             true,
		DateModified:       ptrInt64(now.UnixMilli()),
		DateModifiedString: ptrString(now.Format(time.RFC3339)),
	}
	// Portal-read denormalization (single-table student/guardian self-read).
	if job.WorkspaceId != nil {
		data.WorkspaceId = *job.WorkspaceId
	}
	if job.ClientId != nil && *job.ClientId != "" {
		cid := *job.ClientId
		data.ClientId = &cid
	}

	existing, err := uc.repositories.JobOutcomeSummary.GetByJob(ctx,
		&joboutcomesummarypb.GetJobOutcomeSummaryByJobRequest{JobId: job.Id})
	if err != nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.lookup_job_summary_failed",
			"[ERR-DEFAULT] failed to look up existing job_outcome_summary for job %s: %w"), job.Id, err)
	}

	if existing != nil && existing.JobOutcomeSummary != nil && existing.JobOutcomeSummary.Id != "" {
		prev := existing.JobOutcomeSummary
		// WRITE-BOUNDARY FREEZE GUARD (B2, Phase-1b): refuse to overwrite the
		// year-final grade (scaled_score/scaled_label) on a frozen row. An
		// is_authoritative summary (the imported prod finals) is immutable to
		// recompute. Fail-closed here so the freeze holds for EVERY caller — not
		// just cmd/year-final-compute's enumeration filter. A direct/API recompute
		// of a frozen job aborts instead of clobbering the pinned grade.
		if prev.IsAuthoritative {
			return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.summary_frozen",
				"[ERR-DEFAULT] refusing to overwrite authoritative (frozen) job_outcome_summary for job %s"), job.Id)
		}
		data.Id = prev.Id
		data.DateCreated = prev.DateCreated
		data.DateCreatedString = prev.DateCreatedString
		if data.IssuedBy == "" {
			data.IssuedBy = prev.IssuedBy
		}
		resp, uerr := uc.repositories.JobOutcomeSummary.UpdateJobOutcomeSummary(ctx,
			&joboutcomesummarypb.UpdateJobOutcomeSummaryRequest{Data: data})
		if uerr != nil {
			return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.update_job_summary_failed",
				"[ERR-DEFAULT] failed to update job_outcome_summary for job %s: %w"), job.Id, uerr)
		}
		return firstJobSummary(resp.GetData(), data), nil
	}

	data.Id = uc.services.IDGenerator.GenerateID()
	data.DateCreated = ptrInt64(now.UnixMilli())
	data.DateCreatedString = ptrString(now.Format(time.RFC3339))
	resp, err := uc.repositories.JobOutcomeSummary.CreateJobOutcomeSummary(ctx,
		&joboutcomesummarypb.CreateJobOutcomeSummaryRequest{Data: data})
	if err != nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.create_job_summary_failed",
			"[ERR-DEFAULT] failed to create job_outcome_summary for job %s: %w"), job.Id, err)
	}
	return firstJobSummary(resp.GetData(), data), nil
}

// upsertJobLine writes the single per-subject transcript line under the summary,
// updating the existing line for this summary when one exists. The line label is
// the job's subject name; output_value/label carry the year-final grade.
func (uc *ComputeJobOutcomeUseCase) upsertJobLine(
	ctx context.Context,
	job *jobpb.Job,
	summary *joboutcomesummarypb.JobOutcomeSummary,
	scaledScore float64,
	scaledLabel string,
	band *scorescalebandpb.ScoreScaleBand,
) (*joboutcomelinepb.JobOutcomeLine, error) {
	now := time.Now()
	score := scaledScore
	label := scaledLabel

	data := &joboutcomelinepb.JobOutcomeLine{
		JobOutcomeSummaryId: summary.Id,
		Label:               job.Name,
		OutputValue:         &score,
		OutputLabel:         &label,
		ReportingRole:       enumspb.ReportingRole_REPORTING_ROLE_PRIMARY,
		Active:              true,
		DateModified:        ptrInt64(now.UnixMilli()),
	}
	if band != nil && band.Id != "" {
		bid := band.Id
		data.ScoreScaleBandId = &bid
	}
	// workspace_id is NOT NULL on job_outcome_line — fall back to the summary's.
	if job.WorkspaceId != nil && *job.WorkspaceId != "" {
		data.WorkspaceId = *job.WorkspaceId
	} else {
		data.WorkspaceId = summary.WorkspaceId
	}
	if job.ClientId != nil && *job.ClientId != "" {
		cid := *job.ClientId
		data.ClientId = &cid
	}

	existing, err := uc.findLineBySummary(ctx, summary.Id)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Id != "" {
		data.Id = existing.Id
		data.DateCreated = existing.DateCreated
		resp, uerr := uc.repositories.JobOutcomeLine.UpdateJobOutcomeLine(ctx,
			&joboutcomelinepb.UpdateJobOutcomeLineRequest{Data: data})
		if uerr != nil {
			return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.update_job_line_failed",
				"[ERR-DEFAULT] failed to update job_outcome_line for summary %s: %w"), summary.Id, uerr)
		}
		return firstJobLine(resp.GetData(), data), nil
	}

	data.Id = uc.services.IDGenerator.GenerateID()
	data.DateCreated = ptrInt64(now.UnixMilli())
	resp, err := uc.repositories.JobOutcomeLine.CreateJobOutcomeLine(ctx,
		&joboutcomelinepb.CreateJobOutcomeLineRequest{Data: data})
	if err != nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.create_job_line_failed",
			"[ERR-DEFAULT] failed to create job_outcome_line for summary %s: %w"), summary.Id, err)
	}
	return firstJobLine(resp.GetData(), data), nil
}

// findLineBySummary returns the existing transcript line for a summary (there is
// at most one in this model), filtered server-side by job_outcome_summary_id. A
// query error aborts (fail-loud); a not-found returns nil.
func (uc *ComputeJobOutcomeUseCase) findLineBySummary(ctx context.Context, summaryID string) (*joboutcomelinepb.JobOutcomeLine, error) {
	resp, err := uc.repositories.JobOutcomeLine.ListJobOutcomeLines(ctx, &joboutcomelinepb.ListJobOutcomeLinesRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "job_outcome_summary_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    summaryID,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.lookup_job_line_failed",
			"[ERR-DEFAULT] failed to look up existing job_outcome_line for summary %s: %w"), summaryID, err)
	}
	if resp != nil {
		for _, l := range resp.Data {
			if l != nil && l.Active && l.JobOutcomeSummaryId == summaryID {
				return l, nil
			}
		}
	}
	return nil, nil
}

func (uc *ComputeJobOutcomeUseCase) readJobTemplatePhase(ctx context.Context, id string) (*jobtemplatephasepb.JobTemplatePhase, error) {
	resp, err := uc.repositories.JobTemplatePhase.ReadJobTemplatePhase(ctx, &jobtemplatephasepb.ReadJobTemplatePhaseRequest{
		Data: &jobtemplatephasepb.JobTemplatePhase{Id: id},
	})
	if err != nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.read_job_template_phase_failed",
			"[ERR-DEFAULT] failed to read job_template_phase %s: %w"), id, err)
	}
	if resp == nil || len(resp.Data) == 0 {
		// A dangling template_phase_id is not fatal: the phase rung may still
		// resolve the scheme on its own. Return nil and let the ladder decide.
		return nil, nil
	}
	return resp.Data[0], nil
}

func (uc *ComputeJobOutcomeUseCase) readScoringScheme(ctx context.Context, id string) (*scoringschemepb.ScoringScheme, error) {
	resp, err := uc.repositories.ScoringScheme.ReadScoringScheme(ctx, &scoringschemepb.ReadScoringSchemeRequest{
		Data: &scoringschemepb.ScoringScheme{Id: id},
	})
	if err != nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.read_scheme_failed",
			"[ERR-DEFAULT] failed to read scoring_scheme %s: %w"), id, err)
	}
	if resp == nil || len(resp.Data) == 0 || resp.Data[0] == nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.scheme_not_found",
			"[ERR-DEFAULT] scoring_scheme %s not found"), id)
	}
	return resp.Data[0], nil
}

func (uc *ComputeJobOutcomeUseCase) readScoreScale(ctx context.Context, id string) (*scorescalepb.ScoreScale, error) {
	resp, err := uc.repositories.ScoreScale.ReadScoreScale(ctx, &scorescalepb.ReadScoreScaleRequest{
		Data: &scorescalepb.ScoreScale{Id: id},
	})
	if err != nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.read_scale_failed",
			"[ERR-DEFAULT] failed to read score_scale %s: %w"), id, err)
	}
	if resp == nil || len(resp.Data) == 0 || resp.Data[0] == nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.scale_not_found",
			"[ERR-DEFAULT] score_scale %s not found"), id)
	}
	return resp.Data[0], nil
}

func (uc *ComputeJobOutcomeUseCase) listBands(ctx context.Context, scaleID string) ([]*scorescalebandpb.ScoreScaleBand, error) {
	resp, err := uc.repositories.ScoreScaleBand.ListScoreScaleBands(ctx, &scorescalebandpb.ListScoreScaleBandsRequest{})
	if err != nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.list_bands_failed",
			"[ERR-DEFAULT] failed to list score_scale_bands for scale %s: %w"), scaleID, err)
	}
	var bands []*scorescalebandpb.ScoreScaleBand
	if resp != nil {
		for _, b := range resp.Data {
			if b == nil || b.ScoreScaleId != scaleID {
				continue
			}
			bands = append(bands, b)
		}
	}
	return bands, nil
}

func firstJobSummary(data []*joboutcomesummarypb.JobOutcomeSummary, sent *joboutcomesummarypb.JobOutcomeSummary) *joboutcomesummarypb.JobOutcomeSummary {
	if len(data) > 0 && data[0] != nil {
		return data[0]
	}
	return sent
}

func firstJobLine(data []*joboutcomelinepb.JobOutcomeLine, sent *joboutcomelinepb.JobOutcomeLine) *joboutcomelinepb.JobOutcomeLine {
	if len(data) > 0 && data[0] != nil {
		return data[0]
	}
	return sent
}
