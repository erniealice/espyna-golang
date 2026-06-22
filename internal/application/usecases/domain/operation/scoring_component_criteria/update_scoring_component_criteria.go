package scoring_component_criteria

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_component_criteria"
)

type UpdateScoringComponentCriteriaRepositories struct {
	ScoringComponentCriteria pb.ScoringComponentCriteriaDomainServiceServer
}

type UpdateScoringComponentCriteriaServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdateScoringComponentCriteriaUseCase struct {
	repositories UpdateScoringComponentCriteriaRepositories
	services     UpdateScoringComponentCriteriaServices
}

func NewUpdateScoringComponentCriteriaUseCase(r UpdateScoringComponentCriteriaRepositories, s UpdateScoringComponentCriteriaServices) *UpdateScoringComponentCriteriaUseCase {
	return &UpdateScoringComponentCriteriaUseCase{repositories: r, services: s}
}

func (uc *UpdateScoringComponentCriteriaUseCase) Execute(ctx context.Context, req *pb.UpdateScoringComponentCriteriaRequest) (*pb.UpdateScoringComponentCriteriaResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoringComponentCriteria, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "scoring_component_criteria.validation.data_required", "Data is required [DEFAULT]"))
	}
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s
	return uc.repositories.ScoringComponentCriteria.UpdateScoringComponentCriteria(ctx, req)
}
