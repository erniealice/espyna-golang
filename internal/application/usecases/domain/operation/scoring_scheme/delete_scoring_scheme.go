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

type DeleteScoringSchemeRepositories struct {
	ScoringScheme pb.ScoringSchemeDomainServiceServer
}

type DeleteScoringSchemeServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type DeleteScoringSchemeUseCase struct {
	repositories DeleteScoringSchemeRepositories
	services     DeleteScoringSchemeServices
}

func NewDeleteScoringSchemeUseCase(r DeleteScoringSchemeRepositories, s DeleteScoringSchemeServices) *DeleteScoringSchemeUseCase {
	return &DeleteScoringSchemeUseCase{repositories: r, services: s}
}

func (uc *DeleteScoringSchemeUseCase) Execute(ctx context.Context, req *pb.DeleteScoringSchemeRequest) (*pb.DeleteScoringSchemeResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoringScheme, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "scoring_scheme.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.ScoringScheme.DeleteScoringScheme(ctx, req)
}
