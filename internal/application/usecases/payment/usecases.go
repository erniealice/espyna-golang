package payment

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Payment use cases
	paymentUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/payment/payment"
	paymentAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/payment/payment_attribute"
	paymentMethodUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/payment/payment_method"
	paymentProfileUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/payment/payment_profile"

	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment"
	paymentattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_attribute"
	paymentmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_method"
	paymentprofilepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_profile"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// PaymentRepositories contains all payment domain repositories
type PaymentRepositories struct {
	Payment          paymentpb.PaymentDomainServiceServer
	PaymentAttribute paymentattributepb.PaymentAttributeDomainServiceServer
	PaymentMethod    paymentmethodpb.PaymentMethodDomainServiceServer
	PaymentProfile   paymentprofilepb.PaymentProfileDomainServiceServer
	Attribute        attributepb.AttributeDomainServiceServer
	Client           clientpb.ClientDomainServiceServer
	Subscription     subscriptionpb.SubscriptionDomainServiceServer
}

// PaymentUseCases contains all payment-related use cases
type PaymentUseCases struct {
	Payment          *paymentUseCases.UseCases
	PaymentAttribute *paymentAttributeUseCases.UseCases
	PaymentMethod    *paymentMethodUseCases.UseCases
	PaymentProfile   *paymentProfileUseCases.UseCases
}

// NewUseCases creates all payment use cases with proper constructor injection
func NewUseCases(
	repos PaymentRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
) *PaymentUseCases {
	paymentRepositories := paymentUseCases.PaymentRepositories{
		Payment:      repos.Payment,
		Subscription: repos.Subscription,
	}
	paymentServices := paymentUseCases.PaymentServices{
		AuthorizationService: authSvc,
		TransactionService:   txSvc,
		TranslationService:   i18nSvc,
		IDService:            idService,
	}

	paymentMethodRepositories := paymentMethodUseCases.PaymentMethodRepositories{
		PaymentMethod: repos.PaymentMethod,
	}
	paymentMethodServices := paymentMethodUseCases.PaymentMethodServices{
		AuthorizationService: authSvc,
		TransactionService:   txSvc,
		TranslationService:   i18nSvc,
	}

	paymentAttributeRepositories := paymentAttributeUseCases.PaymentAttributeRepositories{
		PaymentAttribute: repos.PaymentAttribute,
		Payment:          repos.Payment,
		Attribute:        repos.Attribute,
	}
	paymentAttributeServices := paymentAttributeUseCases.PaymentAttributeServices{
		AuthorizationService: authSvc,
		TransactionService:   txSvc,
		TranslationService:   i18nSvc,
		IDService:            idService,
	}

	paymentProfileRepositories := paymentProfileUseCases.PaymentProfileRepositories{
		PaymentProfile: repos.PaymentProfile,
		Client:         repos.Client,
		PaymentMethod:  repos.PaymentMethod,
	}
	paymentProfileServices := paymentProfileUseCases.PaymentProfileServices{
		AuthorizationService: authSvc,
		TransactionService:   txSvc,
		TranslationService:   i18nSvc,
	}

	return &PaymentUseCases{
		Payment:          paymentUseCases.NewUseCases(paymentRepositories, paymentServices),
		PaymentAttribute: paymentAttributeUseCases.NewUseCases(paymentAttributeRepositories, paymentAttributeServices),
		PaymentMethod:    paymentMethodUseCases.NewUseCases(paymentMethodRepositories, paymentMethodServices),
		PaymentProfile:   paymentProfileUseCases.NewUseCases(paymentProfileRepositories, paymentProfileServices),
	}
}
