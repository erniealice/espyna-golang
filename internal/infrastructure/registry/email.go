package registry

import (
	"leapfor.xyz/espyna/internal/application/ports"
	emailpb "leapfor.xyz/esqyma/golang/v1/integration/email"
)

// =============================================================================
// Email Factory Registry Instance
// =============================================================================

var emailRegistry = NewFactoryRegistry[ports.EmailProvider, *emailpb.EmailProviderConfig]("email")

// =============================================================================
// Email Provider Functions
// =============================================================================

func RegisterEmailProviderFactory(name string, factory func() ports.EmailProvider) {
	emailRegistry.RegisterFactory(name, factory)
}

func GetEmailProviderFactory(name string) (func() ports.EmailProvider, bool) {
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

func RegisterEmailBuildFromEnv(name string, builder func() (ports.EmailProvider, error)) {
	emailRegistry.RegisterBuildFromEnv(name, builder)
}

func GetEmailBuildFromEnv(name string) (func() (ports.EmailProvider, error), bool) {
	return emailRegistry.GetBuildFromEnv(name)
}

func BuildEmailProviderFromEnv(name string) (ports.EmailProvider, error) {
	return emailRegistry.BuildFromEnv(name)
}

func ListAvailableEmailBuildFromEnv() []string {
	return emailRegistry.ListBuildFromEnv()
}

func RegisterEmailProvider(name string, factory func() ports.EmailProvider, transformer EmailConfigTransformer) {
	RegisterEmailProviderFactory(name, factory)
	if transformer != nil {
		RegisterEmailConfigTransformer(name, transformer)
	}
}
