package collection

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// CollectionRepositories groups all repository dependencies for collection use cases
type CollectionRepositories struct {
	Collection collectionpb.CollectionDomainServiceServer
}

// CollectionServices groups all business service dependencies for collection use cases
type CollectionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all collection-related use cases
type UseCases struct {
	CreateCollection *CreateCollectionUseCase
	ReadCollection   *ReadCollectionUseCase
	UpdateCollection *UpdateCollectionUseCase
	DeleteCollection *DeleteCollectionUseCase
	ListCollections  *ListCollectionsUseCase
}

// NewUseCases creates a new collection of collection use cases
func NewUseCases(
	repositories CollectionRepositories,
	services CollectionServices,
) *UseCases {
	createRepos := CreateCollectionRepositories(repositories)
	createServices := CreateCollectionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadCollectionRepositories(repositories)
	readServices := ReadCollectionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateCollectionRepositories(repositories)
	updateServices := UpdateCollectionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteCollectionRepositories(repositories)
	deleteServices := DeleteCollectionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListCollectionsRepositories(repositories)
	listServices := ListCollectionsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateCollection: NewCreateCollectionUseCase(createRepos, createServices),
		ReadCollection:   NewReadCollectionUseCase(readRepos, readServices),
		UpdateCollection: NewUpdateCollectionUseCase(updateRepos, updateServices),
		DeleteCollection: NewDeleteCollectionUseCase(deleteRepos, deleteServices),
		ListCollections:  NewListCollectionsUseCase(listRepos, listServices),
	}
}
