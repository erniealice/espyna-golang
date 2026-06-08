package evaluation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	evaluationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation"
)

// SignOffEvaluationRequest is the Go-shaped input. Exactly one signer arc must be
// provided (operator workspace_user OR client portal grant), matching the
// status-coupling DB CHECKs.
type SignOffEvaluationRequest struct {
	EvaluationID string
	// Exactly one of the two signer arcs (num_nonnulls = 1 at the DB).
	SignedOffByWorkspaceUserID     string
	SignedOffByClientPortalGrantID string
}

// SignOffEvaluationRepositories groups all repository dependencies.
type SignOffEvaluationRepositories struct {
	Evaluation evaluationpb.EvaluationDomainServiceServer
}

// SignOffEvaluationServices groups all business service dependencies.
type SignOffEvaluationServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// SignOffEvaluationUseCase moves SUBMITTED→SIGNED_OFF (Q-SIGNOFF-1). It stamps
// exactly one signer + signed_off_at; the status-coupling DB CHECKs reject a
// SIGNED_OFF row missing the actor/timestamp. It cannot re-open to DRAFT. The
// is_owner gate (Q-SERVICING-SCOPE-1 / CR-5): a client signer must own the
// evaluation (client_id == acting_as_client_id); the operator signer arc is
// permission-gated via authcheck.
type SignOffEvaluationUseCase struct {
	repositories SignOffEvaluationRepositories
	services     SignOffEvaluationServices
}

func NewSignOffEvaluationUseCase(repositories SignOffEvaluationRepositories, services SignOffEvaluationServices) *SignOffEvaluationUseCase {
	return &SignOffEvaluationUseCase{repositories: repositories, services: services}
}

func (uc *SignOffEvaluationUseCase) Execute(ctx context.Context, req *SignOffEvaluationRequest) (*evaluationpb.UpdateEvaluationResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.Evaluation, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.EvaluationID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.validation.id_required", "Evaluation ID is required [DEFAULT]"))
	}

	// Exactly one signer arc (mirrors the DB num_nonnulls=1 CHECK).
	hasWorkspace := req.SignedOffByWorkspaceUserID != ""
	hasClient := req.SignedOffByClientPortalGrantID != ""
	if hasWorkspace == hasClient {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.signer_arc", "Exactly one sign-off actor is required [DEFAULT]"))
	}

	readResp, err := uc.repositories.Evaluation.ReadEvaluation(ctx, &evaluationpb.ReadEvaluationRequest{
		Data: &evaluationpb.Evaluation{Id: req.EvaluationID},
	})
	if err != nil {
		return nil, err
	}
	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.not_found", "Evaluation not found [DEFAULT]"))
	}
	eval := readResp.Data[0]

	// Only SUBMITTED may be signed off (no re-open to DRAFT).
	if eval.Status != evaluationpb.EvaluationStatus_EVALUATION_STATUS_SUBMITTED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.not_submitted", "Only a submitted evaluation can be signed off [DEFAULT]"))
	}

	// is_owner gate for the client signer arc (fail-closed).
	if hasClient {
		actingClient := contextutil.GetActingAsClientIDFromContext(ctx)
		if actingClient == "" || eval.ClientId != actingClient {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.not_owned", "This evaluation is not owned by the acting client [DEFAULT]"))
		}
	}

	now := time.Now()
	eval.Status = evaluationpb.EvaluationStatus_EVALUATION_STATUS_SIGNED_OFF
	eval.SignedOffAt = &[]int64{now.UnixMilli()}[0]
	if hasWorkspace {
		v := req.SignedOffByWorkspaceUserID
		eval.SignedOffByWorkspaceUserId = &v
		eval.SignedOffByClientPortalGrantId = nil
	} else {
		v := req.SignedOffByClientPortalGrantID
		eval.SignedOffByClientPortalGrantId = &v
		eval.SignedOffByWorkspaceUserId = nil
	}
	eval.DateModified = &[]int64{now.UnixMilli()}[0]
	eval.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	resp, err := uc.repositories.Evaluation.UpdateEvaluation(ctx, &evaluationpb.UpdateEvaluationRequest{Data: eval})
	if err != nil {
		translated := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.signoff_failed", "Evaluation sign-off failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translated, err)
	}
	return resp, nil
}
