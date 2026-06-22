package job_outcome_line

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_line"
)

// UseCases aggregates the job_outcome_line CRUD + page-data use cases.
type UseCases struct {
	CreateJobOutcomeLine          *CreateJobOutcomeLineUseCase
	ReadJobOutcomeLine            *ReadJobOutcomeLineUseCase
	UpdateJobOutcomeLine          *UpdateJobOutcomeLineUseCase
	DeleteJobOutcomeLine          *DeleteJobOutcomeLineUseCase
	ListJobOutcomeLines           *ListJobOutcomeLinesUseCase
	GetJobOutcomeLineListPageData *GetJobOutcomeLineListPageDataUseCase
	GetJobOutcomeLineItemPageData *GetJobOutcomeLineItemPageDataUseCase
}

// Repositories groups the primary repository dependency.
type Repositories struct {
	JobOutcomeLine pb.JobOutcomeLineDomainServiceServer
}

// Services groups the shared business-service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// NewUseCases wires the job_outcome_line use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.JobOutcomeLine
	return &UseCases{
		CreateJobOutcomeLine:          NewCreateJobOutcomeLineUseCase(CreateJobOutcomeLineRepositories{JobOutcomeLine: repo}, CreateJobOutcomeLineServices(s)),
		ReadJobOutcomeLine:            NewReadJobOutcomeLineUseCase(ReadJobOutcomeLineRepositories{JobOutcomeLine: repo}, ReadJobOutcomeLineServices(s)),
		UpdateJobOutcomeLine:          NewUpdateJobOutcomeLineUseCase(UpdateJobOutcomeLineRepositories{JobOutcomeLine: repo}, UpdateJobOutcomeLineServices(s)),
		DeleteJobOutcomeLine:          NewDeleteJobOutcomeLineUseCase(DeleteJobOutcomeLineRepositories{JobOutcomeLine: repo}, DeleteJobOutcomeLineServices(s)),
		ListJobOutcomeLines:           NewListJobOutcomeLinesUseCase(ListJobOutcomeLinesRepositories{JobOutcomeLine: repo}, ListJobOutcomeLinesServices(s)),
		GetJobOutcomeLineListPageData: NewGetJobOutcomeLineListPageDataUseCase(GetJobOutcomeLineListPageDataRepositories{JobOutcomeLine: repo}, GetJobOutcomeLineListPageDataServices(s)),
		GetJobOutcomeLineItemPageData: NewGetJobOutcomeLineItemPageDataUseCase(GetJobOutcomeLineItemPageDataRepositories{JobOutcomeLine: repo}, GetJobOutcomeLineItemPageDataServices(s)),
	}
}
