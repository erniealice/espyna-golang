package registry

import (
	"leapfor.xyz/espyna/internal/application/ports"
)

// =============================================================================
// Translation Config Type
// =============================================================================

// TranslationProviderConfig is a simple config for translation providers (no protobuf needed)
type TranslationProviderConfig struct {
	Provider         string `json:"provider"`
	Enabled          bool   `json:"enabled"`
	TranslationsPath string `json:"translations_path"`
}

// =============================================================================
// Translation Factory Registry Instance
// =============================================================================

var translationRegistry = NewFactoryRegistry[ports.TranslationService, *TranslationProviderConfig]("translation")

// =============================================================================
// Translation Provider Functions
// =============================================================================

func RegisterTranslationProviderFactory(name string, factory func() ports.TranslationService) {
	translationRegistry.RegisterFactory(name, factory)
}

func GetTranslationProviderFactory(name string) (func() ports.TranslationService, bool) {
	return translationRegistry.GetFactory(name)
}

func ListAvailableTranslationProviderFactories() []string {
	return translationRegistry.ListFactories()
}

// TranslationConfigTransformer transforms raw config to TranslationProviderConfig
type TranslationConfigTransformer func(rawConfig map[string]any) (*TranslationProviderConfig, error)

func RegisterTranslationConfigTransformer(name string, transformer TranslationConfigTransformer) {
	translationRegistry.RegisterConfigTransformer(name, transformer)
}

func GetTranslationConfigTransformer(name string) (TranslationConfigTransformer, bool) {
	return translationRegistry.GetConfigTransformer(name)
}

func TransformTranslationConfig(name string, rawConfig map[string]any) (*TranslationProviderConfig, error) {
	return translationRegistry.TransformConfig(name, rawConfig)
}

func RegisterTranslationBuildFromEnv(name string, builder func() (ports.TranslationService, error)) {
	translationRegistry.RegisterBuildFromEnv(name, builder)
}

func GetTranslationBuildFromEnv(name string) (func() (ports.TranslationService, error), bool) {
	return translationRegistry.GetBuildFromEnv(name)
}

func BuildTranslationProviderFromEnv(name string) (ports.TranslationService, error) {
	return translationRegistry.BuildFromEnv(name)
}

func ListAvailableTranslationBuildFromEnv() []string {
	return translationRegistry.ListBuildFromEnv()
}

// RegisterTranslationProvider registers both factory and config transformer.
func RegisterTranslationProvider(name string, factory func() ports.TranslationService, transformer TranslationConfigTransformer) {
	RegisterTranslationProviderFactory(name, factory)
	if transformer != nil {
		RegisterTranslationConfigTransformer(name, transformer)
	}
}
