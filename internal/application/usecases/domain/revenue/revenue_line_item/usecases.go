package revenuelineitem

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
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
	ActionGatekeeper *actiongate.ActionGatekeeper
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
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadRevenueLineItemRepositories(repositories)
	readServices := ReadRevenueLineItemServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateRevenueLineItemRepositories(repositories)
	updateServices := UpdateRevenueLineItemServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteRevenueLineItemRepositories(repositories)
	deleteServices := DeleteRevenueLineItemServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListRevenueLineItemsRepositories(repositories)
	listServices := ListRevenueLineItemsServices{
		ActionGatekeeper: services.ActionGatekeeper,
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
