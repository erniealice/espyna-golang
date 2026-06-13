package collection_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
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
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadCollectionAttributeRepositories{
		CollectionAttribute: repositories.CollectionAttribute,
	}
	readServices := ReadCollectionAttributeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateCollectionAttributeRepositories{
		CollectionAttribute: repositories.CollectionAttribute,
		Collection:          repositories.Collection,
		Attribute:           repositories.Attribute,
	}
	updateServices := UpdateCollectionAttributeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteCollectionAttributeRepositories{
		CollectionAttribute: repositories.CollectionAttribute,
	}
	deleteServices := DeleteCollectionAttributeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListCollectionAttributesRepositories{
		CollectionAttribute: repositories.CollectionAttribute,
	}
	listServices := ListCollectionAttributesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetCollectionAttributeListPageDataRepositories{
		CollectionAttribute: repositories.CollectionAttribute,
	}
	listPageDataServices := GetCollectionAttributeListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetCollectionAttributeItemPageDataRepositories{
		CollectionAttribute: repositories.CollectionAttribute,
	}
	itemPageDataServices := GetCollectionAttributeItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
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
