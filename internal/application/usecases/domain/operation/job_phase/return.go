package job_phase

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

// ReturnJobPhaseApprovalUseCase is the mixed-state normalizer: it locks a
// recognized sheet with >= 1 non-IN_PROGRESS row and normalizes ALL of S back to
// IN_PROGRESS (plan §4.2 / codex "Exact E1 mixed-state recovery"). A non-blank
// reason is required if any locked row is or was PUBLISHED; the return is blocked
// entirely when the sheet is hard-frozen. All of this is enforced by the adapter
// inside the locked transaction; the use case gates job_phase:return and passes
// the (optional) reason through unchanged.
type ReturnJobPhaseApprovalUseCase struct {
	repositories transitionRepositories
	services     transitionServices
}

func (uc *ReturnJobPhaseApprovalUseCase) Execute(ctx context.Context, req *pb.ReturnJobPhaseApprovalRequest) (*pb.ReturnJobPhaseApprovalResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_phase", Action: actionReturn}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, validateSheetRequest("", "")
	}
	if err := validateSheetRequest(req.GetJobTemplateId(), req.GetJobTemplatePhaseId()); err != nil {
		return nil, err
	}
	if err := uc.services.requireTransaction(); err != nil {
		return nil, err
	}

	// Strict authorization (codex FIX-FIRST 1): shadow-mode RBAC allows-on-deny,
	// so refuse unless AUTHZ_ENFORCE is active and re-check the verb strictly.
	sa, err := requireStrictAuthorizer(uc.services.Authorizer)
	if err != nil {
		return nil, err
	}
	if err := requireStrictVerb(ctx, sa, actionReturn); err != nil {
		return nil, err
	}

	var result *pb.ReturnJobPhaseApprovalResponse
	err = uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// AUTHORITATIVE verb gate (codex P3 §A1): fresh, cache-bypassing, ambient-tx
		// re-check of job_phase:return INSIDE the transaction.
		if err := requireStrictVerbFresh(txCtx, sa, actionReturn); err != nil {
			return err
		}
		res, err := uc.repositories.JobPhase.ReturnJobPhaseApproval(txCtx, req)
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
