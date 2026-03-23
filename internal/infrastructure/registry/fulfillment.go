package registry

import (
	"github.com/erniealice/espyna-golang/internal/application/ports/integration"
)

// =============================================================================
// Fulfillment Factory Registry Instance
// =============================================================================
//
// The config type parameter is map[string]any because esqyma does not yet have
// a fulfillment integration proto package. When esqyma/pkg/schema/v1/integration/fulfillment
// is created, replace map[string]any with *fulfillmentpb.FulfillmentProviderConfig.

var fulfillmentRegistry = NewFactoryRegistry[integration.FulfillmentProvider, map[string]any]("fulfillment")

// =============================================================================
// Fulfillment Provider Functions
// =============================================================================

func RegisterFulfillmentProviderFactory(name string, factory func() integration.FulfillmentProvider) {
	fulfillmentRegistry.RegisterFactory(name, factory)
}

func GetFulfillmentProviderFactory(name string) (func() integration.FulfillmentProvider, bool) {
	return fulfillmentRegistry.GetFactory(name)
}

func ListAvailableFulfillmentProviderFactories() []string {
	return fulfillmentRegistry.ListFactories()
}

type FulfillmentConfigTransformer func(rawConfig map[string]any) (map[string]any, error)

func RegisterFulfillmentConfigTransformer(name string, transformer FulfillmentConfigTransformer) {
	fulfillmentRegistry.RegisterConfigTransformer(name, transformer)
}

func GetFulfillmentConfigTransformer(name string) (FulfillmentConfigTransformer, bool) {
	return fulfillmentRegistry.GetConfigTransformer(name)
}

func TransformFulfillmentConfig(name string, rawConfig map[string]any) (map[string]any, error) {
	return fulfillmentRegistry.TransformConfig(name, rawConfig)
}

func RegisterFulfillmentBuildFromEnv(name string, builder func() (integration.FulfillmentProvider, error)) {
	fulfillmentRegistry.RegisterBuildFromEnv(name, builder)
}

func GetFulfillmentBuildFromEnv(name string) (func() (integration.FulfillmentProvider, error), bool) {
	return fulfillmentRegistry.GetBuildFromEnv(name)
}

func BuildFulfillmentProviderFromEnv(name string) (integration.FulfillmentProvider, error) {
	return fulfillmentRegistry.BuildFromEnv(name)
}

func ListAvailableFulfillmentBuildFromEnv() []string {
	return fulfillmentRegistry.ListBuildFromEnv()
}

func RegisterFulfillmentProvider(name string, factory func() integration.FulfillmentProvider, transformer FulfillmentConfigTransformer) {
	RegisterFulfillmentProviderFactory(name, factory)
	if transformer != nil {
		RegisterFulfillmentConfigTransformer(name, transformer)
	}
}
