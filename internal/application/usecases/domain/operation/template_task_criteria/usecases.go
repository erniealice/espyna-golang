package template_task_criteria

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

// TemplateTaskCriteriaRepositories groups all repository dependencies
type TemplateTaskCriteriaRepositories struct {
	TemplateTaskCriteria pb.TemplateTaskCriteriaDomainServiceServer
}

// TemplateTaskCriteriaServices groups all business service dependencies
type TemplateTaskCriteriaServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all template_task_criteria-related use cases
type UseCases struct {
	CreateTemplateTaskCriteria          *CreateTemplateTaskCriteriaUseCase
	ReadTemplateTaskCriteria            *ReadTemplateTaskCriteriaUseCase
	UpdateTemplateTaskCriteria          *UpdateTemplateTaskCriteriaUseCase
	DeleteTemplateTaskCriteria          *DeleteTemplateTaskCriteriaUseCase
	ListTemplateTaskCriteria            *ListTemplateTaskCriteriaUseCase
	GetTemplateTaskCriteriaListPageData *GetTemplateTaskCriteriaListPageDataUseCase
	GetTemplateTaskCriteriaItemPageData *GetTemplateTaskCriteriaItemPageDataUseCase
	ListByTemplateTask                  *ListByTemplateTaskUseCase
	ListByCriteria                      *ListByCriteriaUseCase
}

// NewUseCases creates a new collection of template_task_criteria use cases
func NewUseCases(
	repositories TemplateTaskCriteriaRepositories,
	services TemplateTaskCriteriaServices,
) *UseCases {
	createRepos := CreateTemplateTaskCriteriaRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	createServices := CreateTemplateTaskCriteriaServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadTemplateTaskCriteriaRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	readServices := ReadTemplateTaskCriteriaServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateTemplateTaskCriteriaRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	updateServices := UpdateTemplateTaskCriteriaServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteTemplateTaskCriteriaRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	deleteServices := DeleteTemplateTaskCriteriaServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListTemplateTaskCriteriaRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	listServices := ListTemplateTaskCriteriaServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetTemplateTaskCriteriaListPageDataRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	listPageDataServices := GetTemplateTaskCriteriaListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetTemplateTaskCriteriaItemPageDataRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	itemPageDataServices := GetTemplateTaskCriteriaItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listByTemplateTaskRepos := ListByTemplateTaskRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	listByTemplateTaskServices := ListByTemplateTaskServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listByCriteriaRepos := ListByCriteriaRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	listByCriteriaServices := ListByCriteriaServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateTemplateTaskCriteria:          NewCreateTemplateTaskCriteriaUseCase(createRepos, createServices),
		ReadTemplateTaskCriteria:            NewReadTemplateTaskCriteriaUseCase(readRepos, readServices),
		UpdateTemplateTaskCriteria:          NewUpdateTemplateTaskCriteriaUseCase(updateRepos, updateServices),
		DeleteTemplateTaskCriteria:          NewDeleteTemplateTaskCriteriaUseCase(deleteRepos, deleteServices),
		ListTemplateTaskCriteria:            NewListTemplateTaskCriteriaUseCase(listRepos, listServices),
		GetTemplateTaskCriteriaListPageData: NewGetTemplateTaskCriteriaListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetTemplateTaskCriteriaItemPageData: NewGetTemplateTaskCriteriaItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
		ListByTemplateTask:                  NewListByTemplateTaskUseCase(listByTemplateTaskRepos, listByTemplateTaskServices),
		ListByCriteria:                      NewListByCriteriaUseCase(listByCriteriaRepos, listByCriteriaServices),
	}
}
