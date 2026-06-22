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

type GetScoringSchemeItemPageDataRepositories struct {
	ScoringScheme pb.ScoringSchemeDomainServiceServer
}

type GetScoringSchemeItemPageDataServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetScoringSchemeItemPageDataUseCase struct {
	repositories GetScoringSchemeItemPageDataRepositories
	services     GetScoringSchemeItemPageDataServices
}

func NewGetScoringSchemeItemPageDataUseCase(r GetScoringSchemeItemPageDataRepositories, s GetScoringSchemeItemPageDataServices) *GetScoringSchemeItemPageDataUseCase {
	return &GetScoringSchemeItemPageDataUseCase{repositories: r, services: s}
}

func (uc *GetScoringSchemeItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetScoringSchemeItemPageDataRequest) (*pb.GetScoringSchemeItemPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoringScheme, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "scoring_scheme.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.ScoringScheme.GetScoringSchemeItemPageData(ctx, req)
}
