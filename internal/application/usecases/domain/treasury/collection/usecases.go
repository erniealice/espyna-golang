package collection

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

// CollectionRepositories groups all repository dependencies for collection use cases
type CollectionRepositories struct {
	Collection collectionpb.CollectionDomainServiceServer
	// CollectionApplication lets update/delete refuse a receipt that anchors APPLIED applications
	// (build-spec §7c C25). Nil on providers without the S1 tables (nothing to protect there).
	CollectionApplication collectionapplicationpb.CollectionApplicationDomainServiceServer
}

// CollectionServices groups all business service dependencies for collection use cases
type CollectionServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator      ports.IDGenerator
}

// UseCases contains all collection-related use cases
type UseCases struct {
	CreateCollection *CreateCollectionUseCase
	ReadCollection   *ReadCollectionUseCase
	UpdateCollection *UpdateCollectionUseCase
	DeleteCollection *DeleteCollectionUseCase
	ListCollections  *ListCollectionsUseCase
	ListByClient     *ListByClientUseCase

	// 20260518-hexagonal-strict-adherence Phase 1.C — advance use cases (selling
	// side) folded back into the entity sub-aggregate from the prior
	// treasury_collection/ parallel home. Constructed by the treasury aggregator
	// (treasury.NewUseCases) after the CRUD use cases above are wired, because
	// the advance flows depend on UpdateCollection per Q1-B caller routing.
	AmortizeAdvance           *AmortizeAdvanceCollectionUseCase
	SettleUnscheduledAdvance  *SettleUnscheduledAdvanceUseCase
	RefundUnscheduledAdvance  *RefundUnscheduledAdvanceUseCase
	CancelAdvance             *CancelAdvanceUseCase
	RecognizeMilestoneAdvance *RecognizeMilestoneAdvanceCollectionUseCase
	ListAdvancesForDashboard  *ListAdvanceCollectionsForDashboardUseCase
}

// NewUseCases creates a new collection of collection use cases
func NewUseCases(
	repositories CollectionRepositories,
	services CollectionServices,
) *UseCases {
	createRepos := CreateCollectionRepositories{Collection: repositories.Collection}
	createServices := CreateCollectionServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		IDGenerator:      services.IDGenerator,
	}

	readRepos := ReadCollectionRepositories{Collection: repositories.Collection}
	readServices := ReadCollectionServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	updateRepos := UpdateCollectionRepositories(repositories)
	updateServices := UpdateCollectionServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	deleteRepos := DeleteCollectionRepositories(repositories)
	deleteServices := DeleteCollectionServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	listRepos := ListCollectionsRepositories{Collection: repositories.Collection}
	listServices := ListCollectionsServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	listByClientRepos := ListByClientRepositories{
		Collection: repositories.Collection,
	}
	listByClientServices := ListByClientServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Translator:       services.Translator,
	}

	return &UseCases{
		CreateCollection: NewCreateCollectionUseCase(createRepos, createServices),
		ReadCollection:   NewReadCollectionUseCase(readRepos, readServices),
		UpdateCollection: NewUpdateCollectionUseCase(updateRepos, updateServices),
		DeleteCollection: NewDeleteCollectionUseCase(deleteRepos, deleteServices),
		ListCollections:  NewListCollectionsUseCase(listRepos, listServices),
		ListByClient:     NewListByClientUseCase(listByClientRepos, listByClientServices),
	}
}
