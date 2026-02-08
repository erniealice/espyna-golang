package price_list

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
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
}

// PriceListRepositories groups all repository dependencies for price list use cases
type PriceListRepositories struct {
	PriceList    pricelistpb.PriceListDomainServiceServer       // Primary entity repository
	PriceProduct priceproductpb.PriceProductDomainServiceServer // For item page data - to fetch prices
}

// PriceListServices groups all business service dependencies for price list use cases
type PriceListServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		TransactionService:   services.TransactionService,
		AuthorizationService: services.AuthorizationService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadPriceListRepositories{
		PriceList: repositories.PriceList,
	}
	readServices := ReadPriceListServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	updateRepos := UpdatePriceListRepositories{
		PriceList: repositories.PriceList,
	}
	updateServices := UpdatePriceListServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	deleteRepos := DeletePriceListRepositories{
		PriceList: repositories.PriceList,
	}
	deleteServices := DeletePriceListServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listRepos := ListPriceListsRepositories{
		PriceList: repositories.PriceList,
	}
	listServices := ListPriceListsServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listPageDataRepos := GetPriceListListPageDataRepositories{
		PriceList: repositories.PriceList,
	}
	listPageDataServices := GetPriceListListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	itemPageDataRepos := GetPriceListItemPageDataRepositories{
		PriceList:    repositories.PriceList,
		PriceProduct: repositories.PriceProduct,
	}
	itemPageDataServices := GetPriceListItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	return &UseCases{
		CreatePriceList:          NewCreatePriceListUseCase(createRepos, createServices),
		ReadPriceList:            NewReadPriceListUseCase(readRepos, readServices),
		UpdatePriceList:          NewUpdatePriceListUseCase(updateRepos, updateServices),
		DeletePriceList:          NewDeletePriceListUseCase(deleteRepos, deleteServices),
		ListPriceLists:           NewListPriceListsUseCase(listRepos, listServices),
		GetPriceListListPageData: NewGetPriceListListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetPriceListItemPageData: NewGetPriceListItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}
