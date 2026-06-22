package scoring_component

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_component"
)

type CreateScoringComponentRepositories struct {
	ScoringComponent pb.ScoringComponentDomainServiceServer
}

type CreateScoringComponentServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreateScoringComponentUseCase struct {
	repositories CreateScoringComponentRepositories
	services     CreateScoringComponentServices
}

func NewCreateScoringComponentUseCase(r CreateScoringComponentRepositories, s CreateScoringComponentServices) *CreateScoringComponentUseCase {
	return &CreateScoringComponentUseCase{repositories: r, services: s}
}

func (uc *CreateScoringComponentUseCase) Execute(ctx context.Context, req *pb.CreateScoringComponentRequest) (*pb.CreateScoringComponentResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoringComponent, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "scoring_component.validation.data_required", "Data is required [DEFAULT]"))
	}
	uc.enrich(req.Data)
	return uc.repositories.ScoringComponent.CreateScoringComponent(ctx, req)
}

func (uc *CreateScoringComponentUseCase) enrich(data *pb.ScoringComponent) {
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
