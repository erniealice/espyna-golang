package journalentry

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	journalentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_entry"
)

// JournalEntryRepositories groups all repository dependencies for journal entry use cases
type JournalEntryRepositories struct {
	JournalEntry journalentrypb.JournalEntryDomainServiceServer // Primary entity repository
}

// JournalEntryServices groups all business service dependencies for journal entry use cases
type JournalEntryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadJournalEntryRepositories(repositories)
	readServices := ReadJournalEntryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateJournalEntryRepositories(repositories)
	updateServices := UpdateJournalEntryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteJournalEntryRepositories(repositories)
	deleteServices := DeleteJournalEntryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListJournalEntriesRepositories(repositories)
	listServices := ListJournalEntriesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetJournalEntryListPageDataRepositories(repositories)
	getListPageDataServices := GetJournalEntryListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	postRepos := PostJournalEntryRepositories(repositories)
	postServices := PostJournalEntryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	reverseRepos := ReverseJournalEntryRepositories(repositories)
	reverseServices := ReverseJournalEntryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
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
