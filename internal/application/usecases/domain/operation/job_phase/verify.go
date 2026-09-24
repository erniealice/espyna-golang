package job_phase

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/shared/approvalctx"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

// VerifyJobPhaseApprovalUseCase transitions a full sheet FOR_REVIEW -> VERIFIED.
// Gate: job_phase:verify ({1,2,7}). Inside the locked transaction the adapter
// enforces workspace ancestry, the STAFF approval scope (workspace scope or a
// reviewer edge — plan 20260924-approval-role-workflow D3), uniform source
// state, separation of duties (job_phase:verify_own waives it — D4), and the
// hard-frozen gate.
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
		// Approval scope + policy capabilities (plan 20260924-approval-role-workflow
		// D3/D4), resolved fresh INSIDE the transaction and enforced by the adapter.
		dctx := approvalctx.WithScopeDecision(txCtx, resolveScopeDecision(txCtx, sa, true, false))
		res, err := uc.repositories.JobPhase.VerifyJobPhaseApproval(dctx, req)
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
