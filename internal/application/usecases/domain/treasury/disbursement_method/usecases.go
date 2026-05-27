// Package disbursementmethod holds the treasury disbursement_method TEMPLATE
// use cases (treasury-domain-rebuild Stage 1, entity-layer-map.md Layer 7).
//
// Symmetric to the collection_method package minus audience_mode / eligibility
// (D-4.9 buying-side asymmetry). Scope: template CRUD + page-data + canonical
// lifecycle transitions (publish / close / archive / revise). NO instance
// entities (later stages), NO approval-gate logic (Stage 6).
package disbursementmethod

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	disbursementmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_method"
)

// DisbursementMethodRepositories groups all repository dependencies.
type DisbursementMethodRepositories struct {
	DisbursementMethod disbursementmethodpb.DisbursementMethodDomainServiceServer
}

// DisbursementMethodServices groups all business service dependencies.
type DisbursementMethodServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all disbursement method-related use cases.
type UseCases struct {
	CreateDisbursementMethod          *CreateDisbursementMethodUseCase
	ReadDisbursementMethod            *ReadDisbursementMethodUseCase
	UpdateDisbursementMethod          *UpdateDisbursementMethodUseCase
	DeleteDisbursementMethod          *DeleteDisbursementMethodUseCase
	ListDisbursementMethods           *ListDisbursementMethodsUseCase
	GetDisbursementMethodListPageData *GetDisbursementMethodListPageDataUseCase
	GetDisbursementMethodItemPageData *GetDisbursementMethodItemPageDataUseCase

	// Lifecycle transitions (D-1.8).
	PublishDisbursementMethod *PublishDisbursementMethodUseCase
	CloseDisbursementMethod   *CloseDisbursementMethodUseCase
	ArchiveDisbursementMethod *ArchiveDisbursementMethodUseCase
	ReviseDisbursementMethod  *ReviseDisbursementMethodUseCase
}

// NewUseCases creates a new collection of disbursement method use cases.
func NewUseCases(
	repositories DisbursementMethodRepositories,
	services DisbursementMethodServices,
) *UseCases {
	createUC := NewCreateDisbursementMethodUseCase(
		CreateDisbursementMethodRepositories(repositories),
		CreateDisbursementMethodServices{
			Authorizer:  services.Authorizer,
			Transactor:  services.Transactor,
			Translator:  services.Translator,
			IDGenerator: services.IDGenerator,
		},
	)

	readUC := NewReadDisbursementMethodUseCase(
		ReadDisbursementMethodRepositories(repositories),
		ReadDisbursementMethodServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	updateUC := NewUpdateDisbursementMethodUseCase(
		UpdateDisbursementMethodRepositories(repositories),
		UpdateDisbursementMethodServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	deleteUC := NewDeleteDisbursementMethodUseCase(
		DeleteDisbursementMethodRepositories(repositories),
		DeleteDisbursementMethodServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	listUC := NewListDisbursementMethodsUseCase(
		ListDisbursementMethodsRepositories(repositories),
		ListDisbursementMethodsServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	listPageDataUC := NewGetDisbursementMethodListPageDataUseCase(
		GetDisbursementMethodListPageDataRepositories(repositories),
		GetDisbursementMethodListPageDataServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	itemPageDataUC := NewGetDisbursementMethodItemPageDataUseCase(
		GetDisbursementMethodItemPageDataRepositories(repositories),
		GetDisbursementMethodItemPageDataServices{
			Authorizer: services.Authorizer,
			Transactor: services.Transactor,
			Translator: services.Translator,
		},
	)

	transitionRepos := TransitionDisbursementMethodRepositories(repositories)
	transitionServices := TransitionDisbursementMethodServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	return &UseCases{
		CreateDisbursementMethod:          createUC,
		ReadDisbursementMethod:            readUC,
		UpdateDisbursementMethod:          updateUC,
		DeleteDisbursementMethod:          deleteUC,
		ListDisbursementMethods:           listUC,
		GetDisbursementMethodListPageData: listPageDataUC,
		GetDisbursementMethodItemPageData: itemPageDataUC,

		PublishDisbursementMethod: NewPublishDisbursementMethodUseCase(transitionRepos, transitionServices, updateUC),
		CloseDisbursementMethod:   NewCloseDisbursementMethodUseCase(transitionRepos, transitionServices, updateUC),
		ArchiveDisbursementMethod: NewArchiveDisbursementMethodUseCase(transitionRepos, transitionServices, updateUC),
		ReviseDisbursementMethod:  NewReviseDisbursementMethodUseCase(transitionRepos, transitionServices, createUC, updateUC),
	}
}
