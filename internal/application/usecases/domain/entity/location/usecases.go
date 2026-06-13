package location

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
)

// LocationRepositories groups all repository dependencies for location use cases
type LocationRepositories struct {
	Location locationpb.LocationDomainServiceServer // Primary entity repository
}

// LocationServices groups all business service dependencies for location use cases
type LocationServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all location-related use cases
type UseCases struct {
	CreateLocation          *CreateLocationUseCase
	ReadLocation            *ReadLocationUseCase
	UpdateLocation          *UpdateLocationUseCase
	DeleteLocation          *DeleteLocationUseCase
	ListLocations           *ListLocationsUseCase
	GetLocationListPageData *GetLocationListPageDataUseCase
	GetLocationItemPageData *GetLocationItemPageDataUseCase
}

// NewUseCases creates a new collection of location use cases
func NewUseCases(
	repositories LocationRepositories,
	services LocationServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateLocationRepositories(repositories)
	createServices := CreateLocationServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadLocationRepositories(repositories)
	readServices := ReadLocationServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateLocationRepositories(repositories)
	updateServices := UpdateLocationServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteLocationRepositories(repositories)
	deleteServices := DeleteLocationServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListLocationsRepositories(repositories)
	listServices := ListLocationsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetLocationListPageDataRepositories(repositories)
	getListPageDataServices := GetLocationListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetLocationItemPageDataRepositories(repositories)
	getItemPageDataServices := GetLocationItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreateLocation:          NewCreateLocationUseCase(createRepos, createServices),
		ReadLocation:            NewReadLocationUseCase(readRepos, readServices),
		UpdateLocation:          NewUpdateLocationUseCase(updateRepos, updateServices),
		DeleteLocation:          NewDeleteLocationUseCase(deleteRepos, deleteServices),
		ListLocations:           NewListLocationsUseCase(listRepos, listServices),
		GetLocationListPageData: NewGetLocationListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetLocationItemPageData: NewGetLocationItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of location use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(locationRepo locationpb.LocationDomainServiceServer) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := LocationRepositories{
		Location: locationRepo,
	}

	services := LocationServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewUseCases(repositories, services)
}
