package revenuepayment

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_payment"
	agreementlinetermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/agreement_line_term"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
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

	// Legacy-payment guard collaborators (20260927-usage-and-pass-through-charges, build-spec §6.1,
	// AC-UC-33). All optional: with no Revenue repository the guard is off and behaviour is unchanged.
	Revenue               revenuepb.RevenueDomainServiceServer
	CollectionApplication collectionapplicationpb.CollectionApplicationDomainServiceServer
	AgreementLineTerm     agreementlinetermpb.AgreementLineTermDomainServiceServer
}

// RevenuePaymentServices groups all business service dependencies
type RevenuePaymentServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator      ports.IDGenerator
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
	createRepos := CreateRevenuePaymentRepositories{RevenuePayment: repositories.RevenuePayment, Guard: guardFrom(repositories)}
	createServices := CreateRevenuePaymentServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		IDGenerator:      services.IDGenerator,
	}

	readRepos := ReadRevenuePaymentRepositories{RevenuePayment: repositories.RevenuePayment}
	readServices := ReadRevenuePaymentServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	updateRepos := UpdateRevenuePaymentRepositories{RevenuePayment: repositories.RevenuePayment, Guard: guardFrom(repositories)}
	updateServices := UpdateRevenuePaymentServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	deleteRepos := DeleteRevenuePaymentRepositories{RevenuePayment: repositories.RevenuePayment, Guard: guardFrom(repositories)}
	deleteServices := DeleteRevenuePaymentServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	listRepos := ListRevenuePaymentsRepositories{RevenuePayment: repositories.RevenuePayment}
	listServices := ListRevenuePaymentsServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	getListPageDataRepos := GetRevenuePaymentListPageDataRepositories{RevenuePayment: repositories.RevenuePayment}
	getListPageDataServices := GetRevenuePaymentListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	getItemPageDataRepos := GetRevenuePaymentItemPageDataRepositories{RevenuePayment: repositories.RevenuePayment}
	getItemPageDataServices := GetRevenuePaymentItemPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
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
