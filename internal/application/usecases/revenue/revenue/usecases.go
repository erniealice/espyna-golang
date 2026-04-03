package revenue

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
)

// RevenueRepositories groups all repository dependencies for revenue use cases
type RevenueRepositories struct {
	Revenue     revenuepb.RevenueDomainServiceServer
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer
}

// RevenueServices groups all business service dependencies for revenue use cases
type RevenueServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all revenue-related use cases
type UseCases struct {
	CreateRevenue          *CreateRevenueUseCase
	ReadRevenue            *ReadRevenueUseCase
	UpdateRevenue          *UpdateRevenueUseCase
	DeleteRevenue          *DeleteRevenueUseCase
	ListRevenues           *ListRevenuesUseCase
	GetRevenueListPageData *GetRevenueListPageDataUseCase
}

// NewUseCases creates a new collection of revenue use cases
func NewUseCases(
	repositories RevenueRepositories,
	services RevenueServices,
) *UseCases {
	createRepos := CreateRevenueRepositories{
		Revenue:     repositories.Revenue,
		PaymentTerm: repositories.PaymentTerm,
	}
	createServices := CreateRevenueServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadRevenueRepositories{
		Revenue: repositories.Revenue,
	}
	readServices := ReadRevenueServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateRevenueRepositories{
		Revenue: repositories.Revenue,
	}
	updateServices := UpdateRevenueServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteRevenueRepositories{
		Revenue: repositories.Revenue,
	}
	deleteServices := DeleteRevenueServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListRevenuesRepositories{
		Revenue: repositories.Revenue,
	}
	listServices := ListRevenuesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetRevenueListPageDataRepositories{
		Revenue: repositories.Revenue,
	}
	getListPageDataServices := GetRevenueListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateRevenue:          NewCreateRevenueUseCase(createRepos, createServices),
		ReadRevenue:            NewReadRevenueUseCase(readRepos, readServices),
		UpdateRevenue:          NewUpdateRevenueUseCase(updateRepos, updateServices),
		DeleteRevenue:          NewDeleteRevenueUseCase(deleteRepos, deleteServices),
		ListRevenues:           NewListRevenuesUseCase(listRepos, listServices),
		GetRevenueListPageData: NewGetRevenueListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
