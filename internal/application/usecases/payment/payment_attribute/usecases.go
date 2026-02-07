package payment_attribute

import (
	"leapfor.xyz/espyna/internal/application/ports"
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
	paymentpb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment"
	paymentattributepb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_attribute"
)

// PaymentAttributeRepositories groups all repository dependencies for payment attribute use cases
type PaymentAttributeRepositories struct {
	PaymentAttribute paymentattributepb.PaymentAttributeDomainServiceServer // Primary entity repository
	Payment          paymentpb.PaymentDomainServiceServer
	Attribute        attributepb.AttributeDomainServiceServer
}

// PaymentAttributeServices groups all business service dependencies for payment attribute use cases
type PaymentAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all payment attribute-related use cases
type UseCases struct {
	CreatePaymentAttribute          *CreatePaymentAttributeUseCase
	CreatePaymentAttributesByCode   *CreatePaymentAttributesByCodeUseCase
	ReadPaymentAttribute            *ReadPaymentAttributeUseCase
	UpdatePaymentAttribute          *UpdatePaymentAttributeUseCase
	DeletePaymentAttribute          *DeletePaymentAttributeUseCase
	ListPaymentAttributes           *ListPaymentAttributesUseCase
	GetPaymentAttributeListPageData *GetPaymentAttributeListPageDataUseCase
	GetPaymentAttributeItemPageData *GetPaymentAttributeItemPageDataUseCase
}

// NewUseCases creates a new collection of payment attribute use cases
func NewUseCases(
	repositories PaymentAttributeRepositories,
	services PaymentAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreatePaymentAttributeRepositories{
		PaymentAttribute: repositories.PaymentAttribute,
		Payment:          repositories.Payment,
		Attribute:        repositories.Attribute,
	}
	createServices := CreatePaymentAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadPaymentAttributeRepositories{
		PaymentAttribute: repositories.PaymentAttribute,
	}
	readServices := ReadPaymentAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdatePaymentAttributeRepositories{
		PaymentAttribute: repositories.PaymentAttribute,
		Payment:          repositories.Payment,
		Attribute:        repositories.Attribute,
	}
	updateServices := UpdatePaymentAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeletePaymentAttributeRepositories{
		PaymentAttribute: repositories.PaymentAttribute,
	}
	deleteServices := DeletePaymentAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListPaymentAttributesRepositories{
		PaymentAttribute: repositories.PaymentAttribute,
	}
	listServices := ListPaymentAttributesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetPaymentAttributeListPageDataRepositories{
		PaymentAttribute: repositories.PaymentAttribute,
	}
	listPageDataServices := GetPaymentAttributeListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	itemPageDataRepos := GetPaymentAttributeItemPageDataRepositories{
		PaymentAttribute: repositories.PaymentAttribute,
	}
	itemPageDataServices := GetPaymentAttributeItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	createUseCase := NewCreatePaymentAttributeUseCase(createRepos, createServices)

	// Build repos for code-based creation (uses Attribute repo for code-to-ID resolution)
	createByCodeRepos := CreatePaymentAttributesByCodeRepositories{
		PaymentAttribute: repositories.PaymentAttribute,
		Attribute:        repositories.Attribute,
	}

	return &UseCases{
		CreatePaymentAttribute:          createUseCase,
		CreatePaymentAttributesByCode:   NewCreatePaymentAttributesByCodeUseCase(createByCodeRepos, createUseCase),
		ReadPaymentAttribute:            NewReadPaymentAttributeUseCase(readRepos, readServices),
		UpdatePaymentAttribute:          NewUpdatePaymentAttributeUseCase(updateRepos, updateServices),
		DeletePaymentAttribute:          NewDeletePaymentAttributeUseCase(deleteRepos, deleteServices),
		ListPaymentAttributes:           NewListPaymentAttributesUseCase(listRepos, listServices),
		GetPaymentAttributeListPageData: NewGetPaymentAttributeListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetPaymentAttributeItemPageData: NewGetPaymentAttributeItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}
