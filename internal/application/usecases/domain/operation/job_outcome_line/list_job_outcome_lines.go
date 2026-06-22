package job_outcome_line

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_line"
)

type ListJobOutcomeLinesRepositories struct {
	JobOutcomeLine pb.JobOutcomeLineDomainServiceServer
}

type ListJobOutcomeLinesServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ListJobOutcomeLinesUseCase struct {
	repositories ListJobOutcomeLinesRepositories
	services     ListJobOutcomeLinesServices
}

func NewListJobOutcomeLinesUseCase(r ListJobOutcomeLinesRepositories, s ListJobOutcomeLinesServices) *ListJobOutcomeLinesUseCase {
	return &ListJobOutcomeLinesUseCase{repositories: r, services: s}
}

func (uc *ListJobOutcomeLinesUseCase) Execute(ctx context.Context, req *pb.ListJobOutcomeLinesRequest) (*pb.ListJobOutcomeLinesResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobOutcomeLine, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_line.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.JobOutcomeLine.ListJobOutcomeLines(ctx, req)
}
