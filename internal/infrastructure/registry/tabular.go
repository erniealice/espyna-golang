package registry

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports/integration"
	tabularpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/tabular"
)

// =============================================================================
// Tabular Factory Registry Instance
// =============================================================================

var tabularRegistry = NewFactoryRegistry[integration.TabularSourceProvider, *tabularpb.TabularProviderConfig]("tabular")

// =============================================================================
// Tabular Provider Functions
// =============================================================================

// TabularConfigTransformer transforms raw config to TabularProviderConfig
type TabularConfigTransformer func(rawConfig map[string]any) (*tabularpb.TabularProviderConfig, error)

// TabularBuildFromEnv creates a TabularSourceProvider from environment variables
type TabularBuildFromEnv func() (integration.TabularSourceProvider, error)

// RegisterTabularProviderFactory registers a factory function for creating tabular providers
func RegisterTabularProviderFactory(name string, factory func() integration.TabularSourceProvider) {
	tabularRegistry.RegisterFactory(name, factory)
}

// GetTabularProviderFactory returns the factory function for a given provider name
func GetTabularProviderFactory(name string) (func() integration.TabularSourceProvider, bool) {
	return tabularRegistry.GetFactory(name)
}

// ListAvailableTabularProviderFactories returns all registered provider factory names
func ListAvailableTabularProviderFactories() []string {
	return tabularRegistry.ListFactories()
}

// RegisterTabularConfigTransformer registers a config transformer for a provider
func RegisterTabularConfigTransformer(name string, transformer TabularConfigTransformer) {
	tabularRegistry.RegisterConfigTransformer(name, transformer)
}

// GetTabularConfigTransformer returns the config transformer for a provider
func GetTabularConfigTransformer(name string) (TabularConfigTransformer, bool) {
	return tabularRegistry.GetConfigTransformer(name)
}

// TransformTabularConfig transforms raw config using the registered transformer
func TransformTabularConfig(name string, rawConfig map[string]any) (*tabularpb.TabularProviderConfig, error) {
	return tabularRegistry.TransformConfig(name, rawConfig)
}

// RegisterTabularBuildFromEnv registers a build-from-env function for a provider
func RegisterTabularBuildFromEnv(name string, builder TabularBuildFromEnv) {
	tabularRegistry.RegisterBuildFromEnv(name, builder)
}

// GetTabularBuildFromEnv returns the build-from-env function for a provider
func GetTabularBuildFromEnv(name string) (TabularBuildFromEnv, bool) {
	return tabularRegistry.GetBuildFromEnv(name)
}

// BuildTabularProviderFromEnv creates and initializes a provider from environment variables
func BuildTabularProviderFromEnv(name string) (integration.TabularSourceProvider, error) {
	builder, exists := GetTabularBuildFromEnv(name)
	if !exists {
		return nil, fmt.Errorf("no build-from-env function registered for tabular provider: %s", name)
	}
	return builder()
}

// ListAvailableTabularBuildFromEnv returns all registered build-from-env function names
func ListAvailableTabularBuildFromEnv() []string {
	return tabularRegistry.ListBuildFromEnv()
}

// RegisterTabularProvider is a convenience function that registers both factory and transformer
func RegisterTabularProvider(name string, factory func() integration.TabularSourceProvider, transformer TabularConfigTransformer) {
	RegisterTabularProviderFactory(name, factory)
	if transformer != nil {
		RegisterTabularConfigTransformer(name, transformer)
	}
}
