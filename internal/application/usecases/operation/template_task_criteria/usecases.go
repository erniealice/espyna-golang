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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadTemplateTaskCriteriaRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	readServices := ReadTemplateTaskCriteriaServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateTemplateTaskCriteriaRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	updateServices := UpdateTemplateTaskCriteriaServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteTemplateTaskCriteriaRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	deleteServices := DeleteTemplateTaskCriteriaServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListTemplateTaskCriteriaRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	listServices := ListTemplateTaskCriteriaServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetTemplateTaskCriteriaListPageDataRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	listPageDataServices := GetTemplateTaskCriteriaListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	itemPageDataRepos := GetTemplateTaskCriteriaItemPageDataRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	itemPageDataServices := GetTemplateTaskCriteriaItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listByTemplateTaskRepos := ListByTemplateTaskRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	listByTemplateTaskServices := ListByTemplateTaskServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listByCriteriaRepos := ListByCriteriaRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	listByCriteriaServices := ListByCriteriaServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
