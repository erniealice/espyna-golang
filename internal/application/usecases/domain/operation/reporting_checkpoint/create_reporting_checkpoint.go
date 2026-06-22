package reporting_checkpoint

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/reporting_checkpoint"
)

type CreateReportingCheckpointRepositories struct {
	ReportingCheckpoint pb.ReportingCheckpointDomainServiceServer
}

type CreateReportingCheckpointServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreateReportingCheckpointUseCase struct {
	repositories CreateReportingCheckpointRepositories
	services     CreateReportingCheckpointServices
}

func NewCreateReportingCheckpointUseCase(r CreateReportingCheckpointRepositories, s CreateReportingCheckpointServices) *CreateReportingCheckpointUseCase {
	return &CreateReportingCheckpointUseCase{repositories: r, services: s}
}

func (uc *CreateReportingCheckpointUseCase) Execute(ctx context.Context, req *pb.CreateReportingCheckpointRequest) (*pb.CreateReportingCheckpointResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ReportingCheckpoint, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "reporting_checkpoint.validation.data_required", "Data is required [DEFAULT]"))
	}
	uc.enrich(req.Data)
	return uc.repositories.ReportingCheckpoint.CreateReportingCheckpoint(ctx, req)
}

func (uc *CreateReportingCheckpointUseCase) enrich(data *pb.ReportingCheckpoint) {
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
