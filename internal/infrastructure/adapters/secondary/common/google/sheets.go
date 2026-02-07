//go:build google && googlesheets

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
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// SheetsEnvPrefix is the environment variable prefix for Google Sheets configuration
const SheetsEnvPrefix = "LEAPFOR_INTEGRATION_DATASHEET_GOOGLESHEETS_"

// SheetsClientManager manages Google Sheets API client with service account delegation
type SheetsClientManager struct {
	service       *sheets.Service
	config        *SheetsConfig
	delegateEmail string
}

// SheetsConfig holds Google Sheets-specific configuration
type SheetsConfig struct {
	// ProjectID is the GCP project ID
	ProjectID string

	// DelegateEmail is the email to impersonate (domain-wide delegation)
	DelegateEmail string

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

// DefaultSheetsConfig creates SheetsConfig from environment variables
// Uses prefix: LEAPFOR_INTEGRATION_DATASHEET_GOOGLESHEETS_
func DefaultSheetsConfig() *SheetsConfig {
	timeout := 30 * time.Second
	if timeoutStr := os.Getenv(SheetsEnvPrefix + "TIMEOUT"); timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		}
	}

	return &SheetsConfig{
		ProjectID:             os.Getenv(SheetsEnvPrefix + "PROJECT_ID"),
		DelegateEmail:         os.Getenv(SheetsEnvPrefix + "DELEGATE_EMAIL"),
		ServiceAccountKeyPath: os.Getenv(SheetsEnvPrefix + "SERVICE_ACCOUNT_KEY_PATH"),
		SecretManagerPath:     os.Getenv(SheetsEnvPrefix + "SECRET_MANAGER_PATH"),
		UseSecretManager:      os.Getenv(SheetsEnvPrefix+"USE_SECRET_MANAGER") == "true",
		Timeout:               timeout,
	}
}

// Validate checks if the Sheets configuration is valid
func (c *SheetsConfig) Validate() error {
	if c.DelegateEmail == "" {
		return fmt.Errorf("delegate email is required (%sDELEGATE_EMAIL)", SheetsEnvPrefix)
	}

	// Must have either service account key path or secret manager path
	if !c.UseSecretManager && c.ServiceAccountKeyPath == "" {
		return fmt.Errorf("service account key path is required when not using Secret Manager (%sSERVICE_ACCOUNT_KEY_PATH)", SheetsEnvPrefix)
	}

	if c.UseSecretManager && c.SecretManagerPath == "" {
		return fmt.Errorf("secret manager path is required when using Secret Manager (%sSECRET_MANAGER_PATH)", SheetsEnvPrefix)
	}

	return nil
}

// NewSheetsClientManager creates a new Google Sheets client manager with service account delegation
func NewSheetsClientManager(ctx context.Context, config *SheetsConfig) (*SheetsClientManager, error) {
	if config == nil {
		config = DefaultSheetsConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Sheets config: %w", err)
	}

	// Get service account credentials
	serviceAccountKey, err := getSheetsServiceAccountKey(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to get service account key: %w", err)
	}

	// Create JWT config for domain-wide delegation with Sheets scopes
	jwtConfig, err := google.JWTConfigFromJSON(serviceAccountKey,
		sheets.SpreadsheetsScope,
		sheets.SpreadsheetsReadonlyScope,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT config: %w", err)
	}

	// Set the subject (email to impersonate)
	jwtConfig.Subject = config.DelegateEmail

	log.Printf("Google Sheets: Using delegated email: %s", config.DelegateEmail)

	// Create Sheets service with impersonation
	sheetsService, err := sheets.NewService(ctx, option.WithTokenSource(jwtConfig.TokenSource(ctx)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Sheets service: %w", err)
	}

	log.Println("Google Sheets API client initialized successfully")

	return &SheetsClientManager{
		service:       sheetsService,
		config:        config,
		delegateEmail: config.DelegateEmail,
	}, nil
}

// getSheetsServiceAccountKey retrieves the service account key from file or Secret Manager
func getSheetsServiceAccountKey(ctx context.Context, config *SheetsConfig) ([]byte, error) {
	if config.UseSecretManager {
		return getSheetsSecretFromSecretManager(ctx, config.SecretManagerPath)
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
		log.Printf("Google Sheets: Service account email: %s", clientEmail)
	}

	return serviceAccountKey, nil
}

// getSheetsSecretFromSecretManager retrieves the service account key from Secret Manager
func getSheetsSecretFromSecretManager(ctx context.Context, secretPath string) ([]byte, error) {
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

// GetService returns the Google Sheets API service
func (m *SheetsClientManager) GetService() *sheets.Service {
	return m.service
}

// GetConfig returns the Sheets configuration
func (m *SheetsClientManager) GetConfig() *SheetsConfig {
	return m.config
}

// GetDelegateEmail returns the delegated email address
func (m *SheetsClientManager) GetDelegateEmail() string {
	return m.delegateEmail
}

// ContainsSheetsScope checks if scopes contain Sheets
func ContainsSheetsScope(scopes []string) bool {
	for _, scope := range scopes {
		if strings.Contains(scope, "spreadsheets") {
			return true
		}
	}
	return false
}

// Close cleans up Sheets client resources
func (m *SheetsClientManager) Close() error {
	// Sheets service doesn't need explicit cleanup
	return nil
}
