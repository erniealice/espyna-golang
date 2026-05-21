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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadPurchaseOrderLineItemRepositories(repositories)
	readServices := ReadPurchaseOrderLineItemServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdatePurchaseOrderLineItemRepositories(repositories)
	updateServices := UpdatePurchaseOrderLineItemServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeletePurchaseOrderLineItemRepositories(repositories)
	deleteServices := DeletePurchaseOrderLineItemServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListPurchaseOrderLineItemsRepositories(repositories)
	listServices := ListPurchaseOrderLineItemsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetPurchaseOrderLineItemListPageDataRepositories(repositories)
	getListPageDataServices := GetPurchaseOrderLineItemListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getItemPageDataRepos := GetPurchaseOrderLineItemItemPageDataRepositories(repositories)
	getItemPageDataServices := GetPurchaseOrderLineItemItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
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
