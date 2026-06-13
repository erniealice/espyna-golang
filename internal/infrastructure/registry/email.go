package registry

import (
	"github.com/erniealice/espyna-golang/internal/application/ports/integration"
	emailpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/email"
)

// =============================================================================
// Email Factory Registry Instance
// =============================================================================

var emailRegistry = NewFactoryRegistry[integration.EmailProvider, *emailpb.EmailProviderConfig]("email")

// =============================================================================
// Email Provider Functions
// =============================================================================

func RegisterEmailProviderFactory(name string, factory func() integration.EmailProvider) {
	emailRegistry.RegisterFactory(name, factory)
}

func GetEmailProviderFactory(name string) (func() integration.EmailProvider, bool) {
	return emailRegistry.GetFactory(name)
}

func ListAvailableEmailProviderFactories() []string {
	return emailRegistry.ListFactories()
}

type EmailConfigTransformer func(rawConfig map[string]any) (*emailpb.EmailProviderConfig, error)

func RegisterEmailConfigTransformer(name string, transformer EmailConfigTransformer) {
	emailRegistry.RegisterConfigTransformer(name, transformer)
}

func GetEmailConfigTransformer(name string) (EmailConfigTransformer, bool) {
	return emailRegistry.GetConfigTransformer(name)
}

func TransformEmailConfig(name string, rawConfig map[string]any) (*emailpb.EmailProviderConfig, error) {
	return emailRegistry.TransformConfig(name, rawConfig)
}

func RegisterEmailBuildFromEnv(name string, builder func() (integration.EmailProvider, error)) {
	emailRegistry.RegisterBuildFromEnv(name, builder)
}

func GetEmailBuildFromEnv(name string) (func() (integration.EmailProvider, error), bool) {
	return emailRegistry.GetBuildFromEnv(name)
}

func BuildEmailProviderFromEnv(name string) (integration.EmailProvider, error) {
	return emailRegistry.BuildFromEnv(name)
}

func ListAvailableEmailBuildFromEnv() []string {
	return emailRegistry.ListBuildFromEnv()
}

func RegisterEmailProvider(name string, factory func() integration.EmailProvider, transformer EmailConfigTransformer) {
	RegisterEmailProviderFactory(name, factory)
	if transformer != nil {
		RegisterEmailConfigTransformer(name, transformer)
	}
}
