package scoring_component_criteria

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_component_criteria"
)

type DeleteScoringComponentCriteriaRepositories struct {
	ScoringComponentCriteria pb.ScoringComponentCriteriaDomainServiceServer
}

type DeleteScoringComponentCriteriaServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type DeleteScoringComponentCriteriaUseCase struct {
	repositories DeleteScoringComponentCriteriaRepositories
	services     DeleteScoringComponentCriteriaServices
}

func NewDeleteScoringComponentCriteriaUseCase(r DeleteScoringComponentCriteriaRepositories, s DeleteScoringComponentCriteriaServices) *DeleteScoringComponentCriteriaUseCase {
	return &DeleteScoringComponentCriteriaUseCase{repositories: r, services: s}
}

func (uc *DeleteScoringComponentCriteriaUseCase) Execute(ctx context.Context, req *pb.DeleteScoringComponentCriteriaRequest) (*pb.DeleteScoringComponentCriteriaResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoringComponentCriteria, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "scoring_component_criteria.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.ScoringComponentCriteria.DeleteScoringComponentCriteria(ctx, req)
}
