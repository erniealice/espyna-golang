package criteria_threshold

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_threshold"
)

// CriteriaThresholdRepositories groups all repository dependencies
type CriteriaThresholdRepositories struct {
	CriteriaThreshold pb.CriteriaThresholdDomainServiceServer
}

// CriteriaThresholdServices groups all business service dependencies
type CriteriaThresholdServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadCriteriaThresholdRepositories{
		CriteriaThreshold: repositories.CriteriaThreshold,
	}
	readServices := ReadCriteriaThresholdServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateCriteriaThresholdRepositories{
		CriteriaThreshold: repositories.CriteriaThreshold,
	}
	updateServices := UpdateCriteriaThresholdServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteCriteriaThresholdRepositories{
		CriteriaThreshold: repositories.CriteriaThreshold,
	}
	deleteServices := DeleteCriteriaThresholdServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListCriteriaThresholdsRepositories{
		CriteriaThreshold: repositories.CriteriaThreshold,
	}
	listServices := ListCriteriaThresholdsServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetCriteriaThresholdListPageDataRepositories{
		CriteriaThreshold: repositories.CriteriaThreshold,
	}
	listPageDataServices := GetCriteriaThresholdListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetCriteriaThresholdItemPageDataRepositories{
		CriteriaThreshold: repositories.CriteriaThreshold,
	}
	itemPageDataServices := GetCriteriaThresholdItemPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listByCriteriaRepos := ListByCriteriaRepositories{
		CriteriaThreshold: repositories.CriteriaThreshold,
	}
	listByCriteriaServices := ListByCriteriaServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
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
