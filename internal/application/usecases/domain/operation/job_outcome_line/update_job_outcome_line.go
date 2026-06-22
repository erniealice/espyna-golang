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

type UpdateJobOutcomeLineRepositories struct {
	JobOutcomeLine pb.JobOutcomeLineDomainServiceServer
}

type UpdateJobOutcomeLineServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdateJobOutcomeLineUseCase struct {
	repositories UpdateJobOutcomeLineRepositories
	services     UpdateJobOutcomeLineServices
}

func NewUpdateJobOutcomeLineUseCase(r UpdateJobOutcomeLineRepositories, s UpdateJobOutcomeLineServices) *UpdateJobOutcomeLineUseCase {
	return &UpdateJobOutcomeLineUseCase{repositories: r, services: s}
}

func (uc *UpdateJobOutcomeLineUseCase) Execute(ctx context.Context, req *pb.UpdateJobOutcomeLineRequest) (*pb.UpdateJobOutcomeLineResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobOutcomeLine, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_line.validation.data_required", "Data is required [DEFAULT]"))
	}
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s
	return uc.repositories.JobOutcomeLine.UpdateJobOutcomeLine(ctx, req)
}
