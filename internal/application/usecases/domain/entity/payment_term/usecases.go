package payment_term

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
)

// PaymentTermRepositories groups all repository dependencies for payment term use cases
type PaymentTermRepositories struct {
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer // Primary entity repository
}

// PaymentTermServices groups all business service dependencies for payment term use cases
type PaymentTermServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all payment-term-related use cases
type UseCases struct {
	CreatePaymentTerm          *CreatePaymentTermUseCase
	ReadPaymentTerm            *ReadPaymentTermUseCase
	UpdatePaymentTerm          *UpdatePaymentTermUseCase
	DeletePaymentTerm          *DeletePaymentTermUseCase
	ListPaymentTerms           *ListPaymentTermsUseCase
	GetPaymentTermListPageData *GetPaymentTermListPageDataUseCase
	GetPaymentTermItemPageData *GetPaymentTermItemPageDataUseCase
}

// NewUseCases creates a new collection of payment term use cases
func NewUseCases(
	repositories PaymentTermRepositories,
	services PaymentTermServices,
) *UseCases {
	createRepos := CreatePaymentTermRepositories{
		PaymentTerm: repositories.PaymentTerm,
	}
	createServices := CreatePaymentTermServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadPaymentTermRepositories{
		PaymentTerm: repositories.PaymentTerm,
	}
	readServices := ReadPaymentTermServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdatePaymentTermRepositories{
		PaymentTerm: repositories.PaymentTerm,
	}
	updateServices := UpdatePaymentTermServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeletePaymentTermRepositories{
		PaymentTerm: repositories.PaymentTerm,
	}
	deleteServices := DeletePaymentTermServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListPaymentTermsRepositories{
		PaymentTerm: repositories.PaymentTerm,
	}
	listServices := ListPaymentTermsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetPaymentTermListPageDataRepositories{
		PaymentTerm: repositories.PaymentTerm,
	}
	getListPageDataServices := GetPaymentTermListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetPaymentTermItemPageDataRepositories{
		PaymentTerm: repositories.PaymentTerm,
	}
	getItemPageDataServices := GetPaymentTermItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreatePaymentTerm:          NewCreatePaymentTermUseCase(createRepos, createServices),
		ReadPaymentTerm:            NewReadPaymentTermUseCase(readRepos, readServices),
		UpdatePaymentTerm:          NewUpdatePaymentTermUseCase(updateRepos, updateServices),
		DeletePaymentTerm:          NewDeletePaymentTermUseCase(deleteRepos, deleteServices),
		ListPaymentTerms:           NewListPaymentTermsUseCase(listRepos, listServices),
		GetPaymentTermListPageData: NewGetPaymentTermListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetPaymentTermItemPageData: NewGetPaymentTermItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of payment term use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(paymentTermRepo paymenttermpb.PaymentTermDomainServiceServer) *UseCases {
	repositories := PaymentTermRepositories{
		PaymentTerm: paymentTermRepo,
	}

	services := PaymentTermServices{
		Authorizer:  nil,
		Transactor:  ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
		IDGenerator: ports.NewNoOpIDGenerator(),
	}

	return NewUseCases(repositories, services)
}
