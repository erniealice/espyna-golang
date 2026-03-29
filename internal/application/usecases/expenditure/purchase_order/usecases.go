package purchaseorder

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	purchaseorderpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order"
)

// PurchaseOrderRepositories groups all repository dependencies for purchase order use cases
type PurchaseOrderRepositories struct {
	PurchaseOrder purchaseorderpb.PurchaseOrderDomainServiceServer // Primary entity repository
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
	createRepos := CreatePurchaseOrderRepositories(repositories)
	createServices := CreatePurchaseOrderServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadPurchaseOrderRepositories(repositories)
	readServices := ReadPurchaseOrderServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdatePurchaseOrderRepositories(repositories)
	updateServices := UpdatePurchaseOrderServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeletePurchaseOrderRepositories(repositories)
	deleteServices := DeletePurchaseOrderServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListPurchaseOrdersRepositories(repositories)
	listServices := ListPurchaseOrdersServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetPurchaseOrderListPageDataRepositories(repositories)
	getListPageDataServices := GetPurchaseOrderListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getItemPageDataRepos := GetPurchaseOrderItemPageDataRepositories(repositories)
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
