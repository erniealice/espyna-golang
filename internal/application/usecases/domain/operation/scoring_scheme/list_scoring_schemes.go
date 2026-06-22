package scoring_scheme

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_scheme"
)

type ListScoringSchemesRepositories struct {
	ScoringScheme pb.ScoringSchemeDomainServiceServer
}

type ListScoringSchemesServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ListScoringSchemesUseCase struct {
	repositories ListScoringSchemesRepositories
	services     ListScoringSchemesServices
}

func NewListScoringSchemesUseCase(r ListScoringSchemesRepositories, s ListScoringSchemesServices) *ListScoringSchemesUseCase {
	return &ListScoringSchemesUseCase{repositories: r, services: s}
}

func (uc *ListScoringSchemesUseCase) Execute(ctx context.Context, req *pb.ListScoringSchemesRequest) (*pb.ListScoringSchemesResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoringScheme, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "scoring_scheme.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.ScoringScheme.ListScoringSchemes(ctx, req)
}
