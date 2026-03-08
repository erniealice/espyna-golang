package disbursement

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

// DisbursementRepositories groups all repository dependencies for disbursement use cases
type DisbursementRepositories struct {
	Disbursement disbursementpb.DisbursementDomainServiceServer
}

// DisbursementServices groups all business service dependencies for disbursement use cases
type DisbursementServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all disbursement-related use cases
type UseCases struct {
	CreateDisbursement *CreateDisbursementUseCase
	ReadDisbursement   *ReadDisbursementUseCase
	UpdateDisbursement *UpdateDisbursementUseCase
	DeleteDisbursement *DeleteDisbursementUseCase
	ListDisbursements  *ListDisbursementsUseCase
}

// NewUseCases creates a new collection of disbursement use cases
func NewUseCases(
	repositories DisbursementRepositories,
	services DisbursementServices,
) *UseCases {
	createRepos := CreateDisbursementRepositories(repositories)
	createServices := CreateDisbursementServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadDisbursementRepositories(repositories)
	readServices := ReadDisbursementServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateDisbursementRepositories(repositories)
	updateServices := UpdateDisbursementServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteDisbursementRepositories(repositories)
	deleteServices := DeleteDisbursementServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListDisbursementsRepositories(repositories)
	listServices := ListDisbursementsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateDisbursement: NewCreateDisbursementUseCase(createRepos, createServices),
		ReadDisbursement:   NewReadDisbursementUseCase(readRepos, readServices),
		UpdateDisbursement: NewUpdateDisbursementUseCase(updateRepos, updateServices),
		DeleteDisbursement: NewDeleteDisbursementUseCase(deleteRepos, deleteServices),
		ListDisbursements:  NewListDisbursementsUseCase(listRepos, listServices),
	}
}
