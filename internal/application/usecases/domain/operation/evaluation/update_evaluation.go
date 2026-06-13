package evaluation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	evaluationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation"
)

// UpdateEvaluationRepositories groups all repository dependencies.
type UpdateEvaluationRepositories struct {
	Evaluation evaluationpb.EvaluationDomainServiceServer
}

// UpdateEvaluationServices groups all business service dependencies.
type UpdateEvaluationServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateEvaluationUseCase updates a DRAFT evaluation's editable fields. The
// SUBMITTED→SIGNED_OFF and DRAFT→SUBMITTED transitions are owned by the dedicated
// SignOffEvaluation / SubmitEvaluation use cases — this CRUD update must not be
// used to move status forward, so client_id is never re-stamped here.
type UpdateEvaluationUseCase struct {
	repositories UpdateEvaluationRepositories
	services     UpdateEvaluationServices
}

func NewUpdateEvaluationUseCase(repositories UpdateEvaluationRepositories, services UpdateEvaluationServices) *UpdateEvaluationUseCase {
	return &UpdateEvaluationUseCase{repositories: repositories, services: services}
}

func (uc *UpdateEvaluationUseCase) Execute(ctx context.Context, req *evaluationpb.UpdateEvaluationRequest) (*evaluationpb.UpdateEvaluationResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Evaluation,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.validation.id_required", "Evaluation ID is required [DEFAULT]"))
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	resp, err := uc.repositories.Evaluation.UpdateEvaluation(ctx, req)
	if err != nil {
		translated := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.errors.update_failed", "Evaluation update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translated, err)
	}
	return resp, nil
}
