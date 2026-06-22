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

type UpdateReportingCheckpointRepositories struct {
	ReportingCheckpoint pb.ReportingCheckpointDomainServiceServer
}

type UpdateReportingCheckpointServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdateReportingCheckpointUseCase struct {
	repositories UpdateReportingCheckpointRepositories
	services     UpdateReportingCheckpointServices
}

func NewUpdateReportingCheckpointUseCase(r UpdateReportingCheckpointRepositories, s UpdateReportingCheckpointServices) *UpdateReportingCheckpointUseCase {
	return &UpdateReportingCheckpointUseCase{repositories: r, services: s}
}

func (uc *UpdateReportingCheckpointUseCase) Execute(ctx context.Context, req *pb.UpdateReportingCheckpointRequest) (*pb.UpdateReportingCheckpointResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ReportingCheckpoint, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "reporting_checkpoint.validation.data_required", "Data is required [DEFAULT]"))
	}
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s
	return uc.repositories.ReportingCheckpoint.UpdateReportingCheckpoint(ctx, req)
}
