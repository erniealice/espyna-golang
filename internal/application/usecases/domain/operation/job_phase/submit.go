package job_phase

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/shared/approvalctx"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

// SubmitJobPhaseApprovalUseCase transitions a full sheet IN_PROGRESS -> FOR_REVIEW.
//
// Authorization (plan §4.2 / codex "Exact partial-reachability result"):
//   - Gate: job_phase:submit (the {1,2,7} staff-tagged capability).
//   - Scope: unless the caller has the D4 admin override (proven
//     job_phase:publish authority), the adapter enforces the strict D7 all-task
//     staff-ownership check over the FULL locked sheet. The override is resolved
//     here (application-layer RBAC) and threaded to the adapter via approvalctx,
//     because the ownership decision must be honoured INSIDE the locked
//     transition transaction.
type SubmitJobPhaseApprovalUseCase struct {
	repositories transitionRepositories
	services     transitionServices
}

func (uc *SubmitJobPhaseApprovalUseCase) Execute(ctx context.Context, req *pb.SubmitJobPhaseApprovalRequest) (*pb.SubmitJobPhaseApprovalResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_phase", Action: actionSubmit}); err != nil {
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

	// Strict authorization (codex FIX-FIRST 1): refuse unless AUTHZ_ENFORCE is
	// active, then re-check job_phase:submit with a deny-capable verdict (the
	// shadow-mode ActionGatekeeper.Check above allows-on-deny).
	sa, err := requireStrictAuthorizer(uc.services.Authorizer)
	if err != nil {
		return nil, err
	}
	if err := requireStrictVerb(ctx, sa, actionSubmit); err != nil {
		return nil, err
	}

	var result *pb.SubmitJobPhaseApprovalResponse
	err = uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// AUTHORITATIVE verb gate (codex P3 §A1): re-check job_phase:submit with a
		// FRESH, cache-bypassing, ambient-tx verdict INSIDE the transaction, so a
		// revocation committed after the pre-filter denies the transition.
		if err := requireStrictVerbFresh(txCtx, sa, actionSubmit); err != nil {
			return err
		}
		// Resolve the admin override INSIDE the transaction so a permission
		// revocation between request receipt and commit denies it too (codex §1
		// MEDIUM: no pre-tx allow bit). Fresh verdict — no shadow allow-on-deny,
		// no stale cache.
		adminOverride := resolveAdminOverride(txCtx, sa)
		dctx := approvalctx.WithSubmitDecision(txCtx, approvalctx.SubmitDecision{AdminOverride: adminOverride})
		res, err := uc.repositories.JobPhase.SubmitJobPhaseApproval(dctx, req)
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
