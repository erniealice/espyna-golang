package location_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
	locationattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_attribute"
)

// LocationAttributeRepositories groups all repository dependencies for location attribute use cases
type LocationAttributeRepositories struct {
	LocationAttribute locationattributepb.LocationAttributeDomainServiceServer // Primary entity repository
	Location          locationpb.LocationDomainServiceServer                   // Foreign key validation
	Attribute         attributepb.AttributeDomainServiceServer                 // Foreign key validation
}

// LocationAttributeServices groups all business service dependencies for location attribute use cases
type LocationAttributeServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all location attribute-related use cases
type UseCases struct {
	CreateLocationAttribute          *CreateLocationAttributeUseCase
	ReadLocationAttribute            *ReadLocationAttributeUseCase
	UpdateLocationAttribute          *UpdateLocationAttributeUseCase
	DeleteLocationAttribute          *DeleteLocationAttributeUseCase
	ListLocationAttributes           *ListLocationAttributesUseCase
	GetLocationAttributeListPageData *GetLocationAttributeListPageDataUseCase
	GetLocationAttributeItemPageData *GetLocationAttributeItemPageDataUseCase
}

// NewUseCases creates a new collection of location attribute use cases
func NewUseCases(
	repositories LocationAttributeRepositories,
	services LocationAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateLocationAttributeRepositories{
		LocationAttribute: repositories.LocationAttribute,
		Location:          repositories.Location,
		Attribute:         repositories.Attribute,
	}
	createServices := CreateLocationAttributeServices{
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadLocationAttributeRepositories{
		LocationAttribute: repositories.LocationAttribute,
	}
	readServices := ReadLocationAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateLocationAttributeRepositories{
		LocationAttribute: repositories.LocationAttribute,
		Location:          repositories.Location,
		Attribute:         repositories.Attribute,
	}
	updateServices := UpdateLocationAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteLocationAttributeRepositories{
		LocationAttribute: repositories.LocationAttribute,
	}
	deleteServices := DeleteLocationAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListLocationAttributesRepositories{
		LocationAttribute: repositories.LocationAttribute,
	}
	listServices := ListLocationAttributesServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetLocationAttributeListPageDataRepositories{
		LocationAttribute: repositories.LocationAttribute,
	}
	listPageDataServices := GetLocationAttributeListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetLocationAttributeItemPageDataRepositories{
		LocationAttribute: repositories.LocationAttribute,
	}
	itemPageDataServices := GetLocationAttributeItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateLocationAttribute:          NewCreateLocationAttributeUseCase(createRepos, createServices),
		ReadLocationAttribute:            NewReadLocationAttributeUseCase(readRepos, readServices),
		UpdateLocationAttribute:          NewUpdateLocationAttributeUseCase(updateRepos, updateServices),
		DeleteLocationAttribute:          NewDeleteLocationAttributeUseCase(deleteRepos, deleteServices),
		ListLocationAttributes:           NewListLocationAttributesUseCase(listRepos, listServices),
		GetLocationAttributeListPageData: NewGetLocationAttributeListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetLocationAttributeItemPageData: NewGetLocationAttributeItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of location attribute use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(
	locationAttributeRepo locationattributepb.LocationAttributeDomainServiceServer,
	locationRepo locationpb.LocationDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := LocationAttributeRepositories{
		LocationAttribute: locationAttributeRepo,
		Location:          locationRepo,
		Attribute:         attributeRepo,
	}

	services := LocationAttributeServices{
		Authorizer:  nil,
		Transactor:  ports.NewNoOpTransactor(),
		Translator:  ports.NewNoOpTranslator(),
		IDGenerator: ports.NewNoOpIDGenerator(),
	}

	return NewUseCases(repositories, services)
}
