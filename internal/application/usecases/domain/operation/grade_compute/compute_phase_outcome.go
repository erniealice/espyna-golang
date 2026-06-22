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
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	phaseoutcomesummarypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
	scorescalepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
	scorescalebandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale_band"
	scoringcomponentcriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_component_criteria"
	scoringschemepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_scheme"
	taskoutcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// ComputePhaseOutcomeRequest is the structured input for the grade roll-up.
//
// JobPhaseId is the phase being graded. ReportingCheckpointId is a pass-through
// snapshot stamped onto the summary (the interim/final report period); when
// empty the summary records no checkpoint (the caller defaults it to the
// terminal checkpoint upstream when a report period is in play).
type ComputePhaseOutcomeRequest struct {
	JobPhaseId            string
	ReportingCheckpointId string
}

// ComputePhaseOutcomeResponse returns the upserted summary carrying the
// transmuted grade (ScaledScore / ScaledLabel) plus the resolved scheme + raw
// composite for traceability.
type ComputePhaseOutcomeResponse struct {
	Summary         *phaseoutcomesummarypb.PhaseOutcomeSummary
	ScoringSchemeId string
	Composite       float64
}

// ComputePhaseOutcomeUseCase turns recorded per-assessment task_outcomes into a
// transmuted report-card grade and upserts it onto phase_outcome_summary.
type ComputePhaseOutcomeUseCase struct {
	repositories Repositories
	services     Services
}

// NewComputePhaseOutcomeUseCase constructs the use case.
func NewComputePhaseOutcomeUseCase(repositories Repositories, services Services) *ComputePhaseOutcomeUseCase {
	return &ComputePhaseOutcomeUseCase{repositories: repositories, services: services}
}

// Execute runs the grade roll-up for one job phase.
func (uc *ComputePhaseOutcomeUseCase) Execute(ctx context.Context, req *ComputePhaseOutcomeRequest) (*ComputePhaseOutcomeResponse, error) {
	// Gate 1 (Action): writing a phase_outcome_summary is a create action.
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.PhaseOutcomeSummary,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	if req == nil || req.JobPhaseId == "" {
		return nil, errors.New(uc.msg(ctx, "grade_compute.validation.job_phase_id_required", "[ERR-DEFAULT] Job phase ID is required"))
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *ComputePhaseOutcomeResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
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
	return uc.executeCore(ctx, req)
}

// executeCore is the data-plumbing around the pure gradecompute algorithm.
func (uc *ComputePhaseOutcomeUseCase) executeCore(ctx context.Context, req *ComputePhaseOutcomeRequest) (*ComputePhaseOutcomeResponse, error) {
	// 1. Resolve the phase + its scoring scheme on the precedence ladder.
	phase, err := uc.readJobPhase(ctx, req.JobPhaseId)
	if err != nil {
		return nil, err
	}

	var templatePhaseSchemeID *string
	if phase.TemplatePhaseId != nil && *phase.TemplatePhaseId != "" {
		tplPhase, terr := uc.readJobTemplatePhase(ctx, *phase.TemplatePhaseId)
		if terr != nil {
			return nil, terr
		}
		if tplPhase != nil {
			templatePhaseSchemeID = tplPhase.ScoringSchemeId
		}
	}

	schemeID, err := gradecompute.ResolveScoringScheme(phase.ScoringSchemeId, templatePhaseSchemeID)
	if err != nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.scheme_unresolved",
			"[ERR-DEFAULT] no scoring scheme resolved for job_phase %s: %w"), req.JobPhaseId, err)
	}

	// 2. Read the scheme -> its score_scale + composite method.
	scheme, err := uc.readScoringScheme(ctx, schemeID)
	if err != nil {
		return nil, err
	}
	if scheme.ScoreScaleId == nil || *scheme.ScoreScaleId == "" {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.scheme_no_scale",
			"[ERR-DEFAULT] scoring scheme %s has no score_scale_id"), schemeID)
	}

	// 3. The criteria in scope for this scheme (the component<->criteria junction).
	inScope, err := uc.inScopeCriteria(ctx, schemeID)
	if err != nil {
		return nil, err
	}
	if len(inScope) == 0 {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.no_scoped_criteria",
			"[ERR-DEFAULT] scoring scheme %s has no scoped criteria (scoring_component_criteria empty)"), schemeID)
	}

	// 4. Read the phase's recorded outcomes; bucket numeric values by criterion.
	outcomes, err := uc.listOutcomes(ctx, req.JobPhaseId)
	if err != nil {
		return nil, err
	}
	inputs, contributing := bucketByCriterion(outcomes, inScope)
	if contributing == 0 {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.no_recorded_values",
			"[ERR-DEFAULT] no numeric task_outcome values recorded against the scoped criteria for job_phase %s"), req.JobPhaseId)
	}

	// 5. Composite method: only SUM is implemented today. Fail loud on any other
	// declared method (e.g. a future WEIGHTED_AVERAGE scheme) rather than silently
	// SUMming + mislabelling it — consistent with the rest of this use-case.
	if !gradecompute.IsSumMethod(scheme.CompositeMethod) {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.unsupported_composite_method",
			"[ERR-DEFAULT] scoring scheme %s declares composite_method %v; only SCORING_METHOD_SUM is implemented"), schemeID, scheme.CompositeMethod)
	}

	// Pure roll-up: within-criterion MAX -> SUM composite.
	rollUp := gradecompute.RollUpCriteria(inputs)

	// 6. Read the scale + its bands; transmute the composite to a grade band.
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

	band, err := gradecompute.Transmute(scale, bands, rollUp.Composite)
	if err != nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.transmute_failed",
			"[ERR-DEFAULT] transmute failed for job_phase %s: %w"), req.JobPhaseId, err)
	}
	scaledScore, scaledLabel := gradecompute.BandOutput(band)

	// 7. Upsert the phase_outcome_summary carrying the transmuted grade.
	summary, err := uc.upsertSummary(ctx, req, phase, schemeID, scheme.CompositeMethod, rollUp, scaledScore, scaledLabel)
	if err != nil {
		return nil, err
	}

	return &ComputePhaseOutcomeResponse{
		Summary:         summary,
		ScoringSchemeId: schemeID,
		Composite:       rollUp.Composite,
	}, nil
}

