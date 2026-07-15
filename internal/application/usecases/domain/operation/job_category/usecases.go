package job_category

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_category"
)

// UseCases aggregates the job_category CRUD + page-data use cases.
type UseCases struct {
	CreateJobCategory          *CreateJobCategoryUseCase
	ReadJobCategory            *ReadJobCategoryUseCase
	UpdateJobCategory          *UpdateJobCategoryUseCase
	DeleteJobCategory          *DeleteJobCategoryUseCase
	ListJobCategories          *ListJobCategoriesUseCase
	GetJobCategoryListPageData *GetJobCategoryListPageDataUseCase
	GetJobCategoryItemPageData *GetJobCategoryItemPageDataUseCase
}

// Repositories groups the primary repository dependency.
type Repositories struct {
	JobCategory pb.JobCategoryDomainServiceServer
}

// Services groups the shared business-service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// NewUseCases wires the job_category use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.JobCategory
	return &UseCases{
		CreateJobCategory:          NewCreateJobCategoryUseCase(CreateJobCategoryRepositories{JobCategory: repo}, CreateJobCategoryServices(s)),
		ReadJobCategory:            NewReadJobCategoryUseCase(ReadJobCategoryRepositories{JobCategory: repo}, ReadJobCategoryServices(s)),
		UpdateJobCategory:          NewUpdateJobCategoryUseCase(UpdateJobCategoryRepositories{JobCategory: repo}, UpdateJobCategoryServices(s)),
		DeleteJobCategory:          NewDeleteJobCategoryUseCase(DeleteJobCategoryRepositories{JobCategory: repo}, DeleteJobCategoryServices(s)),
		ListJobCategories:          NewListJobCategoriesUseCase(ListJobCategoriesRepositories{JobCategory: repo}, ListJobCategoriesServices(s)),
		GetJobCategoryListPageData: NewGetJobCategoryListPageDataUseCase(GetJobCategoryListPageDataRepositories{JobCategory: repo}, GetJobCategoryListPageDataServices(s)),
		GetJobCategoryItemPageData: NewGetJobCategoryItemPageDataUseCase(GetJobCategoryItemPageDataRepositories{JobCategory: repo}, GetJobCategoryItemPageDataServices(s)),
	}
}
