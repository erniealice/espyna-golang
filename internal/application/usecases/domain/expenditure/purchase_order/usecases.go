package purchaseorder

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
	purchaseorderpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order"
)

// PurchaseOrderRepositories groups all repository dependencies for purchase order use cases
type PurchaseOrderRepositories struct {
	PurchaseOrder purchaseorderpb.PurchaseOrderDomainServiceServer // Primary entity repository
	PaymentTerm   paymenttermpb.PaymentTermDomainServiceServer
}

// PurchaseOrderServices groups all business service dependencies for purchase order use cases
type PurchaseOrderServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all purchase order-related use cases
type UseCases struct {
	CreatePurchaseOrder          *CreatePurchaseOrderUseCase
	ReadPurchaseOrder            *ReadPurchaseOrderUseCase
	UpdatePurchaseOrder          *UpdatePurchaseOrderUseCase
	DeletePurchaseOrder          *DeletePurchaseOrderUseCase
	ListPurchaseOrders           *ListPurchaseOrdersUseCase
	GetPurchaseOrderListPageData *GetPurchaseOrderListPageDataUseCase
	GetPurchaseOrderItemPageData *GetPurchaseOrderItemPageDataUseCase
}

// NewUseCases creates a new collection of purchase order use cases
func NewUseCases(
	repositories PurchaseOrderRepositories,
	services PurchaseOrderServices,
) *UseCases {
	createRepos := CreatePurchaseOrderRepositories{
		PurchaseOrder: repositories.PurchaseOrder,
		PaymentTerm:   repositories.PaymentTerm,
	}
	createServices := CreatePurchaseOrderServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadPurchaseOrderRepositories{
		PurchaseOrder: repositories.PurchaseOrder,
	}
	readServices := ReadPurchaseOrderServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdatePurchaseOrderRepositories{
		PurchaseOrder: repositories.PurchaseOrder,
	}
	updateServices := UpdatePurchaseOrderServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeletePurchaseOrderRepositories{
		PurchaseOrder: repositories.PurchaseOrder,
	}
	deleteServices := DeletePurchaseOrderServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListPurchaseOrdersRepositories{
		PurchaseOrder: repositories.PurchaseOrder,
	}
	listServices := ListPurchaseOrdersServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetPurchaseOrderListPageDataRepositories{
		PurchaseOrder: repositories.PurchaseOrder,
	}
	getListPageDataServices := GetPurchaseOrderListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getItemPageDataRepos := GetPurchaseOrderItemPageDataRepositories{
		PurchaseOrder: repositories.PurchaseOrder,
	}
	getItemPageDataServices := GetPurchaseOrderItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreatePurchaseOrder:          NewCreatePurchaseOrderUseCase(createRepos, createServices),
		ReadPurchaseOrder:            NewReadPurchaseOrderUseCase(readRepos, readServices),
		UpdatePurchaseOrder:          NewUpdatePurchaseOrderUseCase(updateRepos, updateServices),
		DeletePurchaseOrder:          NewDeletePurchaseOrderUseCase(deleteRepos, deleteServices),
		ListPurchaseOrders:           NewListPurchaseOrdersUseCase(listRepos, listServices),
		GetPurchaseOrderListPageData: NewGetPurchaseOrderListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetPurchaseOrderItemPageData: NewGetPurchaseOrderItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}