// bucketByCriterion buckets every recorded numeric task_outcome value by its
// outcome_criteria id, keeping only criteria that are in scope for the scheme.
//
// Modeling: task_outcome.criteria_version_id is the FK to outcome_criteria.id
// (the recorded outcome targets a specific criterion version), and
// scoring_component_criteria.outcome_criteria_id references the SAME
// outcome_criteria.id — so the bucket key is the outcome_criteria id and the
// two join directly. Only ACTIVE outcomes carrying a numeric_value contribute;
// a text/categorical/pass-fail outcome (no numeric_value) is skipped, and a
// criterion with no recorded numeric value contributes nothing (not a zero) per
// the gradecompute contract.
func bucketByCriterion(
	outcomes []*taskoutcomepb.TaskOutcome,
	inScope map[string]bool,
) (inputs []gradecompute.CriterionInput, contributing int) {
	values := make(map[string][]float64)
	for _, o := range outcomes {
		if o == nil || !o.Active {
			continue
		}
		critID := o.CriteriaVersionId
		if critID == "" || !inScope[critID] {
			continue
		}
		if o.NumericValue == nil {
			continue
		}
		values[critID] = append(values[critID], *o.NumericValue)
	}
	// Emit one CriterionInput per in-scope criterion (deterministic: iterate the
	// scope set). A criterion with no recorded values gets an empty Values slice,
	// which RollUpCriteria treats as non-contributing.
	for critID := range inScope {
		vs := values[critID]
		inputs = append(inputs, gradecompute.CriterionInput{CriterionID: critID, Values: vs})
		if len(vs) > 0 {
			contributing++
		}
	}
	return inputs, contributing
}

// --- repository reads (each fails loud on a missing required row) ---

func (uc *ComputePhaseOutcomeUseCase) readJobPhase(ctx context.Context, id string) (*jobphasepb.JobPhase, error) {
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

func (uc *ComputePhaseOutcomeUseCase) readJobTemplatePhase(ctx context.Context, id string) (*jobtemplatephasepb.JobTemplatePhase, error) {
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

func (uc *ComputePhaseOutcomeUseCase) readScoringScheme(ctx context.Context, id string) (*scoringschemepb.ScoringScheme, error) {
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

func (uc *ComputePhaseOutcomeUseCase) readScoreScale(ctx context.Context, id string) (*scorescalepb.ScoreScale, error) {
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

// inScopeCriteria returns the set of outcome_criteria ids the scheme grades on,
// drawn from the scoring_component_criteria junction filtered to this scheme.
func (uc *ComputePhaseOutcomeUseCase) inScopeCriteria(ctx context.Context, schemeID string) (map[string]bool, error) {
	resp, err := uc.repositories.ScoringComponentCriteria.ListScoringComponentCriterias(ctx,
		&scoringcomponentcriteriapb.ListScoringComponentCriteriasRequest{})
	if err != nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.list_scoped_criteria_failed",
			"[ERR-DEFAULT] failed to list scoring_component_criteria for scheme %s: %w"), schemeID, err)
	}
	set := make(map[string]bool)
	if resp != nil {
		for _, scc := range resp.Data {
			if scc == nil || !scc.Active {
				continue
			}
			if scc.ScoringSchemeId != schemeID {
				continue
			}
			if scc.OutcomeCriteriaId != "" {
				set[scc.OutcomeCriteriaId] = true
			}
		}
	}
	return set, nil
}

func (uc *ComputePhaseOutcomeUseCase) listOutcomes(ctx context.Context, jobPhaseID string) ([]*taskoutcomepb.TaskOutcome, error) {
	resp, err := uc.repositories.TaskOutcome.ListByJobPhase(ctx, &taskoutcomepb.ListTaskOutcomesByJobPhaseRequest{
		JobPhaseId: jobPhaseID,
	})
	if err != nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.list_outcomes_failed",
			"[ERR-DEFAULT] failed to list task_outcomes for job_phase %s: %w"), jobPhaseID, err)
	}
	if resp == nil {
		return nil, nil
	}
	return resp.TaskOutcomes, nil
}

