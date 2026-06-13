package work_request_type

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	work_request_typepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request_type"
)

// WorkRequestTypeRepositories groups all repository dependencies for work request type use cases
type WorkRequestTypeRepositories struct {
	WorkRequestType work_request_typepb.WorkRequestTypeDomainServiceServer // Primary entity repository
}

// WorkRequestTypeServices groups all business service dependencies for work request type use cases
type WorkRequestTypeServices struct {
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator      ports.IDGenerator
}

// UseCases contains all work request type-related use cases
type UseCases struct {
	CreateWorkRequestType              *CreateWorkRequestTypeUseCase
	ReadWorkRequestType                *ReadWorkRequestTypeUseCase
	UpdateWorkRequestType              *UpdateWorkRequestTypeUseCase
	ListWorkRequestTypes               *ListWorkRequestTypesUseCase
	GetWorkRequestTypeListPageData     *GetWorkRequestTypeListPageDataUseCase
}

// NewUseCases creates a new collection of work request type use cases
func NewUseCases(
	repositories WorkRequestTypeRepositories,
	services WorkRequestTypeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateWorkRequestTypeRepositories(repositories)
	createServices := CreateWorkRequestTypeServices{
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator:      services.IDGenerator,
	}

	readRepos := ReadWorkRequestTypeRepositories(repositories)
	readServices := ReadWorkRequestTypeServices{
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateWorkRequestTypeRepositories(repositories)
	updateServices := UpdateWorkRequestTypeServices{
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListWorkRequestTypesRepositories(repositories)
	listServices := ListWorkRequestTypesServices{
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetWorkRequestTypeListPageDataRepositories(repositories)
	getListPageDataServices := GetWorkRequestTypeListPageDataServices{
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreateWorkRequestType:          NewCreateWorkRequestTypeUseCase(createRepos, createServices),
		ReadWorkRequestType:            NewReadWorkRequestTypeUseCase(readRepos, readServices),
		UpdateWorkRequestType:          NewUpdateWorkRequestTypeUseCase(updateRepos, updateServices),
		ListWorkRequestTypes:           NewListWorkRequestTypesUseCase(listRepos, listServices),
		GetWorkRequestTypeListPageData: NewGetWorkRequestTypeListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
