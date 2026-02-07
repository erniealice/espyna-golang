//go:build google && gmail

package google

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// GmailEnvPrefix is the environment variable prefix for Gmail configuration
const GmailEnvPrefix = "LEAPFOR_INTEGRATION_EMAIL_GMAIL_"

// GmailClientManager manages Gmail API client with service account delegation
type GmailClientManager struct {
	service       *gmail.Service
	config        *GmailConfig
	delegateEmail string
}

// GmailConfig holds Gmail-specific configuration
type GmailConfig struct {
	// ProjectID is the GCP project ID
	ProjectID string

	// DelegateEmail is the email to impersonate (domain-wide delegation)
	DelegateEmail string

	// FromEmail is the default sender email address
	FromEmail string

	// FromName is the default sender display name
	FromName string

	// ReplyToEmail is the default reply-to address
	ReplyToEmail string

	// ServiceAccountKeyPath is the path to the service account JSON file
	ServiceAccountKeyPath string

	// SecretManagerPath is the Secret Manager resource path (for production)
	// Format: projects/PROJECT_ID/secrets/SECRET_NAME/versions/VERSION
	SecretManagerPath string

	// UseSecretManager determines if credentials should be fetched from Secret Manager
	UseSecretManager bool

	// Timeout for API requests
	Timeout time.Duration
}

// DefaultGmailConfig creates GmailConfig from environment variables
// Uses prefix: LEAPFOR_INTEGRATION_EMAIL_GMAIL_
func DefaultGmailConfig() *GmailConfig {
	timeout := 30 * time.Second
	if timeoutStr := os.Getenv(GmailEnvPrefix + "TIMEOUT"); timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		}
	}

	return &GmailConfig{
		ProjectID:             os.Getenv(GmailEnvPrefix + "PROJECT_ID"),
		DelegateEmail:         os.Getenv(GmailEnvPrefix + "DELEGATE_EMAIL"),
		FromEmail:             os.Getenv(GmailEnvPrefix + "FROM_EMAIL"),
		FromName:              os.Getenv(GmailEnvPrefix + "FROM_NAME"),
		ReplyToEmail:          os.Getenv(GmailEnvPrefix + "REPLY_TO_EMAIL"),
		ServiceAccountKeyPath: os.Getenv(GmailEnvPrefix + "SERVICE_ACCOUNT_KEY_PATH"),
		SecretManagerPath:     os.Getenv(GmailEnvPrefix + "SECRET_MANAGER_PATH"),
		UseSecretManager:      os.Getenv(GmailEnvPrefix+"USE_SECRET_MANAGER") == "true",
		Timeout:               timeout,
	}
}

// Validate checks if the Gmail configuration is valid
func (c *GmailConfig) Validate() error {
	if c.DelegateEmail == "" {
		return fmt.Errorf("delegate email is required (%sDELEGATE_EMAIL)", GmailEnvPrefix)
	}

	// Must have either service account key path or secret manager path
	if !c.UseSecretManager && c.ServiceAccountKeyPath == "" {
		return fmt.Errorf("service account key path is required when not using Secret Manager (%sSERVICE_ACCOUNT_KEY_PATH)", GmailEnvPrefix)
	}

	if c.UseSecretManager && c.SecretManagerPath == "" {
		return fmt.Errorf("secret manager path is required when using Secret Manager (%sSECRET_MANAGER_PATH)", GmailEnvPrefix)
	}

	return nil
}

// NewGmailClientManager creates a new Gmail client manager with service account delegation
func NewGmailClientManager(ctx context.Context, config *GmailConfig) (*GmailClientManager, error) {
	if config == nil {
		config = DefaultGmailConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Gmail config: %w", err)
	}

	// Get service account credentials
	serviceAccountKey, err := getGmailServiceAccountKey(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to get service account key: %w", err)
	}

	// Create JWT config for domain-wide delegation
	jwtConfig, err := google.JWTConfigFromJSON(serviceAccountKey, gmail.GmailSendScope)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT config: %w", err)
	}

	// Set the subject (email to impersonate)
	jwtConfig.Subject = config.DelegateEmail

	log.Printf("📧 Gmail: Using delegated email: %s", config.DelegateEmail)

	// Create Gmail service with impersonation
	gmailService, err := gmail.NewService(ctx, option.WithTokenSource(jwtConfig.TokenSource(ctx)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	log.Println("✅ Gmail API client initialized successfully")

	return &GmailClientManager{
		service:       gmailService,
		config:        config,
		delegateEmail: config.DelegateEmail,
	}, nil
}

// getGmailServiceAccountKey retrieves the service account key from file or Secret Manager
func getGmailServiceAccountKey(ctx context.Context, config *GmailConfig) ([]byte, error) {
	if config.UseSecretManager {
		return getGmailSecretFromSecretManager(ctx, config.SecretManagerPath)
	}

	// Read from file
	serviceAccountKey, err := os.ReadFile(config.ServiceAccountKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read service account key file: %w", err)
	}

	// Validate the key structure
	var keyData map[string]interface{}
	if err := json.Unmarshal(serviceAccountKey, &keyData); err != nil {
		return nil, fmt.Errorf("invalid service account JSON: %w", err)
	}

	if clientEmail, ok := keyData["client_email"].(string); ok {
		log.Printf("📧 Gmail: Service account email: %s", clientEmail)
	}

	return serviceAccountKey, nil
}

// getGmailSecretFromSecretManager retrieves the service account key from Secret Manager
func getGmailSecretFromSecretManager(ctx context.Context, secretPath string) ([]byte, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create secretmanager client: %w", err)
	}
	defer client.Close()

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretPath,
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to access secret version: %w", err)
	}

	return result.Payload.Data, nil
}

// GetService returns the Gmail API service
func (m *GmailClientManager) GetService() *gmail.Service {
	return m.service
}

// GetConfig returns the Gmail configuration
func (m *GmailClientManager) GetConfig() *GmailConfig {
	return m.config
}

// GetDelegateEmail returns the delegated email address
func (m *GmailClientManager) GetDelegateEmail() string {
	return m.delegateEmail
}

// GetFromEmail returns the configured from email, or delegate email as fallback
func (m *GmailClientManager) GetFromEmail() string {
	if m.config.FromEmail != "" {
		return m.config.FromEmail
	}
	return m.delegateEmail
}

// GetFromName returns the configured from name
func (m *GmailClientManager) GetFromName() string {
	return m.config.FromName
}

// GetReplyToEmail returns the configured reply-to email
func (m *GmailClientManager) GetReplyToEmail() string {
	return m.config.ReplyToEmail
}

// ContainsGmailScope checks if scopes contain Gmail
func ContainsGmailScope(scopes []string) bool {
	for _, scope := range scopes {
		if strings.Contains(scope, "gmail") {
			return true
		}
	}
	return false
}

// Close cleans up Gmail client resources
func (m *GmailClientManager) Close() error {
	// Gmail service doesn't need explicit cleanup
	return nil
}
