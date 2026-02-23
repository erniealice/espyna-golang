package revenuelineitem

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
)

// RevenueLineItemRepositories groups all repository dependencies
type RevenueLineItemRepositories struct {
	RevenueLineItem pb.RevenueLineItemDomainServiceServer
}

// RevenueLineItemServices groups all business service dependencies
type RevenueLineItemServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all revenue line item use cases
type UseCases struct {
	CreateRevenueLineItem *CreateRevenueLineItemUseCase
	ReadRevenueLineItem   *ReadRevenueLineItemUseCase
	UpdateRevenueLineItem *UpdateRevenueLineItemUseCase
	DeleteRevenueLineItem *DeleteRevenueLineItemUseCase
	ListRevenueLineItems  *ListRevenueLineItemsUseCase
}

// NewUseCases creates a new collection of revenue line item use cases
func NewUseCases(
	repositories RevenueLineItemRepositories,
	services RevenueLineItemServices,
) *UseCases {
	createRepos := CreateRevenueLineItemRepositories(repositories)
	createServices := CreateRevenueLineItemServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadRevenueLineItemRepositories(repositories)
	readServices := ReadRevenueLineItemServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateRevenueLineItemRepositories(repositories)
	updateServices := UpdateRevenueLineItemServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteRevenueLineItemRepositories(repositories)
	deleteServices := DeleteRevenueLineItemServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListRevenueLineItemsRepositories(repositories)
	listServices := ListRevenueLineItemsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateRevenueLineItem: NewCreateRevenueLineItemUseCase(createRepos, createServices),
		ReadRevenueLineItem:   NewReadRevenueLineItemUseCase(readRepos, readServices),
		UpdateRevenueLineItem: NewUpdateRevenueLineItemUseCase(updateRepos, updateServices),
		DeleteRevenueLineItem: NewDeleteRevenueLineItemUseCase(deleteRepos, deleteServices),
		ListRevenueLineItems:  NewListRevenueLineItemsUseCase(listRepos, listServices),
	}
}
