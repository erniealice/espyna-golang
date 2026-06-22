package reporting_checkpoint

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/reporting_checkpoint"
)

type ReadReportingCheckpointRepositories struct {
	ReportingCheckpoint pb.ReportingCheckpointDomainServiceServer
}

type ReadReportingCheckpointServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ReadReportingCheckpointUseCase struct {
	repositories ReadReportingCheckpointRepositories
	services     ReadReportingCheckpointServices
}

func NewReadReportingCheckpointUseCase(r ReadReportingCheckpointRepositories, s ReadReportingCheckpointServices) *ReadReportingCheckpointUseCase {
	return &ReadReportingCheckpointUseCase{repositories: r, services: s}
}

func (uc *ReadReportingCheckpointUseCase) Execute(ctx context.Context, req *pb.ReadReportingCheckpointRequest) (*pb.ReadReportingCheckpointResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ReportingCheckpoint, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "reporting_checkpoint.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.ReportingCheckpoint.ReadReportingCheckpoint(ctx, req)
}
