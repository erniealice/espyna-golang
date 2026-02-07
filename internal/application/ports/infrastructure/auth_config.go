package infrastructure

import (
	"fmt"

	authpb "leapfor.xyz/esqyma/golang/v1/infrastructure/auth"
)

// AuthConfigAdapter provides helpers to convert between map[string]any config
// and proto ProviderConfig for authentication providers
//
// This adapter serves as the bridge between the application configuration layer
// (which loads from environment variables/files as map[string]any) and the
// auth providers (which use strongly-typed proto contracts).
//
// Benefits of this approach:
// - Configuration remains flexible (can load from various sources)
// - Auth providers get type-safe proto configs
// - Clear separation between config loading and business logic
// - Easy to validate and transform configuration
type AuthConfigAdapter struct{}

// NewAuthConfigAdapter creates a new auth config adapter
func NewAuthConfigAdapter() *AuthConfigAdapter {
	return &AuthConfigAdapter{}
}

// ConvertMapToProtoConfig converts map[string]any config to proto ProviderConfig
// This allows ProviderManager to pass map configs while adapters use proto types
func (a *AuthConfigAdapter) ConvertMapToProtoConfig(
	providerName string,
	config map[string]any,
) (*authpb.ProviderConfig, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	protoConfig := &authpb.ProviderConfig{
		Enabled:     getBool(config, "enabled", true),
		DisplayName: getString(config, "display_name", providerName),
	}

	// Determine provider type and extract specific config
	switch providerName {
	case "firebase":
		// Firebase uses GCP provider
		protoConfig.Provider = authpb.Provider_PROVIDER_GCP
		protoConfig.Config = &authpb.ProviderConfig_GcpConfig{
			GcpConfig: &authpb.GcpProviderConfig{
				ProjectId:                getString(config, "project_id", ""),
				ClientId:                 getString(config, "client_id", ""),
				ServiceAccountEmail:      getString(config, "service_account_email", ""),
				WorkloadIdentityPool:     getString(config, "workload_identity_pool", ""),
				WorkloadIdentityProvider: getString(config, "workload_identity_provider", ""),
				TokenEndpoint:            getString(config, "token_endpoint", ""),
				AuthorizationEndpoint:    getString(config, "authorization_endpoint", ""),
				RedirectUris:             getStringSlice(config, "redirect_uris"),
				DefaultScopes:            getStringSlice(config, "default_scopes"),
				UseAdc:                   getBool(config, "use_adc", false),
			},
		}

	case "mock", "noop":
		// Mock/noop uses custom provider
		protoConfig.Provider = authpb.Provider_PROVIDER_CUSTOM
		protoConfig.Config = &authpb.ProviderConfig_CustomConfig{
			CustomConfig: &authpb.CustomProviderConfig{
				ProviderName:  providerName,
				BaseUrl:       getString(config, "base_url", ""),
				ClientId:      getString(config, "client_id", ""),
				RedirectUris:  getStringSlice(config, "redirect_uris"),
				DefaultScopes: getStringSlice(config, "default_scopes"),
				CustomParams:  getStringMap(config, "custom_params"),
			},
		}

	case "azure":
		protoConfig.Provider = authpb.Provider_PROVIDER_AZURE
		protoConfig.Config = &authpb.ProviderConfig_AzureConfig{
			AzureConfig: &authpb.AzureProviderConfig{
				TenantId:                getString(config, "tenant_id", ""),
				ClientId:                getString(config, "client_id", ""),
				Authority:               getString(config, "authority", ""),
				UseManagedIdentity:      getBool(config, "use_managed_identity", false),
				ManagedIdentityClientId: getString(config, "managed_identity_client_id", ""),
				TokenEndpoint:           getString(config, "token_endpoint", ""),
				AuthorizationEndpoint:   getString(config, "authorization_endpoint", ""),
				RedirectUris:            getStringSlice(config, "redirect_uris"),
				DefaultScopes:           getStringSlice(config, "default_scopes"),
			},
		}

	case "aws":
		protoConfig.Provider = authpb.Provider_PROVIDER_AWS
		protoConfig.Config = &authpb.ProviderConfig_AwsConfig{
			AwsConfig: &authpb.AwsProviderConfig{
				Region:            getString(config, "region", ""),
				UserPoolId:        getString(config, "user_pool_id", ""),
				AppClientId:       getString(config, "app_client_id", ""),
				IdentityPoolId:    getString(config, "identity_pool_id", ""),
				IamRoleArn:        getString(config, "iam_role_arn", ""),
				CognitoDomain:     getString(config, "cognito_domain", ""),
				RedirectUris:      getStringSlice(config, "redirect_uris"),
				DefaultScopes:     getStringSlice(config, "default_scopes"),
				UseIamCredentials: getBool(config, "use_iam_credentials", false),
			},
		}

	default:
		// Unknown provider, use custom
		protoConfig.Provider = authpb.Provider_PROVIDER_CUSTOM
		protoConfig.Config = &authpb.ProviderConfig_CustomConfig{
			CustomConfig: &authpb.CustomProviderConfig{
				ProviderName: providerName,
				CustomParams: getStringMap(config, "custom_params"),
			},
		}
	}

	return protoConfig, nil
}

// Helper functions getString, getBool, getStringSlice, getStringMap are defined in helpers.go
