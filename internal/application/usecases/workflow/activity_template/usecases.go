package activity_template

import (
	"leapfor.xyz/espyna/internal/application/ports"
	activityTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity_template"
	stageTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage_template"
)

// ActivityTemplateRepositories groups all repository dependencies for activity template use cases
type ActivityTemplateRepositories struct {
	ActivityTemplate activityTemplatepb.ActivityTemplateDomainServiceServer // Primary entity repository
	StageTemplate    stageTemplatepb.StageTemplateDomainServiceServer       // Foreign key reference
}

// ActivityTemplateServices groups all business service dependencies for activity template use cases
type ActivityTemplateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadActivityTemplateRepositories{
		ActivityTemplate: repositories.ActivityTemplate,
	}
	readServices := ReadActivityTemplateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateActivityTemplateRepositories{
		ActivityTemplate: repositories.ActivityTemplate,
		StageTemplate:    repositories.StageTemplate,
	}
	updateServices := UpdateActivityTemplateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteActivityTemplateRepositories{
		ActivityTemplate: repositories.ActivityTemplate,
	}
	deleteServices := DeleteActivityTemplateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListActivityTemplatesRepositories{
		ActivityTemplate: repositories.ActivityTemplate,
	}
	listServices := ListActivityTemplatesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetActivityTemplateListPageDataRepositories{
		ActivityTemplate: repositories.ActivityTemplate,
		StageTemplate:    repositories.StageTemplate,
	}
	getListPageDataServices := GetActivityTemplateListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getItemPageDataRepos := GetActivityTemplateItemPageDataRepositories{
		ActivityTemplate: repositories.ActivityTemplate,
		StageTemplate:    repositories.StageTemplate,
	}
	getItemPageDataServices := GetActivityTemplateItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	// TODO: Implement when GetActivityTemplatesByStageTemplate use case is available
	// getByStageTemplateRepos := GetActivityTemplatesByStageTemplateRepositories(repositories)
	// getByStageTemplateServices := GetActivityTemplatesByStageTemplateServices{
	// 	AuthorizationService: services.AuthorizationService,
	// 	TransactionService:   services.TransactionService,
	// 	TranslationService:   services.TranslationService,
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
		AuthorizationService: nil, // Will be injected later by container
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewUseCases(repositories, services)
}
