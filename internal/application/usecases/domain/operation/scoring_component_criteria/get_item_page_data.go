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

type GetScoringComponentCriteriaItemPageDataRepositories struct {
	ScoringComponentCriteria pb.ScoringComponentCriteriaDomainServiceServer
}

type GetScoringComponentCriteriaItemPageDataServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetScoringComponentCriteriaItemPageDataUseCase struct {
	repositories GetScoringComponentCriteriaItemPageDataRepositories
	services     GetScoringComponentCriteriaItemPageDataServices
}

func NewGetScoringComponentCriteriaItemPageDataUseCase(r GetScoringComponentCriteriaItemPageDataRepositories, s GetScoringComponentCriteriaItemPageDataServices) *GetScoringComponentCriteriaItemPageDataUseCase {
	return &GetScoringComponentCriteriaItemPageDataUseCase{repositories: r, services: s}
}

func (uc *GetScoringComponentCriteriaItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetScoringComponentCriteriaItemPageDataRequest) (*pb.GetScoringComponentCriteriaItemPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoringComponentCriteria, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "scoring_component_criteria.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.ScoringComponentCriteria.GetScoringComponentCriteriaItemPageData(ctx, req)
}
