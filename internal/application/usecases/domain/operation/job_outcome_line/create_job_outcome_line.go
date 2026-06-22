package job_outcome_line

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_line"
)

type CreateJobOutcomeLineRepositories struct {
	JobOutcomeLine pb.JobOutcomeLineDomainServiceServer
}

type CreateJobOutcomeLineServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreateJobOutcomeLineUseCase struct {
	repositories CreateJobOutcomeLineRepositories
	services     CreateJobOutcomeLineServices
}

func NewCreateJobOutcomeLineUseCase(r CreateJobOutcomeLineRepositories, s CreateJobOutcomeLineServices) *CreateJobOutcomeLineUseCase {
	return &CreateJobOutcomeLineUseCase{repositories: r, services: s}
}

func (uc *CreateJobOutcomeLineUseCase) Execute(ctx context.Context, req *pb.CreateJobOutcomeLineRequest) (*pb.CreateJobOutcomeLineResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobOutcomeLine, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_line.validation.data_required", "Data is required [DEFAULT]"))
	}
	uc.enrich(req.Data)
	return uc.repositories.JobOutcomeLine.CreateJobOutcomeLine(ctx, req)
}

func (uc *CreateJobOutcomeLineUseCase) enrich(data *pb.JobOutcomeLine) {
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
