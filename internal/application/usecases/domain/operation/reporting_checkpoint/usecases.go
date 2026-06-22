package reporting_checkpoint

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/reporting_checkpoint"
)

// UseCases aggregates the reporting_checkpoint CRUD + page-data use cases.
type UseCases struct {
	CreateReportingCheckpoint          *CreateReportingCheckpointUseCase
	ReadReportingCheckpoint            *ReadReportingCheckpointUseCase
	UpdateReportingCheckpoint          *UpdateReportingCheckpointUseCase
	DeleteReportingCheckpoint          *DeleteReportingCheckpointUseCase
	ListReportingCheckpoints           *ListReportingCheckpointsUseCase
	GetReportingCheckpointListPageData *GetReportingCheckpointListPageDataUseCase
	GetReportingCheckpointItemPageData *GetReportingCheckpointItemPageDataUseCase
}

// Repositories groups the primary repository dependency.
type Repositories struct {
	ReportingCheckpoint pb.ReportingCheckpointDomainServiceServer
}

// Services groups the shared business-service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// NewUseCases wires the reporting_checkpoint use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.ReportingCheckpoint
	return &UseCases{
		CreateReportingCheckpoint:          NewCreateReportingCheckpointUseCase(CreateReportingCheckpointRepositories{ReportingCheckpoint: repo}, CreateReportingCheckpointServices(s)),
		ReadReportingCheckpoint:            NewReadReportingCheckpointUseCase(ReadReportingCheckpointRepositories{ReportingCheckpoint: repo}, ReadReportingCheckpointServices(s)),
		UpdateReportingCheckpoint:          NewUpdateReportingCheckpointUseCase(UpdateReportingCheckpointRepositories{ReportingCheckpoint: repo}, UpdateReportingCheckpointServices(s)),
		DeleteReportingCheckpoint:          NewDeleteReportingCheckpointUseCase(DeleteReportingCheckpointRepositories{ReportingCheckpoint: repo}, DeleteReportingCheckpointServices(s)),
		ListReportingCheckpoints:           NewListReportingCheckpointsUseCase(ListReportingCheckpointsRepositories{ReportingCheckpoint: repo}, ListReportingCheckpointsServices(s)),
		GetReportingCheckpointListPageData: NewGetReportingCheckpointListPageDataUseCase(GetReportingCheckpointListPageDataRepositories{ReportingCheckpoint: repo}, GetReportingCheckpointListPageDataServices(s)),
		GetReportingCheckpointItemPageData: NewGetReportingCheckpointItemPageDataUseCase(GetReportingCheckpointItemPageDataRepositories{ReportingCheckpoint: repo}, GetReportingCheckpointItemPageDataServices(s)),
	}
}
