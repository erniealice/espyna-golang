//go:build google || firebase

package gcp

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"google.golang.org/api/option"
)

// CredentialConfig holds GCP credential configuration
type CredentialConfig struct {
	// EnvPrefix is the environment variable prefix ("GOOGLE_" or "FIREBASE_")
	EnvPrefix string

	// ProjectID is the GCP project ID
	ProjectID string

	// CredentialsPath is the path to credentials file
	// (from GOOGLE_APPLICATION_CREDENTIALS)
	CredentialsPath string

	// UseServiceAccountJSON determines if service account JSON should be
	// constructed from environment variables
	UseServiceAccountJSON bool

	// ServiceAccountKeyPath is an alternative path to service account JSON file
	ServiceAccountKeyPath string
}

// DefaultCredentialConfig creates a CredentialConfig from environment variables
// with the given prefix (e.g., "GOOGLE_" or "FIREBASE_")
func DefaultCredentialConfig(envPrefix string) *CredentialConfig {
	return &CredentialConfig{
		EnvPrefix:             envPrefix,
		ProjectID:             os.Getenv(envPrefix + "CLOUD_PROJECT_ID"),
		CredentialsPath:       os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		UseServiceAccountJSON: os.Getenv(envPrefix+"USE_SERVICE_ACCOUNT") == "true",
		ServiceAccountKeyPath: os.Getenv(envPrefix + "SERVICE_ACCOUNT_KEY_PATH"),
	}
}

// ServiceAccountKey represents the structure of a GCP service account JSON key
type ServiceAccountKey struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

// GetServiceAccountJSON creates service account JSON from environment variables
// using the configured environment prefix
func GetServiceAccountJSON(config *CredentialConfig) ([]byte, error) {
	prefix := config.EnvPrefix

	// Construct service account key from environment variables
	key := ServiceAccountKey{
		Type:                    os.Getenv(prefix + "TYPE"),
		ProjectID:               os.Getenv(prefix + "PROJECT_ID"),
		PrivateKeyID:            os.Getenv(prefix + "PRIVATE_KEY_ID"),
		PrivateKey:              strings.ReplaceAll(os.Getenv(prefix+"PRIVATE_KEY"), "\\n", "\n"),
		ClientEmail:             os.Getenv(prefix + "CLIENT_EMAIL"),
		ClientID:                os.Getenv(prefix + "CLIENT_ID"),
		AuthURI:                 os.Getenv(prefix + "AUTH_URI"),
		TokenURI:                os.Getenv(prefix + "TOKEN_URI"),
		AuthProviderX509CertURL: os.Getenv(prefix + "AUTH_PROVIDER_CERT_URL"),
		ClientX509CertURL:       os.Getenv(prefix + "CLIENT_CERT_URL"),
	}

	// Validate required fields
	if key.Type == "" || key.ProjectID == "" || key.PrivateKey == "" || key.ClientEmail == "" {
		return nil, fmt.Errorf("missing required service account fields (type, project_id, private_key, or client_email)")
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(key)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal service account key: %w", err)
	}

	return jsonBytes, nil
}

// GetClientOption creates a google.golang.org/api/option.ClientOption from config
//
// This function handles three credential scenarios:
// 1. Service account from environment variables (UseServiceAccountJSON = true)
// 2. Credentials file path (CredentialsPath is set)
// 3. Service account key file (ServiceAccountKeyPath is set)
// 4. Default credentials (returns nil, uses Application Default Credentials)
func GetClientOption(config *CredentialConfig) (option.ClientOption, error) {
	// Scenario 1: Construct service account JSON from environment variables
	if config.UseServiceAccountJSON {
		serviceAccountJSON, err := GetServiceAccountJSON(config)
		if err != nil {
			return nil, fmt.Errorf("failed to get service account JSON: %w", err)
		}
		return option.WithCredentialsJSON(serviceAccountJSON), nil
	}

	// Scenario 2: Use credentials file path
	if config.CredentialsPath != "" {
		return option.WithCredentialsFile(config.CredentialsPath), nil
	}

	// Scenario 3: Use service account key file
	if config.ServiceAccountKeyPath != "" {
		// Set environment variable for Google libraries
		err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", config.ServiceAccountKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to set GOOGLE_APPLICATION_CREDENTIALS: %w", err)
		}
		return nil, nil // Will use default credentials (from env var)
	}

	// Scenario 4: Use Application Default Credentials
	// (e.g., from gcloud CLI, Cloud Run, GKE, etc.)
	return nil, nil
}

// Validate checks if the credential configuration is valid
func (c *CredentialConfig) Validate() error {
	if c.EnvPrefix == "" {
		return fmt.Errorf("environment prefix is required")
	}

	if c.ProjectID == "" {
		return fmt.Errorf("project ID is required")
	}

	return nil
}
