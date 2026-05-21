package revenueattribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_attribute"
)

// RevenueAttributeRepositories groups all repository dependencies
type RevenueAttributeRepositories struct {
	RevenueAttribute pb.RevenueAttributeDomainServiceServer
}

// RevenueAttributeServices groups all business service dependencies
type RevenueAttributeServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all revenue attribute use cases
type UseCases struct {
	CreateRevenueAttribute *CreateRevenueAttributeUseCase
	ReadRevenueAttribute   *ReadRevenueAttributeUseCase
	UpdateRevenueAttribute *UpdateRevenueAttributeUseCase
	DeleteRevenueAttribute *DeleteRevenueAttributeUseCase
	ListRevenueAttributes  *ListRevenueAttributesUseCase
}

// NewUseCases creates a new collection of revenue attribute use cases
func NewUseCases(
	repositories RevenueAttributeRepositories,
	services RevenueAttributeServices,
) *UseCases {
	createRepos := CreateRevenueAttributeRepositories(repositories)
	createServices := CreateRevenueAttributeServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadRevenueAttributeRepositories(repositories)
	readServices := ReadRevenueAttributeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateRevenueAttributeRepositories(repositories)
	updateServices := UpdateRevenueAttributeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteRevenueAttributeRepositories(repositories)
	deleteServices := DeleteRevenueAttributeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListRevenueAttributesRepositories(repositories)
	listServices := ListRevenueAttributesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateRevenueAttribute: NewCreateRevenueAttributeUseCase(createRepos, createServices),
		ReadRevenueAttribute:   NewReadRevenueAttributeUseCase(readRepos, readServices),
		UpdateRevenueAttribute: NewUpdateRevenueAttributeUseCase(updateRepos, updateServices),
		DeleteRevenueAttribute: NewDeleteRevenueAttributeUseCase(deleteRepos, deleteServices),
		ListRevenueAttributes:  NewListRevenueAttributesUseCase(listRepos, listServices),
	}
}
