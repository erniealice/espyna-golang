package payment

import (
	"leapfor.xyz/espyna/internal/application/ports"
	paymentpb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment"
	subscriptionpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/subscription"
)

// PaymentRepositories groups all repository dependencies for payment use cases
type PaymentRepositories struct {
	Payment      paymentpb.PaymentDomainServiceServer
	Subscription subscriptionpb.SubscriptionDomainServiceServer
}

// PaymentServices groups all business service dependencies for payment use cases
type PaymentServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all payment-related use cases
type UseCases struct {
	CreatePayment *CreatePaymentUseCase
	ReadPayment   *ReadPaymentUseCase
	UpdatePayment *UpdatePaymentUseCase
	DeletePayment *DeletePaymentUseCase
	ListPayments  *ListPaymentsUseCase
}

// NewUseCases creates a new collection of payment use cases
func NewUseCases(
	repositories PaymentRepositories,
	services PaymentServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreatePaymentRepositories{
		Payment:      repositories.Payment,
		Subscription: repositories.Subscription,
	}
	createServices := CreatePaymentServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadPaymentRepositories{
		Payment: repositories.Payment,
	}
	readServices := ReadPaymentServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	} // No IDService for Read

	updateRepos := UpdatePaymentRepositories{
		Payment:      repositories.Payment,
		Subscription: repositories.Subscription,
	}
	updateServices := UpdatePaymentServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	} // No IDService for Update

	deleteRepos := DeletePaymentRepositories{
		Payment: repositories.Payment,
	}
	deleteServices := DeletePaymentServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	} // No IDService for Delete

	listRepos := ListPaymentsRepositories{
		Payment: repositories.Payment,
	}
	listServices := ListPaymentsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	} // No IDService for List

	return &UseCases{
		CreatePayment: NewCreatePaymentUseCase(createRepos, createServices),
		ReadPayment:   NewReadPaymentUseCase(readRepos, readServices),
		UpdatePayment: NewUpdatePaymentUseCase(updateRepos, updateServices),
		DeletePayment: NewDeletePaymentUseCase(deleteRepos, deleteServices),
		ListPayments:  NewListPaymentsUseCase(listRepos, listServices),
	}
}
