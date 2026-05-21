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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadRevenueAttributeRepositories(repositories)
	readServices := ReadRevenueAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateRevenueAttributeRepositories(repositories)
	updateServices := UpdateRevenueAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteRevenueAttributeRepositories(repositories)
	deleteServices := DeleteRevenueAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListRevenueAttributesRepositories(repositories)
	listServices := ListRevenueAttributesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateRevenueAttribute: NewCreateRevenueAttributeUseCase(createRepos, createServices),
		ReadRevenueAttribute:   NewReadRevenueAttributeUseCase(readRepos, readServices),
		UpdateRevenueAttribute: NewUpdateRevenueAttributeUseCase(updateRepos, updateServices),
		DeleteRevenueAttribute: NewDeleteRevenueAttributeUseCase(deleteRepos, deleteServices),
		ListRevenueAttributes:  NewListRevenueAttributesUseCase(listRepos, listServices),
	}
}
