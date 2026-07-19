package job_phase

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

// PublishJobPhaseApprovalUseCase transitions a full sheet VERIFIED -> PUBLISHED.
// Admin-tier verb (D4: publish is admin-only). The adapter enforces workspace
// ancestry, uniform VERIFIED source, and the hard-frozen gate inside the locked
// transaction.
type PublishJobPhaseApprovalUseCase struct {
	repositories transitionRepositories
	services     transitionServices
}

func (uc *PublishJobPhaseApprovalUseCase) Execute(ctx context.Context, req *pb.PublishJobPhaseApprovalRequest) (*pb.PublishJobPhaseApprovalResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_phase", Action: actionPublish}); err != nil {
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
	if err := requireStrictVerb(ctx, sa, actionPublish); err != nil {
		return nil, err
	}

	var result *pb.PublishJobPhaseApprovalResponse
	err = uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// AUTHORITATIVE verb gate (codex P3 §A1): fresh, cache-bypassing, ambient-tx
		// re-check of job_phase:publish INSIDE the transaction.
		if err := requireStrictVerbFresh(txCtx, sa, actionPublish); err != nil {
			return err
		}
		res, err := uc.repositories.JobPhase.PublishJobPhaseApproval(txCtx, req)
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
