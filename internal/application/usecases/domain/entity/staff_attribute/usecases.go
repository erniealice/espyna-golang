package staff_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	staffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff"
	staffattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff_attribute"
)

// UseCases contains all staff attribute-related use cases
type UseCases struct {
	CreateStaffAttribute          *CreateStaffAttributeUseCase
	ReadStaffAttribute            *ReadStaffAttributeUseCase
	UpdateStaffAttribute          *UpdateStaffAttributeUseCase
	DeleteStaffAttribute          *DeleteStaffAttributeUseCase
	ListStaffAttributes           *ListStaffAttributesUseCase
	GetStaffAttributeListPageData *GetStaffAttributeListPageDataUseCase
	GetStaffAttributeItemPageData *GetStaffAttributeItemPageDataUseCase
}

// StaffAttributeRepositories groups all repository dependencies for staff attribute use cases
type StaffAttributeRepositories struct {
	StaffAttribute staffattributepb.StaffAttributeDomainServiceServer // Primary entity repository
	Staff          staffpb.StaffDomainServiceServer                   // Entity reference validation
	Attribute      attributepb.AttributeDomainServiceServer           // Entity reference validation
}

// StaffAttributeServices groups all business service dependencies for staff attribute use cases
type StaffAttributeServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of staff attribute use cases
func NewUseCases(
	repositories StaffAttributeRepositories,
	services StaffAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateStaffAttributeRepositories(repositories)
	createServices := CreateStaffAttributeServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadStaffAttributeRepositories{
		StaffAttribute: repositories.StaffAttribute,
	}
	readServices := ReadStaffAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateStaffAttributeRepositories{
		StaffAttribute: repositories.StaffAttribute,
		Staff:          repositories.Staff,
		Attribute:      repositories.Attribute,
	}
	updateServices := UpdateStaffAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteStaffAttributeRepositories{
		StaffAttribute: repositories.StaffAttribute,
	}
	deleteServices := DeleteStaffAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListStaffAttributesRepositories{
		StaffAttribute: repositories.StaffAttribute,
	}
	listServices := ListStaffAttributesServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetStaffAttributeListPageDataRepositories{
		StaffAttribute: repositories.StaffAttribute,
	}
	getListPageDataServices := GetStaffAttributeListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetStaffAttributeItemPageDataRepositories{
		StaffAttribute: repositories.StaffAttribute,
	}
	getItemPageDataServices := GetStaffAttributeItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreateStaffAttribute:          NewCreateStaffAttributeUseCase(createRepos, createServices),
		ReadStaffAttribute:            NewReadStaffAttributeUseCase(readRepos, readServices),
		UpdateStaffAttribute:          NewUpdateStaffAttributeUseCase(updateRepos, updateServices),
		DeleteStaffAttribute:          NewDeleteStaffAttributeUseCase(deleteRepos, deleteServices),
		ListStaffAttributes:           NewListStaffAttributesUseCase(listRepos, listServices),
		GetStaffAttributeListPageData: NewGetStaffAttributeListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetStaffAttributeItemPageData: NewGetStaffAttributeItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of staff attribute use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(
	staffAttributeRepo staffattributepb.StaffAttributeDomainServiceServer,
	staffRepo staffpb.StaffDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
	authorizationService ports.Authorizer,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := StaffAttributeRepositories{
		StaffAttribute: staffAttributeRepo,
		Staff:          staffRepo,
		Attribute:      attributeRepo,
	}

	services := StaffAttributeServices{
		Authorizer: authorizationService,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewUseCases(repositories, services)
}
