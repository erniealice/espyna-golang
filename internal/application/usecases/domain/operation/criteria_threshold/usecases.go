package criteria_threshold

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_threshold"
)

// CriteriaThresholdRepositories groups all repository dependencies
type CriteriaThresholdRepositories struct {
	CriteriaThreshold pb.CriteriaThresholdDomainServiceServer
}

// CriteriaThresholdServices groups all business service dependencies
type CriteriaThresholdServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all criteria_threshold-related use cases
type UseCases struct {
	CreateCriteriaThreshold          *CreateCriteriaThresholdUseCase
	ReadCriteriaThreshold            *ReadCriteriaThresholdUseCase
	UpdateCriteriaThreshold          *UpdateCriteriaThresholdUseCase
	DeleteCriteriaThreshold          *DeleteCriteriaThresholdUseCase
	ListCriteriaThresholds           *ListCriteriaThresholdsUseCase
	GetCriteriaThresholdListPageData *GetCriteriaThresholdListPageDataUseCase
	GetCriteriaThresholdItemPageData *GetCriteriaThresholdItemPageDataUseCase
	ListByCriteria                   *ListByCriteriaUseCase
}

// NewUseCases creates a new collection of criteria_threshold use cases
func NewUseCases(
	repositories CriteriaThresholdRepositories,
	services CriteriaThresholdServices,
) *UseCases {
	createRepos := CreateCriteriaThresholdRepositories{
		CriteriaThreshold: repositories.CriteriaThreshold,
	}
	createServices := CreateCriteriaThresholdServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadCriteriaThresholdRepositories{
		CriteriaThreshold: repositories.CriteriaThreshold,
	}
	readServices := ReadCriteriaThresholdServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateCriteriaThresholdRepositories{
		CriteriaThreshold: repositories.CriteriaThreshold,
	}
	updateServices := UpdateCriteriaThresholdServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteCriteriaThresholdRepositories{
		CriteriaThreshold: repositories.CriteriaThreshold,
	}
	deleteServices := DeleteCriteriaThresholdServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListCriteriaThresholdsRepositories{
		CriteriaThreshold: repositories.CriteriaThreshold,
	}
	listServices := ListCriteriaThresholdsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetCriteriaThresholdListPageDataRepositories{
		CriteriaThreshold: repositories.CriteriaThreshold,
	}
	listPageDataServices := GetCriteriaThresholdListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	itemPageDataRepos := GetCriteriaThresholdItemPageDataRepositories{
		CriteriaThreshold: repositories.CriteriaThreshold,
	}
	itemPageDataServices := GetCriteriaThresholdItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listByCriteriaRepos := ListByCriteriaRepositories{
		CriteriaThreshold: repositories.CriteriaThreshold,
	}
	listByCriteriaServices := ListByCriteriaServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateCriteriaThreshold:          NewCreateCriteriaThresholdUseCase(createRepos, createServices),
		ReadCriteriaThreshold:            NewReadCriteriaThresholdUseCase(readRepos, readServices),
		UpdateCriteriaThreshold:          NewUpdateCriteriaThresholdUseCase(updateRepos, updateServices),
		DeleteCriteriaThreshold:          NewDeleteCriteriaThresholdUseCase(deleteRepos, deleteServices),
		ListCriteriaThresholds:           NewListCriteriaThresholdsUseCase(listRepos, listServices),
		GetCriteriaThresholdListPageData: NewGetCriteriaThresholdListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetCriteriaThresholdItemPageData: NewGetCriteriaThresholdItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
		ListByCriteria:                   NewListByCriteriaUseCase(listByCriteriaRepos, listByCriteriaServices),
	}
}
