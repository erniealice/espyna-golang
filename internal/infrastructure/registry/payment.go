package registry

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/payment"
)

// =============================================================================
// Payment Factory Registry Instance
// =============================================================================

var paymentRegistry = NewFactoryRegistry[ports.PaymentProvider, *paymentpb.PaymentProviderConfig]("payment")

// =============================================================================
// Payment Provider Functions
// =============================================================================

func RegisterPaymentProviderFactory(name string, factory func() ports.PaymentProvider) {
	paymentRegistry.RegisterFactory(name, factory)
}

func GetPaymentProviderFactory(name string) (func() ports.PaymentProvider, bool) {
	return paymentRegistry.GetFactory(name)
}

func ListAvailablePaymentProviderFactories() []string {
	return paymentRegistry.ListFactories()
}

type PaymentConfigTransformer func(rawConfig map[string]any) (*paymentpb.PaymentProviderConfig, error)

func RegisterPaymentConfigTransformer(name string, transformer PaymentConfigTransformer) {
	paymentRegistry.RegisterConfigTransformer(name, transformer)
}

func GetPaymentConfigTransformer(name string) (PaymentConfigTransformer, bool) {
	return paymentRegistry.GetConfigTransformer(name)
}

func TransformPaymentConfig(name string, rawConfig map[string]any) (*paymentpb.PaymentProviderConfig, error) {
	return paymentRegistry.TransformConfig(name, rawConfig)
}

func RegisterPaymentBuildFromEnv(name string, builder func() (ports.PaymentProvider, error)) {
	paymentRegistry.RegisterBuildFromEnv(name, builder)
}

func GetPaymentBuildFromEnv(name string) (func() (ports.PaymentProvider, error), bool) {
	return paymentRegistry.GetBuildFromEnv(name)
}

func BuildPaymentProviderFromEnv(name string) (ports.PaymentProvider, error) {
	return paymentRegistry.BuildFromEnv(name)
}

func ListAvailablePaymentBuildFromEnv() []string {
	return paymentRegistry.ListBuildFromEnv()
}

func RegisterPaymentProvider(name string, factory func() ports.PaymentProvider, transformer PaymentConfigTransformer) {
	RegisterPaymentProviderFactory(name, factory)
	if transformer != nil {
		RegisterPaymentConfigTransformer(name, transformer)
	}
}
