package collection_method

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// CollectionMethodRepositories groups all repository dependencies for collection method use cases
type CollectionMethodRepositories struct {
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer // Primary entity repository
}

// CollectionMethodServices groups all business service dependencies for collection method use cases
type CollectionMethodServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all collection-method-related use cases
type UseCases struct {
	CreateCollectionMethod          *CreateCollectionMethodUseCase
	ReadCollectionMethod            *ReadCollectionMethodUseCase
	UpdateCollectionMethod          *UpdateCollectionMethodUseCase
	DeleteCollectionMethod          *DeleteCollectionMethodUseCase
	ListCollectionMethods           *ListCollectionMethodsUseCase
	GetCollectionMethodListPageData *GetCollectionMethodListPageDataUseCase
	GetCollectionMethodItemPageData *GetCollectionMethodItemPageDataUseCase
}

// NewUseCases creates a new collection of collection method use cases
func NewUseCases(
	repositories CollectionMethodRepositories,
	services CollectionMethodServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateCollectionMethodRepositories(repositories)
	createServices := CreateCollectionMethodServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadCollectionMethodRepositories(repositories)
	readServices := ReadCollectionMethodServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateCollectionMethodRepositories(repositories)
	updateServices := UpdateCollectionMethodServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteCollectionMethodRepositories(repositories)
	deleteServices := DeleteCollectionMethodServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListCollectionMethodsRepositories(repositories)
	listServices := ListCollectionMethodsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetCollectionMethodListPageDataRepositories(repositories)
	getListPageDataServices := GetCollectionMethodListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getItemPageDataRepos := GetCollectionMethodItemPageDataRepositories(repositories)
	getItemPageDataServices := GetCollectionMethodItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateCollectionMethod:          NewCreateCollectionMethodUseCase(createRepos, createServices),
		ReadCollectionMethod:            NewReadCollectionMethodUseCase(readRepos, readServices),
		UpdateCollectionMethod:          NewUpdateCollectionMethodUseCase(updateRepos, updateServices),
		DeleteCollectionMethod:          NewDeleteCollectionMethodUseCase(deleteRepos, deleteServices),
		ListCollectionMethods:           NewListCollectionMethodsUseCase(listRepos, listServices),
		GetCollectionMethodListPageData: NewGetCollectionMethodListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetCollectionMethodItemPageData: NewGetCollectionMethodItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of collection method use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(collectionMethodRepo collectionmethodpb.CollectionMethodDomainServiceServer) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := CollectionMethodRepositories{
		CollectionMethod: collectionMethodRepo,
	}

	services := CollectionMethodServices{
		Authorizer:  nil,
		Transactor:  ports.NewNoOpTransactor(),
		Translator:  ports.NewNoOpTranslator(),
		IDGenerator: ports.NewNoOpIDGenerator(),
	}

	return NewUseCases(repositories, services)
}
