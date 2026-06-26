package messaging

import (
	"fmt"
)

// =============================================================================
// CONTAINER INTERFACE (imported from infrastructure for consistency)
// =============================================================================

// Container defines the interface for container configuration
type Container interface {
	GetConfig() interface{}
	SetConfig(interface{})
}

// ContainerOption defines a function that can configure a Container
type ContainerOption func(Container) error

// EmailConfigSetter defines methods for setting email configuration
type EmailConfigSetter interface {
	SetEmailConfig(config interface{})
}

// =============================================================================
// EMAIL CONFIGURATION TYPES
// =============================================================================

// EmailConfig holds configuration for email providers
type EmailConfig struct {
	Gmail     *GmailConfig     `json:"gmail,omitempty"`
	Microsoft *MicrosoftConfig `json:"microsoft,omitempty"`
	Mock      bool             `json:"mock,omitempty"`
}

// GmailConfig holds configuration for Gmail provider
type GmailConfig struct {
	DelegateEmail         string `json:"delegate_email"`
	FromEmail             string `json:"from_email"`
	FromName              string `json:"from_name"`
	ReplyToEmail          string `json:"reply_to_email"`
	ServiceAccountKeyPath string `json:"service_account_key_path"`
	SecretManagerPath     string `json:"secret_manager_path"`
	UseSecretManager      bool   `json:"use_secret_manager"`
	Timeout               string `json:"timeout,omitempty"`
}

// Validate validates the Gmail configuration
func (c GmailConfig) Validate() error {
	if c.DelegateEmail == "" {
		return fmt.Errorf("gmail delegate email is required")
	}
	if c.FromEmail == "" {
		return fmt.Errorf("gmail from email is required")
	}
	return nil
}

// MicrosoftConfig holds configuration for Microsoft Graph email provider
type MicrosoftConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	TenantID     string `json:"tenant_id"`
	FromEmail    string `json:"from_email"`
	FromName     string `json:"from_name"`
}

// Validate validates the Microsoft configuration
func (c MicrosoftConfig) Validate() error {
	if c.ClientID == "" {
		return fmt.Errorf("microsoft client ID is required")
	}
	if c.TenantID == "" {
		return fmt.Errorf("microsoft tenant ID is required")
	}
	return nil
}

// =============================================================================
// EMAIL PROVIDER OPTIONS
// =============================================================================
//
// NOTE: The legacy WithEmailFromEnv loader (which read un-prefixed GMAIL_*/
// MICROSOFT_* env vars) has been removed. It had no live caller — the live
// email path is providers/integration/email.go CreateEmailProvider ->
// registry.BuildEmailProviderFromEnv -> each adapter's buildFromEnv, which
// reads the CONCERN-prefixed EMAIL_GMAIL_*/EMAIL_MICROSOFT_* env vars in the
// contrib adapters. The EmailConfig/GmailConfig/MicrosoftConfig types below
// are retained because options/config.go still references messaging.EmailConfig.

// WithGmail configures Gmail as email provider
func WithGmail(config GmailConfig) ContainerOption {
	return func(c Container) error {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid gmail configuration: %w", err)
		}

		if setter, ok := c.(EmailConfigSetter); ok {
			setter.SetEmailConfig(EmailConfig{Gmail: &config})
		} else {
			return fmt.Errorf("container does not implement SetEmailConfig method")
		}

		fmt.Printf("📧 Configured Gmail: %s\n", config.FromEmail)
		return nil
	}
}

// WithMicrosoft configures Microsoft Graph as email provider
func WithMicrosoft(config MicrosoftConfig) ContainerOption {
	return func(c Container) error {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid microsoft configuration: %w", err)
		}

		if setter, ok := c.(EmailConfigSetter); ok {
			setter.SetEmailConfig(EmailConfig{Microsoft: &config})
		} else {
			return fmt.Errorf("container does not implement SetEmailConfig method")
		}

		fmt.Printf("📧 Configured Microsoft Graph: %s\n", config.FromEmail)
		return nil
	}
}

// WithMockEmail configures mock email for testing/development
func WithMockEmail() ContainerOption {
	return func(c Container) error {
		if setter, ok := c.(EmailConfigSetter); ok {
			setter.SetEmailConfig(EmailConfig{Mock: true})
		} else {
			return fmt.Errorf("container does not implement SetEmailConfig method")
		}

		fmt.Printf("🧪 Configured mock email\n")
		return nil
	}
}
