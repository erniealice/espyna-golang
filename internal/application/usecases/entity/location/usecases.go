package location

import (
	"leapfor.xyz/espyna/internal/application/ports"
	locationpb "leapfor.xyz/esqyma/golang/v1/domain/entity/location"
)

// LocationRepositories groups all repository dependencies for location use cases
type LocationRepositories struct {
	Location locationpb.LocationDomainServiceServer // Primary entity repository
}

// LocationServices groups all business service dependencies for location use cases
type LocationServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	readRepos := ReadLocationRepositories(repositories)
	readServices := ReadLocationServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateLocationRepositories(repositories)
	updateServices := UpdateLocationServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteLocationRepositories(repositories)
	deleteServices := DeleteLocationServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListLocationsRepositories(repositories)
	listServices := ListLocationsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetLocationListPageDataRepositories(repositories)
	getListPageDataServices := GetLocationListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getItemPageDataRepos := GetLocationItemPageDataRepositories(repositories)
	getItemPageDataServices := GetLocationItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
