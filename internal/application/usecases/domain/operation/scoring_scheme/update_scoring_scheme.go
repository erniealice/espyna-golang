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

type UpdateScoringSchemeRepositories struct {
	ScoringScheme pb.ScoringSchemeDomainServiceServer
}

type UpdateScoringSchemeServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdateScoringSchemeUseCase struct {
	repositories UpdateScoringSchemeRepositories
	services     UpdateScoringSchemeServices
}

func NewUpdateScoringSchemeUseCase(r UpdateScoringSchemeRepositories, s UpdateScoringSchemeServices) *UpdateScoringSchemeUseCase {
	return &UpdateScoringSchemeUseCase{repositories: r, services: s}
}

func (uc *UpdateScoringSchemeUseCase) Execute(ctx context.Context, req *pb.UpdateScoringSchemeRequest) (*pb.UpdateScoringSchemeResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoringScheme, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "scoring_scheme.validation.data_required", "Data is required [DEFAULT]"))
	}
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s
	return uc.repositories.ScoringScheme.UpdateScoringScheme(ctx, req)
}
