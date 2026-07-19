package job_phase

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

// VerifyJobPhaseApprovalUseCase transitions a full sheet FOR_REVIEW -> VERIFIED.
// Operator-tier verb (D4: verify is {1,2}, no staff-ownership scope); the adapter
// enforces workspace ancestry, uniform source state, and the hard-frozen gate
// inside the locked transaction.
type VerifyJobPhaseApprovalUseCase struct {
	repositories transitionRepositories
	services     transitionServices
}

func (uc *VerifyJobPhaseApprovalUseCase) Execute(ctx context.Context, req *pb.VerifyJobPhaseApprovalRequest) (*pb.VerifyJobPhaseApprovalResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_phase", Action: actionVerify}); err != nil {
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
	if err := requireStrictVerb(ctx, sa, actionVerify); err != nil {
		return nil, err
	}

	var result *pb.VerifyJobPhaseApprovalResponse
	err = uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// AUTHORITATIVE verb gate (codex P3 §A1): fresh, cache-bypassing, ambient-tx
		// re-check of job_phase:verify INSIDE the transaction.
		if err := requireStrictVerbFresh(txCtx, sa, actionVerify); err != nil {
			return err
		}
		res, err := uc.repositories.JobPhase.VerifyJobPhaseApproval(txCtx, req)
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
