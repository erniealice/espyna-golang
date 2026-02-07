package registry

import (
	"leapfor.xyz/espyna/internal/application/ports"
)

// =============================================================================
// ID Config Type
// =============================================================================

// IDProviderConfig is a simple config for ID providers (no protobuf needed)
type IDProviderConfig struct {
	Provider string `json:"provider"`
	Enabled  bool   `json:"enabled"`
}

// =============================================================================
// ID Factory Registry Instance
// =============================================================================

var idRegistry = NewFactoryRegistry[ports.IDService, *IDProviderConfig]("id")

// =============================================================================
// ID Provider Functions
// =============================================================================

func RegisterIDProviderFactory(name string, factory func() ports.IDService) {
	idRegistry.RegisterFactory(name, factory)
}

func GetIDProviderFactory(name string) (func() ports.IDService, bool) {
	return idRegistry.GetFactory(name)
}

func ListAvailableIDProviderFactories() []string {
	return idRegistry.ListFactories()
}

// IDConfigTransformer transforms raw config to IDProviderConfig
type IDConfigTransformer func(rawConfig map[string]any) (*IDProviderConfig, error)

func RegisterIDConfigTransformer(name string, transformer IDConfigTransformer) {
	idRegistry.RegisterConfigTransformer(name, transformer)
}

func GetIDConfigTransformer(name string) (IDConfigTransformer, bool) {
	return idRegistry.GetConfigTransformer(name)
}

func TransformIDConfig(name string, rawConfig map[string]any) (*IDProviderConfig, error) {
	return idRegistry.TransformConfig(name, rawConfig)
}

func RegisterIDBuildFromEnv(name string, builder func() (ports.IDService, error)) {
	idRegistry.RegisterBuildFromEnv(name, builder)
}

func GetIDBuildFromEnv(name string) (func() (ports.IDService, error), bool) {
	return idRegistry.GetBuildFromEnv(name)
}

func BuildIDProviderFromEnv(name string) (ports.IDService, error) {
	return idRegistry.BuildFromEnv(name)
}

func ListAvailableIDBuildFromEnv() []string {
	return idRegistry.ListBuildFromEnv()
}

// RegisterIDProvider registers both factory and config transformer.
func RegisterIDProvider(name string, factory func() ports.IDService, transformer IDConfigTransformer) {
	RegisterIDProviderFactory(name, factory)
	if transformer != nil {
		RegisterIDConfigTransformer(name, transformer)
	}
}
