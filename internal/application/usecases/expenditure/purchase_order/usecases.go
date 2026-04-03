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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadPurchaseOrderRepositories{
		PurchaseOrder: repositories.PurchaseOrder,
	}
	readServices := ReadPurchaseOrderServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdatePurchaseOrderRepositories{
		PurchaseOrder: repositories.PurchaseOrder,
	}
	updateServices := UpdatePurchaseOrderServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeletePurchaseOrderRepositories{
		PurchaseOrder: repositories.PurchaseOrder,
	}
	deleteServices := DeletePurchaseOrderServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListPurchaseOrdersRepositories{
		PurchaseOrder: repositories.PurchaseOrder,
	}
	listServices := ListPurchaseOrdersServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetPurchaseOrderListPageDataRepositories{
		PurchaseOrder: repositories.PurchaseOrder,
	}
	getListPageDataServices := GetPurchaseOrderListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getItemPageDataRepos := GetPurchaseOrderItemPageDataRepositories{
		PurchaseOrder: repositories.PurchaseOrder,
	}
	getItemPageDataServices := GetPurchaseOrderItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
