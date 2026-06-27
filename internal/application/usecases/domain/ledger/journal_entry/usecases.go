package journalentry

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	journalentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_entry"
)

// JournalEntryRepositories groups all repository dependencies for journal entry use cases
type JournalEntryRepositories struct {
	JournalEntry journalentrypb.JournalEntryDomainServiceServer // Primary entity repository
}

// JournalEntryServices groups all business service dependencies for journal entry use cases
type JournalEntryServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all journal-entry-related use cases
type UseCases struct {
	CreateJournalEntry          *CreateJournalEntryUseCase
	ReadJournalEntry            *ReadJournalEntryUseCase
	UpdateJournalEntry          *UpdateJournalEntryUseCase
	DeleteJournalEntry          *DeleteJournalEntryUseCase
	ListJournalEntries          *ListJournalEntriesUseCase
	GetJournalEntryListPageData *GetJournalEntryListPageDataUseCase
	PostJournalEntry            *PostJournalEntryUseCase
	ReverseJournalEntry         *ReverseJournalEntryUseCase
}

// NewUseCases creates a new collection of journal entry use cases
func NewUseCases(
	repositories JournalEntryRepositories,
	services JournalEntryServices,
) *UseCases {
	createRepos := CreateJournalEntryRepositories(repositories)
	createServices := CreateJournalEntryServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadJournalEntryRepositories(repositories)
	readServices := ReadJournalEntryServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateJournalEntryRepositories(repositories)
	updateServices := UpdateJournalEntryServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteJournalEntryRepositories(repositories)
	deleteServices := DeleteJournalEntryServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListJournalEntriesRepositories(repositories)
	listServices := ListJournalEntriesServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetJournalEntryListPageDataRepositories(repositories)
	getListPageDataServices := GetJournalEntryListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	postRepos := PostJournalEntryRepositories(repositories)
	postServices := PostJournalEntryServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	reverseRepos := ReverseJournalEntryRepositories(repositories)
	reverseServices := ReverseJournalEntryServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	return &UseCases{
		CreateJournalEntry:          NewCreateJournalEntryUseCase(createRepos, createServices),
		ReadJournalEntry:            NewReadJournalEntryUseCase(readRepos, readServices),
		UpdateJournalEntry:          NewUpdateJournalEntryUseCase(updateRepos, updateServices),
		DeleteJournalEntry:          NewDeleteJournalEntryUseCase(deleteRepos, deleteServices),
		ListJournalEntries:          NewListJournalEntriesUseCase(listRepos, listServices),
		GetJournalEntryListPageData: NewGetJournalEntryListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		PostJournalEntry:            NewPostJournalEntryUseCase(postRepos, postServices),
		ReverseJournalEntry:         NewReverseJournalEntryUseCase(reverseRepos, reverseServices),
	}
}
