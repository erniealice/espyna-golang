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

type GetReportingCheckpointItemPageDataRepositories struct {
	ReportingCheckpoint pb.ReportingCheckpointDomainServiceServer
}

type GetReportingCheckpointItemPageDataServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetReportingCheckpointItemPageDataUseCase struct {
	repositories GetReportingCheckpointItemPageDataRepositories
	services     GetReportingCheckpointItemPageDataServices
}

func NewGetReportingCheckpointItemPageDataUseCase(r GetReportingCheckpointItemPageDataRepositories, s GetReportingCheckpointItemPageDataServices) *GetReportingCheckpointItemPageDataUseCase {
	return &GetReportingCheckpointItemPageDataUseCase{repositories: r, services: s}
}

func (uc *GetReportingCheckpointItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetReportingCheckpointItemPageDataRequest) (*pb.GetReportingCheckpointItemPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ReportingCheckpoint, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "reporting_checkpoint.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.ReportingCheckpoint.GetReportingCheckpointItemPageData(ctx, req)
}
