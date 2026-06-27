package activity_template

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	activityTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity_template"
	stageTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage_template"
)

// ActivityTemplateRepositories groups all repository dependencies for activity template use cases
type ActivityTemplateRepositories struct {
	ActivityTemplate activityTemplatepb.ActivityTemplateDomainServiceServer // Primary entity repository
	StageTemplate    stageTemplatepb.StageTemplateDomainServiceServer       // Foreign key reference
}

// ActivityTemplateServices groups all business service dependencies for activity template use cases
type ActivityTemplateServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all activity template-related use cases
type UseCases struct {
	CreateActivityTemplate          *CreateActivityTemplateUseCase
	ReadActivityTemplate            *ReadActivityTemplateUseCase
	UpdateActivityTemplate          *UpdateActivityTemplateUseCase
	DeleteActivityTemplate          *DeleteActivityTemplateUseCase
	ListActivityTemplates           *ListActivityTemplatesUseCase
	GetActivityTemplateListPageData *GetActivityTemplateListPageDataUseCase
	GetActivityTemplateItemPageData *GetActivityTemplateItemPageDataUseCase
	// GetActivityTemplatesByStageTemplate *GetActivityTemplatesByStageTemplateUseCase // TODO: Implement
}

// NewUseCases creates a new collection of activity template use cases
func NewUseCases(
	repositories ActivityTemplateRepositories,
	services ActivityTemplateServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateActivityTemplateRepositories(repositories)
	createServices := CreateActivityTemplateServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadActivityTemplateRepositories{
		ActivityTemplate: repositories.ActivityTemplate,
	}
	readServices := ReadActivityTemplateServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateActivityTemplateRepositories{
		ActivityTemplate: repositories.ActivityTemplate,
		StageTemplate:    repositories.StageTemplate,
	}
	updateServices := UpdateActivityTemplateServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteActivityTemplateRepositories{
		ActivityTemplate: repositories.ActivityTemplate,
	}
	deleteServices := DeleteActivityTemplateServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListActivityTemplatesRepositories{
		ActivityTemplate: repositories.ActivityTemplate,
	}
	listServices := ListActivityTemplatesServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetActivityTemplateListPageDataRepositories{
		ActivityTemplate: repositories.ActivityTemplate,
		StageTemplate:    repositories.StageTemplate,
	}
	getListPageDataServices := GetActivityTemplateListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getItemPageDataRepos := GetActivityTemplateItemPageDataRepositories{
		ActivityTemplate: repositories.ActivityTemplate,
		StageTemplate:    repositories.StageTemplate,
	}
	getItemPageDataServices := GetActivityTemplateItemPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	// TODO: Implement when GetActivityTemplatesByStageTemplate use case is available
	// getByStageTemplateRepos := GetActivityTemplatesByStageTemplateRepositories(repositories)
	// getByStageTemplateServices := GetActivityTemplatesByStageTemplateServices{
	// 	Authorizer: services.Authorizer,
	// 	Transactor:   services.Transactor,
	// 	Translator:   services.Translator,
	// }

	return &UseCases{
		CreateActivityTemplate:          NewCreateActivityTemplateUseCase(createRepos, createServices),
		ReadActivityTemplate:            NewReadActivityTemplateUseCase(readRepos, readServices),
		UpdateActivityTemplate:          NewUpdateActivityTemplateUseCase(updateRepos, updateServices),
		DeleteActivityTemplate:          NewDeleteActivityTemplateUseCase(deleteRepos, deleteServices),
		ListActivityTemplates:           NewListActivityTemplatesUseCase(listRepos, listServices),
		GetActivityTemplateListPageData: NewGetActivityTemplateListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetActivityTemplateItemPageData: NewGetActivityTemplateItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
		// GetActivityTemplatesByStageTemplate:  NewGetActivityTemplatesByStageTemplateUseCase(getByStageTemplateRepos, getByStageTemplateServices), // TODO: Implement
	}
}

// NewUseCasesUngrouped creates a new collection of activity template use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(activityTemplateRepo activityTemplatepb.ActivityTemplateDomainServiceServer, stageTemplateRepo stageTemplatepb.StageTemplateDomainServiceServer) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := ActivityTemplateRepositories{
		ActivityTemplate: activityTemplateRepo,
		StageTemplate:    stageTemplateRepo,
	}

	services := ActivityTemplateServices{
		Authorizer:  nil, // Will be injected later by container
		Transactor:  ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
		IDGenerator: ports.NewNoOpIDGenerator(),
	}

	return NewUseCases(repositories, services)
}
