package location_area

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	locationareapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_area"
)

// LocationAreaRepositories groups all repository dependencies for location area use cases
type LocationAreaRepositories struct {
	LocationArea locationareapb.LocationAreaDomainServiceServer // Primary entity repository
}

// LocationAreaServices groups all business service dependencies for location area use cases
type LocationAreaServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all location-area-related use cases
type UseCases struct {
	CreateLocationArea          *CreateLocationAreaUseCase
	ReadLocationArea            *ReadLocationAreaUseCase
	UpdateLocationArea          *UpdateLocationAreaUseCase
	DeleteLocationArea          *DeleteLocationAreaUseCase
	ListLocationAreas           *ListLocationAreasUseCase
	GetLocationAreaListPageData *GetLocationAreaListPageDataUseCase
	GetLocationAreaItemPageData *GetLocationAreaItemPageDataUseCase
}

// NewUseCases creates a new collection of location area use cases
func NewUseCases(
	repositories LocationAreaRepositories,
	services LocationAreaServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateLocationAreaRepositories(repositories)
	createServices := CreateLocationAreaServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadLocationAreaRepositories(repositories)
	readServices := ReadLocationAreaServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateLocationAreaRepositories(repositories)
	updateServices := UpdateLocationAreaServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteLocationAreaRepositories(repositories)
	deleteServices := DeleteLocationAreaServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListLocationAreasRepositories(repositories)
	listServices := ListLocationAreasServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetLocationAreaListPageDataRepositories(repositories)
	getListPageDataServices := GetLocationAreaListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetLocationAreaItemPageDataRepositories(repositories)
	getItemPageDataServices := GetLocationAreaItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreateLocationArea:          NewCreateLocationAreaUseCase(createRepos, createServices),
		ReadLocationArea:            NewReadLocationAreaUseCase(readRepos, readServices),
		UpdateLocationArea:          NewUpdateLocationAreaUseCase(updateRepos, updateServices),
		DeleteLocationArea:          NewDeleteLocationAreaUseCase(deleteRepos, deleteServices),
		ListLocationAreas:           NewListLocationAreasUseCase(listRepos, listServices),
		GetLocationAreaListPageData: NewGetLocationAreaListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetLocationAreaItemPageData: NewGetLocationAreaItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of location area use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(locationAreaRepo locationareapb.LocationAreaDomainServiceServer) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := LocationAreaRepositories{
		LocationArea: locationAreaRepo,
	}

	services := LocationAreaServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewUseCases(repositories, services)
}
