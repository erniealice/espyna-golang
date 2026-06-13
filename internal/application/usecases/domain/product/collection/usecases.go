package collection

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection"
)

// CollectionRepositories groups all repository dependencies for collection use cases
type CollectionRepositories struct {
	Collection collectionpb.CollectionDomainServiceServer // Primary entity repository
}

// CollectionServices groups all business service dependencies for collection use cases
type CollectionServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
	// Build individual grouped parameters for each use case
	createRepos := CreateCollectionRepositories(repositories)
	createServices := CreateCollectionServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadCollectionRepositories(repositories)
	readServices := ReadCollectionServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateCollectionRepositories(repositories)
	updateServices := UpdateCollectionServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteCollectionRepositories(repositories)
	deleteServices := DeleteCollectionServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListCollectionsRepositories(repositories)
	listServices := ListCollectionsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateCollection: NewCreateCollectionUseCase(createRepos, createServices),
		ReadCollection:   NewReadCollectionUseCase(readRepos, readServices),
		UpdateCollection: NewUpdateCollectionUseCase(updateRepos, updateServices),
		DeleteCollection: NewDeleteCollectionUseCase(deleteRepos, deleteServices),
		ListCollections:  NewListCollectionsUseCase(listRepos, listServices),
	}
}
