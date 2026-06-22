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

type ReadScoringSchemeRepositories struct {
	ScoringScheme pb.ScoringSchemeDomainServiceServer
}

type ReadScoringSchemeServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ReadScoringSchemeUseCase struct {
	repositories ReadScoringSchemeRepositories
	services     ReadScoringSchemeServices
}

func NewReadScoringSchemeUseCase(r ReadScoringSchemeRepositories, s ReadScoringSchemeServices) *ReadScoringSchemeUseCase {
	return &ReadScoringSchemeUseCase{repositories: r, services: s}
}

func (uc *ReadScoringSchemeUseCase) Execute(ctx context.Context, req *pb.ReadScoringSchemeRequest) (*pb.ReadScoringSchemeResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoringScheme, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "scoring_scheme.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.ScoringScheme.ReadScoringScheme(ctx, req)
}
