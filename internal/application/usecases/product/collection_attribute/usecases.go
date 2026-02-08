package collection_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection"
	collectionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_attribute"
)

// CollectionAttributeRepositories groups all repository dependencies for collection attribute use cases
type CollectionAttributeRepositories struct {
	CollectionAttribute collectionattributepb.CollectionAttributeDomainServiceServer // Primary entity repository
	Collection          collectionpb.CollectionDomainServiceServer
	Attribute           attributepb.AttributeDomainServiceServer
}

// CollectionAttributeServices groups all business service dependencies for collection attribute use cases
type CollectionAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all collection attribute-related use cases
type UseCases struct {
	CreateCollectionAttribute          *CreateCollectionAttributeUseCase
	ReadCollectionAttribute            *ReadCollectionAttributeUseCase
	UpdateCollectionAttribute          *UpdateCollectionAttributeUseCase
	DeleteCollectionAttribute          *DeleteCollectionAttributeUseCase
	ListCollectionAttributes           *ListCollectionAttributesUseCase
	GetCollectionAttributeListPageData *GetCollectionAttributeListPageDataUseCase
	GetCollectionAttributeItemPageData *GetCollectionAttributeItemPageDataUseCase
}

// NewUseCases creates a new collection of collection attribute use cases
func NewUseCases(
	repositories CollectionAttributeRepositories,
	services CollectionAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateCollectionAttributeRepositories{
		CollectionAttribute: repositories.CollectionAttribute,
		Collection:          repositories.Collection,
		Attribute:           repositories.Attribute,
	}
	createServices := CreateCollectionAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadCollectionAttributeRepositories{
		CollectionAttribute: repositories.CollectionAttribute,
	}
	readServices := ReadCollectionAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateCollectionAttributeRepositories{
		CollectionAttribute: repositories.CollectionAttribute,
		Collection:          repositories.Collection,
		Attribute:           repositories.Attribute,
	}
	updateServices := UpdateCollectionAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteCollectionAttributeRepositories{
		CollectionAttribute: repositories.CollectionAttribute,
	}
	deleteServices := DeleteCollectionAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListCollectionAttributesRepositories{
		CollectionAttribute: repositories.CollectionAttribute,
	}
	listServices := ListCollectionAttributesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetCollectionAttributeListPageDataRepositories{
		CollectionAttribute: repositories.CollectionAttribute,
	}
	listPageDataServices := GetCollectionAttributeListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	itemPageDataRepos := GetCollectionAttributeItemPageDataRepositories{
		CollectionAttribute: repositories.CollectionAttribute,
	}
	itemPageDataServices := GetCollectionAttributeItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	return &UseCases{
		CreateCollectionAttribute:          NewCreateCollectionAttributeUseCase(createRepos, createServices),
		ReadCollectionAttribute:            NewReadCollectionAttributeUseCase(readRepos, readServices),
		UpdateCollectionAttribute:          NewUpdateCollectionAttributeUseCase(updateRepos, updateServices),
		DeleteCollectionAttribute:          NewDeleteCollectionAttributeUseCase(deleteRepos, deleteServices),
		ListCollectionAttributes:           NewListCollectionAttributesUseCase(listRepos, listServices),
		GetCollectionAttributeListPageData: NewGetCollectionAttributeListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetCollectionAttributeItemPageData: NewGetCollectionAttributeItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}
