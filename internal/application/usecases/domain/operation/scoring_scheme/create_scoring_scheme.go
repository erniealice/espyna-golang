package scoring_scheme

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_scheme"
)

type CreateScoringSchemeRepositories struct {
	ScoringScheme pb.ScoringSchemeDomainServiceServer
}

type CreateScoringSchemeServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreateScoringSchemeUseCase struct {
	repositories CreateScoringSchemeRepositories
	services     CreateScoringSchemeServices
}

func NewCreateScoringSchemeUseCase(r CreateScoringSchemeRepositories, s CreateScoringSchemeServices) *CreateScoringSchemeUseCase {
	return &CreateScoringSchemeUseCase{repositories: r, services: s}
}

func (uc *CreateScoringSchemeUseCase) Execute(ctx context.Context, req *pb.CreateScoringSchemeRequest) (*pb.CreateScoringSchemeResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoringScheme, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "scoring_scheme.validation.data_required", "Data is required [DEFAULT]"))
	}
	uc.enrich(req.Data)
	return uc.repositories.ScoringScheme.CreateScoringScheme(ctx, req)
}

func (uc *CreateScoringSchemeUseCase) enrich(data *pb.ScoringScheme) {
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
