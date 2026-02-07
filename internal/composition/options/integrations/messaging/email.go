package messaging

import (
	"fmt"
	"os"
	"strings"
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
// ENVIRONMENT CONFIGURATION LOADERS
// =============================================================================

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func createGmailConfigFromEnv() GmailConfig {
	return GmailConfig{
		DelegateEmail:         getEnv("GMAIL_DELEGATE_EMAIL", ""),
		FromEmail:             getEnv("GMAIL_FROM_EMAIL", ""),
		FromName:              getEnv("GMAIL_FROM_NAME", ""),
		ReplyToEmail:          getEnv("GMAIL_REPLY_TO_EMAIL", ""),
		ServiceAccountKeyPath: getEnv("GMAIL_SERVICE_ACCOUNT_KEY_PATH", ""),
		SecretManagerPath:     getEnv("GMAIL_SECRET_MANAGER_PATH", ""),
		UseSecretManager:      getEnv("GMAIL_USE_SECRET_MANAGER", "false") == "true",
		Timeout:               getEnv("GMAIL_TIMEOUT", "30s"),
	}
}

func createMicrosoftConfigFromEnv() MicrosoftConfig {
	return MicrosoftConfig{
		ClientID:     getEnv("MICROSOFT_CLIENT_ID", ""),
		ClientSecret: getEnv("MICROSOFT_CLIENT_SECRET", ""),
		TenantID:     getEnv("MICROSOFT_TENANT_ID", ""),
		FromEmail:    getEnv("MICROSOFT_FROM_EMAIL", ""),
		FromName:     getEnv("MICROSOFT_FROM_NAME", ""),
	}
}

// =============================================================================
// EMAIL PROVIDER OPTIONS
// =============================================================================

// WithEmailFromEnv dynamically selects email provider based on CONFIG_EMAIL_PROVIDER
func WithEmailFromEnv() ContainerOption {
	return func(c Container) error {
		emailProvider := strings.ToLower(getEnv("CONFIG_EMAIL_PROVIDER", "mock"))

		switch emailProvider {
		case "gmail":
			return WithGmail(createGmailConfigFromEnv())(c)
		case "microsoft":
			return WithMicrosoft(createMicrosoftConfigFromEnv())(c)
		case "mock", "":
			return WithMockEmail()(c)
		default:
			return fmt.Errorf("unsupported email provider: %s", emailProvider)
		}
	}
}

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
