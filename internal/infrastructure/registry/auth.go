package registry

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/auth"
)

// =============================================================================
// Auth Factory Registry Instance
// =============================================================================

var authRegistry = NewFactoryRegistry[ports.AuthProvider, *authpb.ProviderConfig]("auth")

// =============================================================================
// Auth Provider Functions
// =============================================================================

func RegisterAuthProviderFactory(name string, factory func() ports.AuthProvider) {
	authRegistry.RegisterFactory(name, factory)
}

func GetAuthProviderFactory(name string) (func() ports.AuthProvider, bool) {
	return authRegistry.GetFactory(name)
}

func ListAvailableAuthProviderFactories() []string {
	return authRegistry.ListFactories()
}

type AuthConfigTransformer func(rawConfig map[string]any) (*authpb.ProviderConfig, error)

func RegisterAuthConfigTransformer(name string, transformer AuthConfigTransformer) {
	authRegistry.RegisterConfigTransformer(name, transformer)
}

func GetAuthConfigTransformer(name string) (AuthConfigTransformer, bool) {
	return authRegistry.GetConfigTransformer(name)
}

func TransformAuthConfig(name string, rawConfig map[string]any) (*authpb.ProviderConfig, error) {
	return authRegistry.TransformConfig(name, rawConfig)
}

func RegisterAuthBuildFromEnv(name string, builder func() (ports.AuthProvider, error)) {
	authRegistry.RegisterBuildFromEnv(name, builder)
}

func GetAuthBuildFromEnv(name string) (func() (ports.AuthProvider, error), bool) {
	return authRegistry.GetBuildFromEnv(name)
}

func BuildAuthProviderFromEnv(name string) (ports.AuthProvider, error) {
	return authRegistry.BuildFromEnv(name)
}

func ListAvailableAuthBuildFromEnv() []string {
	return authRegistry.ListBuildFromEnv()
}

func RegisterAuthProvider(name string, factory func() ports.AuthProvider, transformer AuthConfigTransformer) {
	RegisterAuthProviderFactory(name, factory)
	if transformer != nil {
		RegisterAuthConfigTransformer(name, transformer)
	}
}
