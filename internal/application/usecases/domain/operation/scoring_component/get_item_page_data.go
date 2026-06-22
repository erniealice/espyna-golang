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

type GetScoringComponentItemPageDataRepositories struct {
	ScoringComponent pb.ScoringComponentDomainServiceServer
}

type GetScoringComponentItemPageDataServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetScoringComponentItemPageDataUseCase struct {
	repositories GetScoringComponentItemPageDataRepositories
	services     GetScoringComponentItemPageDataServices
}

func NewGetScoringComponentItemPageDataUseCase(r GetScoringComponentItemPageDataRepositories, s GetScoringComponentItemPageDataServices) *GetScoringComponentItemPageDataUseCase {
	return &GetScoringComponentItemPageDataUseCase{repositories: r, services: s}
}

func (uc *GetScoringComponentItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetScoringComponentItemPageDataRequest) (*pb.GetScoringComponentItemPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoringComponent, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "scoring_component.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.ScoringComponent.GetScoringComponentItemPageData(ctx, req)
}
