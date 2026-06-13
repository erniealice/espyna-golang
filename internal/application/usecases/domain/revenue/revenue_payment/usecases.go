package revenuepayment

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_payment"
)

// entityRevenuePayment is the authorization entity id for revenue_payment.
//
// NOTE: the canonical const entityid.RevenuePayment is owned by the Wire wave
// (design doc §190 adds it to registry/entityid/entityid.go alongside the other
// Revenue-domain consts + RevenueEntities). Until that lands, this package mirrors
// the sibling revenue_line_item convention (local const) so it compiles both in
// isolation and in the aggregate without a hard dependency on the Wire wave.
const entityRevenuePayment = "revenue_payment"

// RevenuePaymentRepositories groups all repository dependencies
type RevenuePaymentRepositories struct {
	RevenuePayment pb.RevenuePaymentDomainServiceServer // Primary entity repository
}

// RevenuePaymentServices groups all business service dependencies
type RevenuePaymentServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all revenue payment use cases
type UseCases struct {
	CreateRevenuePayment          *CreateRevenuePaymentUseCase
	ReadRevenuePayment            *ReadRevenuePaymentUseCase
	UpdateRevenuePayment          *UpdateRevenuePaymentUseCase
	DeleteRevenuePayment          *DeleteRevenuePaymentUseCase
	ListRevenuePayments           *ListRevenuePaymentsUseCase
	GetRevenuePaymentListPageData *GetRevenuePaymentListPageDataUseCase
	GetRevenuePaymentItemPageData *GetRevenuePaymentItemPageDataUseCase
}

// NewUseCases creates a new collection of revenue payment use cases
func NewUseCases(
	repositories RevenuePaymentRepositories,
	services RevenuePaymentServices,
) *UseCases {
	createRepos := CreateRevenuePaymentRepositories(repositories)
	createServices := CreateRevenuePaymentServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadRevenuePaymentRepositories(repositories)
	readServices := ReadRevenuePaymentServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateRevenuePaymentRepositories(repositories)
	updateServices := UpdateRevenuePaymentServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteRevenuePaymentRepositories(repositories)
	deleteServices := DeleteRevenuePaymentServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListRevenuePaymentsRepositories(repositories)
	listServices := ListRevenuePaymentsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetRevenuePaymentListPageDataRepositories(repositories)
	getListPageDataServices := GetRevenuePaymentListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getItemPageDataRepos := GetRevenuePaymentItemPageDataRepositories(repositories)
	getItemPageDataServices := GetRevenuePaymentItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateRevenuePayment:          NewCreateRevenuePaymentUseCase(createRepos, createServices),
		ReadRevenuePayment:            NewReadRevenuePaymentUseCase(readRepos, readServices),
		UpdateRevenuePayment:          NewUpdateRevenuePaymentUseCase(updateRepos, updateServices),
		DeleteRevenuePayment:          NewDeleteRevenuePaymentUseCase(deleteRepos, deleteServices),
		ListRevenuePayments:           NewListRevenuePaymentsUseCase(listRepos, listServices),
		GetRevenuePaymentListPageData: NewGetRevenuePaymentListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetRevenuePaymentItemPageData: NewGetRevenuePaymentItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}
