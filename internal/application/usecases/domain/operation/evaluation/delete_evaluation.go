package evaluation

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	evaluationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation"
)

// DeleteEvaluationRepositories groups all repository dependencies.
type DeleteEvaluationRepositories struct {
	Evaluation evaluationpb.EvaluationDomainServiceServer
}

// DeleteEvaluationServices groups all business service dependencies.
type DeleteEvaluationServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteEvaluationUseCase soft-deletes an evaluation.
type DeleteEvaluationUseCase struct {
	repositories DeleteEvaluationRepositories
	services     DeleteEvaluationServices
}

func NewDeleteEvaluationUseCase(repositories DeleteEvaluationRepositories, services DeleteEvaluationServices) *DeleteEvaluationUseCase {
	return &DeleteEvaluationUseCase{repositories: repositories, services: services}
}

func (uc *DeleteEvaluationUseCase) Execute(ctx context.Context, req *evaluationpb.DeleteEvaluationRequest) (*evaluationpb.DeleteEvaluationResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Evaluation,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.validation.id_required", "Evaluation ID is required [DEFAULT]"))
	}
	return uc.repositories.Evaluation.DeleteEvaluation(ctx, req)
}
