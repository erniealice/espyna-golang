package price_list

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pricelistpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_list"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
)

// UseCases contains all price list-related use cases
type UseCases struct {
	CreatePriceList          *CreatePriceListUseCase
	ReadPriceList            *ReadPriceListUseCase
	UpdatePriceList          *UpdatePriceListUseCase
	DeletePriceList          *DeletePriceListUseCase
	ListPriceLists           *ListPriceListsUseCase
	GetPriceListListPageData *GetPriceListListPageDataUseCase
	GetPriceListItemPageData *GetPriceListItemPageDataUseCase
	FindApplicablePriceList  *FindApplicablePriceListUseCase
}

// PriceListRepositories groups all repository dependencies for price list use cases
type PriceListRepositories struct {
	PriceList    pricelistpb.PriceListDomainServiceServer       // Primary entity repository
	PriceProduct priceproductpb.PriceProductDomainServiceServer // For item page data - to fetch prices
}

// PriceListServices groups all business service dependencies for price list use cases
type PriceListServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of price list use cases
func NewUseCases(
	repositories PriceListRepositories,
	services PriceListServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreatePriceListRepositories{
		PriceList: repositories.PriceList,
	}
	createServices := CreatePriceListServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor:  services.Transactor,
		Authorizer:  services.Authorizer,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadPriceListRepositories{
		PriceList: repositories.PriceList,
	}
	readServices := ReadPriceListServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor: services.Transactor,
		Authorizer: services.Authorizer,
		Translator: services.Translator,
	}

	updateRepos := UpdatePriceListRepositories{
		PriceList: repositories.PriceList,
	}
	updateServices := UpdatePriceListServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor: services.Transactor,
		Authorizer: services.Authorizer,
		Translator: services.Translator,
	}

	deleteRepos := DeletePriceListRepositories{
		PriceList: repositories.PriceList,
	}
	deleteServices := DeletePriceListServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor: services.Transactor,
		Authorizer: services.Authorizer,
		Translator: services.Translator,
	}

	listRepos := ListPriceListsRepositories{
		PriceList: repositories.PriceList,
	}
	listServices := ListPriceListsServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor: services.Transactor,
		Authorizer: services.Authorizer,
		Translator: services.Translator,
	}

	listPageDataRepos := GetPriceListListPageDataRepositories{
		PriceList: repositories.PriceList,
	}
	listPageDataServices := GetPriceListListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor: services.Transactor,
		Authorizer: services.Authorizer,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetPriceListItemPageDataRepositories{
		PriceList:    repositories.PriceList,
		PriceProduct: repositories.PriceProduct,
	}
	itemPageDataServices := GetPriceListItemPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor: services.Transactor,
		Authorizer: services.Authorizer,
		Translator: services.Translator,
	}

	findApplicableRepos := FindApplicablePriceListRepositories{
		PriceList: repositories.PriceList,
	}
	findApplicableServices := FindApplicablePriceListServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor: services.Transactor,
		Authorizer: services.Authorizer,
		Translator: services.Translator,
	}

	return &UseCases{
		CreatePriceList:          NewCreatePriceListUseCase(createRepos, createServices),
		ReadPriceList:            NewReadPriceListUseCase(readRepos, readServices),
		UpdatePriceList:          NewUpdatePriceListUseCase(updateRepos, updateServices),
		DeletePriceList:          NewDeletePriceListUseCase(deleteRepos, deleteServices),
		ListPriceLists:           NewListPriceListsUseCase(listRepos, listServices),
		GetPriceListListPageData: NewGetPriceListListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetPriceListItemPageData: NewGetPriceListItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
		FindApplicablePriceList:  NewFindApplicablePriceListUseCase(findApplicableRepos, findApplicableServices),
	}
}
