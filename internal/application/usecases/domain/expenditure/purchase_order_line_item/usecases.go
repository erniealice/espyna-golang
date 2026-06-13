package purchaseorderlineitem

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
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
	ActionGatekeeper *actiongate.ActionGatekeeper
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
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadPurchaseOrderLineItemRepositories(repositories)
	readServices := ReadPurchaseOrderLineItemServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdatePurchaseOrderLineItemRepositories(repositories)
	updateServices := UpdatePurchaseOrderLineItemServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeletePurchaseOrderLineItemRepositories(repositories)
	deleteServices := DeletePurchaseOrderLineItemServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListPurchaseOrderLineItemsRepositories(repositories)
	listServices := ListPurchaseOrderLineItemsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetPurchaseOrderLineItemListPageDataRepositories(repositories)
	getListPageDataServices := GetPurchaseOrderLineItemListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetPurchaseOrderLineItemItemPageDataRepositories(repositories)
	getItemPageDataServices := GetPurchaseOrderLineItemItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
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
