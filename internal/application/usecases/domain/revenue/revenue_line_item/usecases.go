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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadRevenueLineItemRepositories(repositories)
	readServices := ReadRevenueLineItemServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateRevenueLineItemRepositories(repositories)
	updateServices := UpdateRevenueLineItemServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteRevenueLineItemRepositories(repositories)
	deleteServices := DeleteRevenueLineItemServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListRevenueLineItemsRepositories(repositories)
	listServices := ListRevenueLineItemsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateRevenueLineItem: NewCreateRevenueLineItemUseCase(createRepos, createServices),
		ReadRevenueLineItem:   NewReadRevenueLineItemUseCase(readRepos, readServices),
		UpdateRevenueLineItem: NewUpdateRevenueLineItemUseCase(updateRepos, updateServices),
		DeleteRevenueLineItem: NewDeleteRevenueLineItemUseCase(deleteRepos, deleteServices),
		ListRevenueLineItems:  NewListRevenueLineItemsUseCase(listRepos, listServices),
	}
}
