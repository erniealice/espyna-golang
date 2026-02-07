package plan_attribute

import (
	"leapfor.xyz/espyna/internal/application/ports"
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
	planpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan"
	planattributepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan_attribute"
)

// UseCases contains all plan attribute-related use cases
type UseCases struct {
	CreatePlanAttribute          *CreatePlanAttributeUseCase
	ReadPlanAttribute            *ReadPlanAttributeUseCase
	UpdatePlanAttribute          *UpdatePlanAttributeUseCase
	DeletePlanAttribute          *DeletePlanAttributeUseCase
	ListPlanAttributes           *ListPlanAttributesUseCase
	GetPlanAttributeListPageData *GetPlanAttributeListPageDataUseCase
	GetPlanAttributeItemPageData *GetPlanAttributeItemPageDataUseCase
}

// PlanAttributeRepositories groups all repository dependencies for plan attribute use cases
type PlanAttributeRepositories struct {
	PlanAttribute planattributepb.PlanAttributeDomainServiceServer // Primary entity repository
	Plan          planpb.PlanDomainServiceServer                   // Entity reference validation
	Attribute     attributepb.AttributeDomainServiceServer         // Entity reference validation
}

// PlanAttributeServices groups all business service dependencies for plan attribute use cases
type PlanAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// NewUseCases creates a new collection of plan attribute use cases
func NewUseCases(
	repositories PlanAttributeRepositories,
	services PlanAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreatePlanAttributeRepositories(repositories)
	createServices := CreatePlanAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadPlanAttributeRepositories{
		PlanAttribute: repositories.PlanAttribute,
	}
	readServices := ReadPlanAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	updateRepos := UpdatePlanAttributeRepositories{
		PlanAttribute: repositories.PlanAttribute,
		Plan:          repositories.Plan,
		Attribute:     repositories.Attribute,
	}
	updateServices := UpdatePlanAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	deleteRepos := DeletePlanAttributeRepositories{
		PlanAttribute: repositories.PlanAttribute,
	}
	deleteServices := DeletePlanAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listRepos := ListPlanAttributesRepositories{
		PlanAttribute: repositories.PlanAttribute,
	}
	listServices := ListPlanAttributesServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getListPageDataRepos := GetPlanAttributeListPageDataRepositories{
		PlanAttribute: repositories.PlanAttribute,
	}
	getListPageDataServices := GetPlanAttributeListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getItemPageDataRepos := GetPlanAttributeItemPageDataRepositories{
		PlanAttribute: repositories.PlanAttribute,
	}
	getItemPageDataServices := GetPlanAttributeItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	return &UseCases{
		CreatePlanAttribute:          NewCreatePlanAttributeUseCase(createRepos, createServices),
		ReadPlanAttribute:            NewReadPlanAttributeUseCase(readRepos, readServices),
		UpdatePlanAttribute:          NewUpdatePlanAttributeUseCase(updateRepos, updateServices),
		DeletePlanAttribute:          NewDeletePlanAttributeUseCase(deleteRepos, deleteServices),
		ListPlanAttributes:           NewListPlanAttributesUseCase(listRepos, listServices),
		GetPlanAttributeListPageData: NewGetPlanAttributeListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetPlanAttributeItemPageData: NewGetPlanAttributeItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}
