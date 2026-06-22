package scoring_component

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_component"
)

type ListScoringComponentsRepositories struct {
	ScoringComponent pb.ScoringComponentDomainServiceServer
}

type ListScoringComponentsServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ListScoringComponentsUseCase struct {
	repositories ListScoringComponentsRepositories
	services     ListScoringComponentsServices
}

func NewListScoringComponentsUseCase(r ListScoringComponentsRepositories, s ListScoringComponentsServices) *ListScoringComponentsUseCase {
	return &ListScoringComponentsUseCase{repositories: r, services: s}
}

func (uc *ListScoringComponentsUseCase) Execute(ctx context.Context, req *pb.ListScoringComponentsRequest) (*pb.ListScoringComponentsResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoringComponent, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "scoring_component.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.ScoringComponent.ListScoringComponents(ctx, req)
}
