package plan_settings

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"

	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	plansettingspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_settings"
)

// PlanSettingsRepositories groups all repository dependencies for plan_settings use cases
type PlanSettingsRepositories struct {
	PlanSettings plansettingspb.PlanSettingsDomainServiceServer // Primary entity repository
	Plan         planpb.PlanDomainServiceServer                 // Entity reference dependency
}

// PlanSettingsServices groups all business service dependencies for plan_settings use cases
type PlanSettingsServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator // Only for CreatePlanSettings
}

// UseCases contains all plan_settings-related use cases
type UseCases struct {
	CreatePlanSettings *CreatePlanSettingsUseCase
	ReadPlanSettings   *ReadPlanSettingsUseCase
	UpdatePlanSettings *UpdatePlanSettingsUseCase
	DeletePlanSettings *DeletePlanSettingsUseCase
	ListPlanSettings   *ListPlanSettingsUseCase
}

// NewUseCases creates a new collection of plan_settings use cases
func NewUseCases(
	repositories PlanSettingsRepositories,
	services PlanSettingsServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreatePlanSettingsRepositories{
		PlanSettings: repositories.PlanSettings,
		Plan:         repositories.Plan,
	}
	createServices := CreatePlanSettingsServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadPlanSettingsRepositories{
		PlanSettings: repositories.PlanSettings,
	}
	readServices := ReadPlanSettingsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdatePlanSettingsRepositories{
		PlanSettings: repositories.PlanSettings,
		Plan:         repositories.Plan,
	}
	updateServices := UpdatePlanSettingsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeletePlanSettingsRepositories{
		PlanSettings: repositories.PlanSettings,
	}
	deleteServices := DeletePlanSettingsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListPlanSettingsRepositories{
		PlanSettings: repositories.PlanSettings,
	}
	listServices := ListPlanSettingsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreatePlanSettings: NewCreatePlanSettingsUseCase(createRepos, createServices),
		ReadPlanSettings:   NewReadPlanSettingsUseCase(readRepos, readServices),
		UpdatePlanSettings: NewUpdatePlanSettingsUseCase(updateRepos, updateServices),
		DeletePlanSettings: NewDeletePlanSettingsUseCase(deleteRepos, deleteServices),
		ListPlanSettings:   NewListPlanSettingsUseCase(listRepos, listServices),
	}
}
