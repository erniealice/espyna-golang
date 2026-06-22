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

type CreateScoringComponentCriteriaRepositories struct {
	ScoringComponentCriteria pb.ScoringComponentCriteriaDomainServiceServer
}

type CreateScoringComponentCriteriaServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreateScoringComponentCriteriaUseCase struct {
	repositories CreateScoringComponentCriteriaRepositories
	services     CreateScoringComponentCriteriaServices
}

func NewCreateScoringComponentCriteriaUseCase(r CreateScoringComponentCriteriaRepositories, s CreateScoringComponentCriteriaServices) *CreateScoringComponentCriteriaUseCase {
	return &CreateScoringComponentCriteriaUseCase{repositories: r, services: s}
}

func (uc *CreateScoringComponentCriteriaUseCase) Execute(ctx context.Context, req *pb.CreateScoringComponentCriteriaRequest) (*pb.CreateScoringComponentCriteriaResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoringComponentCriteria, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "scoring_component_criteria.validation.data_required", "Data is required [DEFAULT]"))
	}
	uc.enrich(req.Data)
	return uc.repositories.ScoringComponentCriteria.CreateScoringComponentCriteria(ctx, req)
}

func (uc *CreateScoringComponentCriteriaUseCase) enrich(data *pb.ScoringComponentCriteria) {
	now := time.Now()
	if data.Id == "" && uc.services.IDGenerator != nil {
		data.Id = uc.services.IDGenerator.GenerateID()
	}
	data.Active = true
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	data.DateCreated = &ms
	data.DateCreatedString = &s
	data.DateModified = &ms
	data.DateModifiedString = &s
}
