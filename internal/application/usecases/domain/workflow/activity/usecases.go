package activity

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	activitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity"
	activityTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity_template"
	stagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage"
)

// ActivityRepositories groups all repository dependencies for activity use cases
type ActivityRepositories struct {
	Activity         activitypb.ActivityDomainServiceServer                 // Primary entity repository
	Stage            stagepb.StageDomainServiceServer                       // Foreign key reference
	ActivityTemplate activityTemplatepb.ActivityTemplateDomainServiceServer // Foreign key reference
}

// ActivityServices groups all business service dependencies for activity use cases
type ActivityServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	IDGenerator ports.IDGenerator // Required for Create use case
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}
	readRepos := ReadActivityRepositories{
		Activity: repositories.Activity,
	}
	readServices := ReadActivityServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateActivityRepositories{
		Activity: repositories.Activity,
	}
	updateServices := UpdateActivityServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteActivityRepositories{
		Activity: repositories.Activity,
	}
	deleteServices := DeleteActivityServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListActivitiesRepositories{
		Activity: repositories.Activity,
	}
	listServices := ListActivitiesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateActivity: NewCreateActivityUseCase(createRepos, createServices),
		ReadActivity:   NewReadActivityUseCase(readRepos, readServices),
		UpdateActivity: NewUpdateActivityUseCase(updateRepos, updateServices),
		DeleteActivity: NewDeleteActivityUseCase(deleteRepos, deleteServices),
		ListActivities: NewListActivitiesUseCase(listRepos, listServices),
	}
}
