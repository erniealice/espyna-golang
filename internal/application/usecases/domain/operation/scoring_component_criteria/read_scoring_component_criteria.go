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

type ReadScoringComponentCriteriaRepositories struct {
	ScoringComponentCriteria pb.ScoringComponentCriteriaDomainServiceServer
}

type ReadScoringComponentCriteriaServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ReadScoringComponentCriteriaUseCase struct {
	repositories ReadScoringComponentCriteriaRepositories
	services     ReadScoringComponentCriteriaServices
}

func NewReadScoringComponentCriteriaUseCase(r ReadScoringComponentCriteriaRepositories, s ReadScoringComponentCriteriaServices) *ReadScoringComponentCriteriaUseCase {
	return &ReadScoringComponentCriteriaUseCase{repositories: r, services: s}
}

func (uc *ReadScoringComponentCriteriaUseCase) Execute(ctx context.Context, req *pb.ReadScoringComponentCriteriaRequest) (*pb.ReadScoringComponentCriteriaResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoringComponentCriteria, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "scoring_component_criteria.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.ScoringComponentCriteria.ReadScoringComponentCriteria(ctx, req)
}
