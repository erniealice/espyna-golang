package evaluation

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	evaluationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation"
)

// ReadEvaluationRepositories groups all repository dependencies.
type ReadEvaluationRepositories struct {
	Evaluation evaluationpb.EvaluationDomainServiceServer
}

// ReadEvaluationServices groups all business service dependencies.
type ReadEvaluationServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadEvaluationUseCase reads a single evaluation. The workspace_id + client_id +
// visibility_type IDOR scoping is enforced in the adapter's query predicate
// (item page data path); this use case applies the authcheck gate.
type ReadEvaluationUseCase struct {
	repositories ReadEvaluationRepositories
	services     ReadEvaluationServices
}

func NewReadEvaluationUseCase(repositories ReadEvaluationRepositories, services ReadEvaluationServices) *ReadEvaluationUseCase {
	return &ReadEvaluationUseCase{repositories: repositories, services: services}
}

func (uc *ReadEvaluationUseCase) Execute(ctx context.Context, req *evaluationpb.ReadEvaluationRequest) (*evaluationpb.ReadEvaluationResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.Evaluation, entityid.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "evaluation.validation.id_required", "Evaluation ID is required [DEFAULT]"))
	}
	return uc.repositories.Evaluation.ReadEvaluation(ctx, req)
}
