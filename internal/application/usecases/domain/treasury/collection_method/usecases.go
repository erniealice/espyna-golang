// Package collectionmethod holds the treasury collection_method TEMPLATE use
// cases (treasury-domain-rebuild Stage 1, entity-layer-map.md Layer 7).
//
// Scope: template CRUD + page-data + the canonical lifecycle transitions
// (publish / close / archive / revise). NO eligibility-rule / grant / instance
// entities (later stages) and NO approval-gate logic (Stage 6) — per D-1.8 the
// transitions here are the canonical ones the Stage-6 gate will WRAP.
package collectionmethod

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// CollectionMethodRepositories groups all repository dependencies for collection method use cases.
type CollectionMethodRepositories struct {
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer
}

// CollectionMethodServices groups all business service dependencies.
type CollectionMethodServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all collection method-related use cases.
type UseCases struct {
	CreateCollectionMethod          *CreateCollectionMethodUseCase
	ReadCollectionMethod            *ReadCollectionMethodUseCase
	UpdateCollectionMethod          *UpdateCollectionMethodUseCase
	DeleteCollectionMethod          *DeleteCollectionMethodUseCase
	ListCollectionMethods           *ListCollectionMethodsUseCase
	GetCollectionMethodListPageData *GetCollectionMethodListPageDataUseCase
	GetCollectionMethodItemPageData *GetCollectionMethodItemPageDataUseCase

	// Lifecycle transitions (D-1.8).
	PublishCollectionMethod *PublishCollectionMethodUseCase
	CloseCollectionMethod   *CloseCollectionMethodUseCase
	ArchiveCollectionMethod *ArchiveCollectionMethodUseCase
	ReviseCollectionMethod  *ReviseCollectionMethodUseCase
}

// NewUseCases creates a new collection of collection method use cases.
func NewUseCases(
	repositories CollectionMethodRepositories,
	services CollectionMethodServices,
) *UseCases {
	createUC := NewCreateCollectionMethodUseCase(
		CreateCollectionMethodRepositories(repositories),
		CreateCollectionMethodServices{
			Authorizer:  services.Authorizer,
			Transactor:  services.Transactor,
			Translator:  services.Translator,
			IDGenerator: services.IDGenerator,
		},
	)

	readUC := NewReadCollectionMethodUseCase(
		ReadCollectionMethodRepositories(repositories),
		ReadCollectionMethodServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	updateUC := NewUpdateCollectionMethodUseCase(
		UpdateCollectionMethodRepositories(repositories),
		UpdateCollectionMethodServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	deleteUC := NewDeleteCollectionMethodUseCase(
		DeleteCollectionMethodRepositories(repositories),
		DeleteCollectionMethodServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	listUC := NewListCollectionMethodsUseCase(
		ListCollectionMethodsRepositories(repositories),
		ListCollectionMethodsServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	listPageDataUC := NewGetCollectionMethodListPageDataUseCase(
		GetCollectionMethodListPageDataRepositories(repositories),
		GetCollectionMethodListPageDataServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	itemPageDataUC := NewGetCollectionMethodItemPageDataUseCase(
		GetCollectionMethodItemPageDataRepositories(repositories),
		GetCollectionMethodItemPageDataServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	transitionRepos := TransitionCollectionMethodRepositories(repositories)
	transitionServices := TransitionCollectionMethodServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	return &UseCases{
		CreateCollectionMethod:          createUC,
		ReadCollectionMethod:            readUC,
		UpdateCollectionMethod:          updateUC,
		DeleteCollectionMethod:          deleteUC,
		ListCollectionMethods:           listUC,
		GetCollectionMethodListPageData: listPageDataUC,
		GetCollectionMethodItemPageData: itemPageDataUC,

		PublishCollectionMethod: NewPublishCollectionMethodUseCase(transitionRepos, transitionServices, updateUC),
		CloseCollectionMethod:   NewCloseCollectionMethodUseCase(transitionRepos, transitionServices, updateUC),
		ArchiveCollectionMethod: NewArchiveCollectionMethodUseCase(transitionRepos, transitionServices, updateUC),
		ReviseCollectionMethod:  NewReviseCollectionMethodUseCase(transitionRepos, transitionServices, createUC, updateUC),
	}
}
