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
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService // Only for CreatePlanSettings
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadPlanSettingsRepositories{
		PlanSettings: repositories.PlanSettings,
	}
	readServices := ReadPlanSettingsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdatePlanSettingsRepositories{
		PlanSettings: repositories.PlanSettings,
		Plan:         repositories.Plan,
	}
	updateServices := UpdatePlanSettingsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeletePlanSettingsRepositories{
		PlanSettings: repositories.PlanSettings,
	}
	deleteServices := DeletePlanSettingsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListPlanSettingsRepositories{
		PlanSettings: repositories.PlanSettings,
	}
	listServices := ListPlanSettingsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreatePlanSettings: NewCreatePlanSettingsUseCase(createRepos, createServices),
		ReadPlanSettings:   NewReadPlanSettingsUseCase(readRepos, readServices),
		UpdatePlanSettings: NewUpdatePlanSettingsUseCase(updateRepos, updateServices),
		DeletePlanSettings: NewDeletePlanSettingsUseCase(deleteRepos, deleteServices),
		ListPlanSettings:   NewListPlanSettingsUseCase(listRepos, listServices),
	}
}
