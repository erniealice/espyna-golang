package domain

import (
	"fmt"

	"leapfor.xyz/espyna/internal/composition/contracts"
	"leapfor.xyz/espyna/internal/infrastructure/registry"

	// Protobuf domain services - Common domain
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"

	// Protobuf domain services - Entity domain
	clientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/client"

	// Protobuf domain services - Payment domain
	paymentpb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment"
	paymentattributepb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_attribute"
	paymentmethodpb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_method"
	paymentprofilepb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_profile"

	// Protobuf domain services - Subscription domain
	subscriptionpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/subscription"
)

// PaymentRepositories contains all 4 payment domain repositories and cross-domain dependencies
// Payment domain: Payment, PaymentAttribute, PaymentMethod, PaymentProfile (4 entities)
// Cross-domain: Attribute, Client, Subscription (needed by Payment use cases)
type PaymentRepositories struct {
	Payment          paymentpb.PaymentDomainServiceServer
	PaymentAttribute paymentattributepb.PaymentAttributeDomainServiceServer
	PaymentMethod    paymentmethodpb.PaymentMethodDomainServiceServer
	PaymentProfile   paymentprofilepb.PaymentProfileDomainServiceServer
	Attribute        attributepb.AttributeDomainServiceServer       // Cross-domain dependency
	Client           clientpb.ClientDomainServiceServer             // Cross-domain dependency
	Subscription     subscriptionpb.SubscriptionDomainServiceServer // Cross-domain dependency
}

// NewPaymentRepositories creates and returns a new set of PaymentRepositories
func NewPaymentRepositories(dbProvider contracts.Provider, dbTableConfig *registry.DatabaseTableConfig) (*PaymentRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	// Create each repository individually using configured table names directly from dbTableConfig
	paymentRepo, err := repoCreator.CreateRepository("payment", conn, dbTableConfig.Payment)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment repository: %w", err)
	}

	paymentAttributeRepo, err := repoCreator.CreateRepository("payment_attribute", conn, dbTableConfig.PaymentAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment_attribute repository: %w", err)
	}

	paymentMethodRepo, err := repoCreator.CreateRepository("payment_method", conn, dbTableConfig.PaymentMethod)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment_method repository: %w", err)
	}

	paymentProfileRepo, err := repoCreator.CreateRepository("payment_profile", conn, dbTableConfig.PaymentProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment_profile repository: %w", err)
	}

	attributeRepo, err := repoCreator.CreateRepository("attribute", conn, dbTableConfig.Attribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create attribute repository: %w", err)
	}

	clientRepo, err := repoCreator.CreateRepository("client", conn, dbTableConfig.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to create client repository: %w", err)
	}

	subscriptionRepo, err := repoCreator.CreateRepository("subscription", conn, dbTableConfig.Subscription)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription repository: %w", err)
	}

	return &PaymentRepositories{
		Payment:          paymentRepo.(paymentpb.PaymentDomainServiceServer),
		PaymentAttribute: paymentAttributeRepo.(paymentattributepb.PaymentAttributeDomainServiceServer),
		PaymentMethod:    paymentMethodRepo.(paymentmethodpb.PaymentMethodDomainServiceServer),
		PaymentProfile:   paymentProfileRepo.(paymentprofilepb.PaymentProfileDomainServiceServer),
		Attribute:        attributeRepo.(attributepb.AttributeDomainServiceServer),
		Client:           clientRepo.(clientpb.ClientDomainServiceServer),
		Subscription:     subscriptionRepo.(subscriptionpb.SubscriptionDomainServiceServer),
	}, nil
}