func (uc *ComputePhaseOutcomeUseCase) listBands(ctx context.Context, scaleID string) ([]*scorescalebandpb.ScoreScaleBand, error) {
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

// upsertSummary writes the transmuted grade onto phase_outcome_summary,
// updating an existing summary for this phase when one exists (idempotent
// re-grade) or creating a fresh one. Identity/dates are enriched via the
// IDGenerator + clock, mirroring the sibling create use-case.
func (uc *ComputePhaseOutcomeUseCase) upsertSummary(
	ctx context.Context,
	req *ComputePhaseOutcomeRequest,
	phase *jobphasepb.JobPhase,
	schemeID string,
	compositeMethod enumspb.ScoringMethod,
	rollUp gradecompute.RollUp,
	scaledScore float64,
	scaledLabel string,
) (*phaseoutcomesummarypb.PhaseOutcomeSummary, error) {
	now := time.Now()
	composite := rollUp.Composite
	score := scaledScore
	label := scaledLabel

	// Look for an existing summary to update (re-grade) rather than duplicate.
	existing, _ := uc.repositories.PhaseOutcomeSummary.GetByJobPhase(ctx,
		&phaseoutcomesummarypb.GetPhaseOutcomeSummaryByJobPhaseRequest{JobPhaseId: req.JobPhaseId})

	// NOTE: phase_outcome_summary has no scoring_scheme_id column, so the resolved
	// scheme id cannot be snapshotted onto the row (it is returned in the response
	// for traceability). The scheme's composite_method IS snapshotted onto
	// ScoringMethod (guaranteed SUM by the guard above). If a durable scheme-id
	// snapshot is required, add scoring_scheme_id to the proto + migration.
	data := &phaseoutcomesummarypb.PhaseOutcomeSummary{
		JobPhaseId:         req.JobPhaseId,
		JobId:              phase.JobId,
		SummaryType:        enumspb.SummaryType_SUMMARY_TYPE_ACADEMIC_RECORD,
		ScoringMethod:      compositeMethod,
		SummaryScore:       &composite,
		TotalCriteriaCount: int32(len(rollUp.PerCriterion)),
		ScaledScore:        &score,
		ScaledLabel:        &label,
		Active:             true,
		DateModified:       ptrInt64(now.UnixMilli()),
		DateModifiedString: ptrString(now.Format(time.RFC3339)),
	}
	if req.ReportingCheckpointId != "" {
		ckpt := req.ReportingCheckpointId
		data.ReportingCheckpointId = &ckpt
	}

	if existing != nil && existing.PhaseOutcomeSummary != nil && existing.PhaseOutcomeSummary.Id != "" {
		prev := existing.PhaseOutcomeSummary
		data.Id = prev.Id
		data.DateCreated = prev.DateCreated
		data.DateCreatedString = prev.DateCreatedString
		if data.IssuedBy == "" {
			data.IssuedBy = prev.IssuedBy
		}
		resp, err := uc.repositories.PhaseOutcomeSummary.UpdatePhaseOutcomeSummary(ctx,
			&phaseoutcomesummarypb.UpdatePhaseOutcomeSummaryRequest{Data: data})
		if err != nil {
			return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.update_summary_failed",
				"[ERR-DEFAULT] failed to update phase_outcome_summary for job_phase %s: %w"), req.JobPhaseId, err)
		}
		return firstSummary(resp.GetData(), data), nil
	}

	data.Id = uc.services.IDGenerator.GenerateID()
	data.DateCreated = ptrInt64(now.UnixMilli())
	data.DateCreatedString = ptrString(now.Format(time.RFC3339))
	resp, err := uc.repositories.PhaseOutcomeSummary.CreatePhaseOutcomeSummary(ctx,
		&phaseoutcomesummarypb.CreatePhaseOutcomeSummaryRequest{Data: data})
	if err != nil {
		return nil, fmt.Errorf(uc.msg(ctx, "grade_compute.errors.create_summary_failed",
			"[ERR-DEFAULT] failed to create phase_outcome_summary for job_phase %s: %w"), req.JobPhaseId, err)
	}
	return firstSummary(resp.GetData(), data), nil
}

// firstSummary returns the first summary from a response slice, falling back to
// the enriched data we sent when the adapter echoes nothing.
func firstSummary(data []*phaseoutcomesummarypb.PhaseOutcomeSummary, sent *phaseoutcomesummarypb.PhaseOutcomeSummary) *phaseoutcomesummarypb.PhaseOutcomeSummary {
	if len(data) > 0 && data[0] != nil {
		return data[0]
	}
	return sent
}

func (uc *ComputePhaseOutcomeUseCase) msg(ctx context.Context, key, def string) string {
	return contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, key, def)
}

func ptrInt64(v int64) *int64    { return &v }
func ptrString(v string) *string { return &v }
