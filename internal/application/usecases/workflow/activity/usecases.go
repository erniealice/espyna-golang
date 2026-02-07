package activity

import (
	"leapfor.xyz/espyna/internal/application/ports"
	activitypb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity"
	activityTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity_template"
	stagepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage"
)

// ActivityRepositories groups all repository dependencies for activity use cases
type ActivityRepositories struct {
	Activity         activitypb.ActivityDomainServiceServer                 // Primary entity repository
	Stage            stagepb.StageDomainServiceServer                       // Foreign key reference
	ActivityTemplate activityTemplatepb.ActivityTemplateDomainServiceServer // Foreign key reference
}

// ActivityServices groups all business service dependencies for activity use cases
type ActivityServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService // Required for Create use case
}

// UseCases contains all activity-related use cases
type UseCases struct {
	CreateActivity *CreateActivityUseCase
	ReadActivity   *ReadActivityUseCase
	UpdateActivity *UpdateActivityUseCase
	DeleteActivity *DeleteActivityUseCase
	ListActivities *ListActivitiesUseCase
}

// NewUseCases creates a new collection of activity use cases
func NewUseCases(
	repositories ActivityRepositories,
	services ActivityServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateActivityRepositories{
		Activity:         repositories.Activity,
		Stage:            repositories.Stage,
		ActivityTemplate: repositories.ActivityTemplate,
	}
	createServices := CreateActivityServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}
	readRepos := ReadActivityRepositories{
		Activity: repositories.Activity,
	}
	readServices := ReadActivityServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateActivityRepositories{
		Activity: repositories.Activity,
	}
	updateServices := UpdateActivityServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteActivityRepositories{
		Activity: repositories.Activity,
	}
	deleteServices := DeleteActivityServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListActivitiesRepositories{
		Activity: repositories.Activity,
	}
	listServices := ListActivitiesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateActivity: NewCreateActivityUseCase(createRepos, createServices),
		ReadActivity:   NewReadActivityUseCase(readRepos, readServices),
		UpdateActivity: NewUpdateActivityUseCase(updateRepos, updateServices),
		DeleteActivity: NewDeleteActivityUseCase(deleteRepos, deleteServices),
		ListActivities: NewListActivitiesUseCase(listRepos, listServices),
	}
}
