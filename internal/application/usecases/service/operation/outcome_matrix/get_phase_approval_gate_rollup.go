package outcome_matrix

import (
	"context"
	"errors"

	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

// GetPhaseApprovalGateRollupRepositories groups infrastructure dependencies.
type GetPhaseApprovalGateRollupRepositories struct {
	Query query
}

// GetPhaseApprovalGateRollupServices groups application services.
type GetPhaseApprovalGateRollupServices struct {
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetPhaseApprovalGateRollupUseCase serves the document render gate's
// group-grain input read (plan 20260729 Phase 2): per requested template phase,
// the gate inputs (target_count / any_workflow_entered / all_published /
// has_data) plus the exact applied-group echo, derived over the group-narrowed
// sheet by the provider.
//
// This is a DOCUMENT-INTEGRITY input, so every step is fail-closed — including
// the nil-port case, which deliberately DIVERGES from GetOutcomeMatrixUseCase's
// empty-success degrade (see Execute).
type GetPhaseApprovalGateRollupUseCase struct {
	repositories GetPhaseApprovalGateRollupRepositories
	services     GetPhaseApprovalGateRollupServices
}

// NewGetPhaseApprovalGateRollupUseCase wires the use case.
func NewGetPhaseApprovalGateRollupUseCase(
	repositories GetPhaseApprovalGateRollupRepositories,
	services GetPhaseApprovalGateRollupServices,
) *GetPhaseApprovalGateRollupUseCase {
	if services.Translator == nil {
		services.Translator = ports.NewNoOpTranslator()
	}
	return &GetPhaseApprovalGateRollupUseCase{repositories: repositories, services: services}
}

// Execute runs the gate-input read. Order (each step fail-closed):
//
//	(a) ActionGatekeeper.Check on job_phase:list, then job_task:list, then
//	    task_outcome:list. The RPC collapses the gate's former three-entity
//	    read walk (job_phase → job_task → task_outcome), so it must require
//	    the SAME three list capabilities — anything less silently WIDENS what
//	    a partially-permissioned role can learn (sheet state / data
//	    presence). Check has a nil-receiver guard that DENIES, so a mis-wired
//	    nil gatekeeper fails closed instead of skipping the gates.
//	(b) validation: nil request / empty subscription_group_id / empty
//	    job_template_phase_ids → error.
//	(c) nil Query port → ERROR, never an empty success. DELIBERATE divergence
//	    from GetOutcomeMatrixUseCase (which degrades a nil port to an empty
//	    success): this response feeds a document-integrity decision, and an
//	    empty success would read as "no sheets" — the locked design mandates
//	    configured-but-unprovable ⇒ failure at the consumer, so the missing
//	    provider must be observable as an error here.
//	(d) delegate to the provider unchanged.
//
// No scope enum, no widen logic — the read is already maximally narrow (one
// validated group).
func (uc *GetPhaseApprovalGateRollupUseCase) Execute(
	ctx context.Context,
	req *matrixpb.GetPhaseApprovalGateRollupRequest,
) (*matrixpb.GetPhaseApprovalGateRollupResponse, error) {
	for _, entity := range []string{entityid.JobPhase, entityid.JobTask, entityid.TaskOutcome} {
		if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
			Entity: entity,
			Action: entityid.ActionList,
		}); err != nil {
			return nil, err
		}
	}

	if req == nil || req.GetSubscriptionGroupId() == "" || len(req.GetJobTemplatePhaseIds()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"outcome_matrix.validation.gate_rollup_request_invalid",
			"phase approval gate rollup requires a subscription group id and at least one job template phase id"))
	}

	if uc.repositories.Query == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"outcome_matrix.errors.gate_rollup_provider_missing",
			"phase approval gate rollup provider not registered — fail closed"))
	}

	return uc.repositories.Query.GetPhaseApprovalGateRollup(ctx, req)
}
