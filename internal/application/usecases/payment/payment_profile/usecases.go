package payment_profile

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	paymentMethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_method"
	paymentProfilepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_profile"
)

// PaymentProfileRepositories groups all repository dependencies for payment profile use cases
type PaymentProfileRepositories struct {
	PaymentProfile paymentProfilepb.PaymentProfileDomainServiceServer // Primary entity repository
	Client         clientpb.ClientDomainServiceServer
	PaymentMethod  paymentMethodpb.PaymentMethodDomainServiceServer
}

// PaymentProfileServices groups all business service dependencies for payment profile use cases
type PaymentProfileServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// UseCases contains all payment profile-related use cases
type UseCases struct {
	CreatePaymentProfile          *CreatePaymentProfileUseCase
	ReadPaymentProfile            *ReadPaymentProfileUseCase
	UpdatePaymentProfile          *UpdatePaymentProfileUseCase
	DeletePaymentProfile          *DeletePaymentProfileUseCase
	ListPaymentProfiles           *ListPaymentProfilesUseCase
	GetPaymentProfileListPageData *GetPaymentProfileListPageDataUseCase
	GetPaymentProfileItemPageData *GetPaymentProfileItemPageDataUseCase
}

// NewUseCases creates a new collection of payment profile use cases
func NewUseCases(
	repositories PaymentProfileRepositories,
	services PaymentProfileServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreatePaymentProfileRepositories(repositories)
	createServices := CreatePaymentProfileServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	readRepos := ReadPaymentProfileRepositories{
		PaymentProfile: repositories.PaymentProfile,
	}
	readServices := ReadPaymentProfileServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	} // No IDService for Read

	updateRepos := UpdatePaymentProfileRepositories(repositories)
	updateServices := UpdatePaymentProfileServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	} // No IDService for Update

	deleteRepos := DeletePaymentProfileRepositories{
		PaymentProfile: repositories.PaymentProfile,
	}
	deleteServices := DeletePaymentProfileServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	} // No IDService for Delete

	listRepos := ListPaymentProfilesRepositories{
		PaymentProfile: repositories.PaymentProfile,
	}
	listServices := ListPaymentProfilesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	} // No IDService for List

	listPageDataRepos := GetPaymentProfileListPageDataRepositories{
		PaymentProfile: repositories.PaymentProfile,
	}
	listPageDataServices := GetPaymentProfileListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	itemPageDataRepos := GetPaymentProfileItemPageDataRepositories{
		PaymentProfile: repositories.PaymentProfile,
	}
	itemPageDataServices := GetPaymentProfileItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	return &UseCases{
		CreatePaymentProfile:          NewCreatePaymentProfileUseCase(createRepos, createServices),
		ReadPaymentProfile:            NewReadPaymentProfileUseCase(readRepos, readServices),
		UpdatePaymentProfile:          NewUpdatePaymentProfileUseCase(updateRepos, updateServices),
		DeletePaymentProfile:          NewDeletePaymentProfileUseCase(deleteRepos, deleteServices),
		ListPaymentProfiles:           NewListPaymentProfilesUseCase(listRepos, listServices),
		GetPaymentProfileListPageData: NewGetPaymentProfileListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetPaymentProfileItemPageData: NewGetPaymentProfileItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}
