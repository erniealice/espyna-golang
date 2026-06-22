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

type UpdateScoringComponentRepositories struct {
	ScoringComponent pb.ScoringComponentDomainServiceServer
}

type UpdateScoringComponentServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdateScoringComponentUseCase struct {
	repositories UpdateScoringComponentRepositories
	services     UpdateScoringComponentServices
}

func NewUpdateScoringComponentUseCase(r UpdateScoringComponentRepositories, s UpdateScoringComponentServices) *UpdateScoringComponentUseCase {
	return &UpdateScoringComponentUseCase{repositories: r, services: s}
}

func (uc *UpdateScoringComponentUseCase) Execute(ctx context.Context, req *pb.UpdateScoringComponentRequest) (*pb.UpdateScoringComponentResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoringComponent, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "scoring_component.validation.data_required", "Data is required [DEFAULT]"))
	}
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s
	return uc.repositories.ScoringComponent.UpdateScoringComponent(ctx, req)
}
