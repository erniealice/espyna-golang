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

// query is the port the outcome-matrix use case reads from. It is exactly the
// GENERATED operationv1.OutcomeMatrixServiceServer interface (Q-PROTO-MODE:
// service{rpc}) — no hand-written port. The postgres adapter embeds
// UnimplementedOutcomeMatrixServiceServer and satisfies this directly; on
// mock/non-postgres builds the port is nil and Execute degrades to an empty,
// successful response.
type query = matrixpb.OutcomeMatrixServiceServer

// GetOutcomeMatrixRepositories groups infrastructure dependencies.
type GetOutcomeMatrixRepositories struct {
	Query query
}

// GetOutcomeMatrixServices groups application services.
type GetOutcomeMatrixServices struct {
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetOutcomeMatrixUseCase serves the principal-scoped outcome-score matrix for
// one job_template. It gatekeeps the read on task_outcome:list, fail-closed-
// downgrades an unauthorized scope=ALL request to scope=MINE (never errors on
// the downgrade), then delegates to the adapter port.
type GetOutcomeMatrixUseCase struct {
	repositories GetOutcomeMatrixRepositories
	services     GetOutcomeMatrixServices
}

// NewGetOutcomeMatrixUseCase wires the use case. Any dep may be nil; Execute
// degrades to an empty, successful response when the Query port is missing.
func NewGetOutcomeMatrixUseCase(
	repositories GetOutcomeMatrixRepositories,
	services GetOutcomeMatrixServices,
) *GetOutcomeMatrixUseCase {
	if services.Translator == nil {
		services.Translator = ports.NewNoOpTranslator()
	}
	return &GetOutcomeMatrixUseCase{repositories: repositories, services: services}
}

// Execute runs the matrix read.
//
//	(a) ActionGatekeeper.Check(task_outcome, list) — the base read gate.
//	(b) FAIL-CLOSED scope=ALL gate — honor scope=ALL only when job_template_id
//	    is set AND the acting principal passes the workspace:list authcheck
//	    (the exact capability a teacher-only role LACKS — same gate as the
//	    legacy grade_sheet superAdminScopeCode). Otherwise SILENTLY downgrade to
//	    scope=MINE (never error, never widen). scope=UNSPECIFIED is already
//	    fail-closed to MINE by the adapter (principalscope applies unless ALL).
//	(c) delegate to the adapter port.
func (uc *GetOutcomeMatrixUseCase) Execute(
	ctx context.Context,
	req *matrixpb.GetOutcomeMatrixRequest,
) (*matrixpb.GetOutcomeMatrixResponse, error) {
	// Unconditional: Check has a nil-receiver guard that DENIES, so a
	// mis-wired nil gatekeeper fails closed instead of skipping the gate.
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.TaskOutcome,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"outcome_matrix.validation.request_required",
			"outcome matrix request is required"))
	}

	if uc.repositories.Query == nil {
		// No provider registered (mock / non-postgres) — empty, successful.
		return &matrixpb.GetOutcomeMatrixResponse{
			JobTemplateId: req.GetJobTemplateId(),
			Success:       true,
		}, nil
	}

	// (b) Fail-closed scope=ALL downgrade. Honor ALL only when job_template_id
	// is set AND the workspace:list authcheck passes (the capability a
	// teacher-only role LACKS). Check is nil-receiver-safe (denies) → a nil
	// gatekeeper cannot prove the widen is authorized → downgrade. Never error
	// on the downgrade. The downgrade target is safe: the adapter fail-closes
	// non-ALL scope to ZERO rows for any non-staff principal.
	if req.GetScope() == matrixpb.OutcomeMatrixScope_OUTCOME_MATRIX_SCOPE_ALL {
		allowed := false
		if req.GetJobTemplateId() != "" {
			if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
				Entity: "workspace",
				Action: entityid.ActionList,
			}); err == nil {
				allowed = true
			}
		}
		if !allowed {
			req.Scope = matrixpb.OutcomeMatrixScope_OUTCOME_MATRIX_SCOPE_MINE
		}
	}

	return uc.repositories.Query.GetOutcomeMatrix(ctx, req)
}
