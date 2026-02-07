package payment_method

import (
	"leapfor.xyz/espyna/internal/application/ports"
	paymentmethodpb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_method"
)

// PaymentMethodRepositories groups all repository dependencies for payment method use cases
type PaymentMethodRepositories struct {
	PaymentMethod paymentmethodpb.PaymentMethodDomainServiceServer // Primary entity repository
}

// PaymentMethodServices groups all business service dependencies for payment method use cases
type PaymentMethodServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// UseCases contains all payment method-related use cases
type UseCases struct {
	CreatePaymentMethod *CreatePaymentMethodUseCase
	ReadPaymentMethod   *ReadPaymentMethodUseCase
	UpdatePaymentMethod *UpdatePaymentMethodUseCase
	DeletePaymentMethod *DeletePaymentMethodUseCase
	ListPaymentMethods  *ListPaymentMethodsUseCase
}

// NewUseCases creates a new collection of payment method use cases
func NewUseCases(
	repositories PaymentMethodRepositories,
	services PaymentMethodServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreatePaymentMethodRepositories(repositories)
	createServices := CreatePaymentMethodServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	readRepos := ReadPaymentMethodRepositories(repositories)
	readServices := ReadPaymentMethodServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	} // No IDService for Read

	updateRepos := UpdatePaymentMethodRepositories(repositories)
	updateServices := UpdatePaymentMethodServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	} // No IDService for Update

	deleteRepos := DeletePaymentMethodRepositories(repositories)
	deleteServices := DeletePaymentMethodServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	} // No IDService for Delete

	listRepos := ListPaymentMethodsRepositories(repositories)
	listServices := ListPaymentMethodsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	} // No IDService for List

	return &UseCases{
		CreatePaymentMethod: NewCreatePaymentMethodUseCase(createRepos, createServices),
		ReadPaymentMethod:   NewReadPaymentMethodUseCase(readRepos, readServices),
		UpdatePaymentMethod: NewUpdatePaymentMethodUseCase(updateRepos, updateServices),
		DeletePaymentMethod: NewDeletePaymentMethodUseCase(deleteRepos, deleteServices),
		ListPaymentMethods:  NewListPaymentMethodsUseCase(listRepos, listServices),
	}
}
