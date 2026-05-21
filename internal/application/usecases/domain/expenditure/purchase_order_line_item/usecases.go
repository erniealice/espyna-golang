package purchaseorderlineitem

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	purchaseorderlineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order_line_item"
)

// PurchaseOrderLineItemRepositories groups all repository dependencies for purchase order line item use cases
type PurchaseOrderLineItemRepositories struct {
	PurchaseOrderLineItem purchaseorderlineitempb.PurchaseOrderLineItemDomainServiceServer // Primary entity repository
}

// PurchaseOrderLineItemServices groups all business service dependencies for purchase order line item use cases
type PurchaseOrderLineItemServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all purchase order line item-related use cases
type UseCases struct {
	CreatePurchaseOrderLineItem          *CreatePurchaseOrderLineItemUseCase
	ReadPurchaseOrderLineItem            *ReadPurchaseOrderLineItemUseCase
	UpdatePurchaseOrderLineItem          *UpdatePurchaseOrderLineItemUseCase
	DeletePurchaseOrderLineItem          *DeletePurchaseOrderLineItemUseCase
	ListPurchaseOrderLineItems           *ListPurchaseOrderLineItemsUseCase
	GetPurchaseOrderLineItemListPageData *GetPurchaseOrderLineItemListPageDataUseCase
	GetPurchaseOrderLineItemItemPageData *GetPurchaseOrderLineItemItemPageDataUseCase
}

// NewUseCases creates a new collection of purchase order line item use cases
func NewUseCases(
	repositories PurchaseOrderLineItemRepositories,
	services PurchaseOrderLineItemServices,
) *UseCases {
	createRepos := CreatePurchaseOrderLineItemRepositories(repositories)
	createServices := CreatePurchaseOrderLineItemServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadPurchaseOrderLineItemRepositories(repositories)
	readServices := ReadPurchaseOrderLineItemServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdatePurchaseOrderLineItemRepositories(repositories)
	updateServices := UpdatePurchaseOrderLineItemServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeletePurchaseOrderLineItemRepositories(repositories)
	deleteServices := DeletePurchaseOrderLineItemServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListPurchaseOrderLineItemsRepositories(repositories)
	listServices := ListPurchaseOrderLineItemsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetPurchaseOrderLineItemListPageDataRepositories(repositories)
	getListPageDataServices := GetPurchaseOrderLineItemListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getItemPageDataRepos := GetPurchaseOrderLineItemItemPageDataRepositories(repositories)
	getItemPageDataServices := GetPurchaseOrderLineItemItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreatePurchaseOrderLineItem:          NewCreatePurchaseOrderLineItemUseCase(createRepos, createServices),
		ReadPurchaseOrderLineItem:            NewReadPurchaseOrderLineItemUseCase(readRepos, readServices),
		UpdatePurchaseOrderLineItem:          NewUpdatePurchaseOrderLineItemUseCase(updateRepos, updateServices),
		DeletePurchaseOrderLineItem:          NewDeletePurchaseOrderLineItemUseCase(deleteRepos, deleteServices),
		ListPurchaseOrderLineItems:           NewListPurchaseOrderLineItemsUseCase(listRepos, listServices),
		GetPurchaseOrderLineItemListPageData: NewGetPurchaseOrderLineItemListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetPurchaseOrderLineItemItemPageData: NewGetPurchaseOrderLineItemItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}
