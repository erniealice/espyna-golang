package plan_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	planattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_attribute"
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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of plan attribute use cases
func NewUseCases(
	repositories PlanAttributeRepositories,
	services PlanAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreatePlanAttributeRepositories(repositories)
	createServices := CreatePlanAttributeServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadPlanAttributeRepositories{
		PlanAttribute: repositories.PlanAttribute,
	}
	readServices := ReadPlanAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdatePlanAttributeRepositories{
		PlanAttribute: repositories.PlanAttribute,
		Plan:          repositories.Plan,
		Attribute:     repositories.Attribute,
	}
	updateServices := UpdatePlanAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeletePlanAttributeRepositories{
		PlanAttribute: repositories.PlanAttribute,
	}
	deleteServices := DeletePlanAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListPlanAttributesRepositories{
		PlanAttribute: repositories.PlanAttribute,
	}
	listServices := ListPlanAttributesServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetPlanAttributeListPageDataRepositories{
		PlanAttribute: repositories.PlanAttribute,
	}
	getListPageDataServices := GetPlanAttributeListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getItemPageDataRepos := GetPlanAttributeItemPageDataRepositories{
		PlanAttribute: repositories.PlanAttribute,
	}
	getItemPageDataServices := GetPlanAttributeItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
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
