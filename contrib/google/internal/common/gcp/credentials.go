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
	// EnvPrefix is the fully-explicit {CONCERN}_{PROVIDER}_ environment-variable
	// prefix for this credential set (e.g. "AUTH_FIREBASE_" or "STORAGE_GCS_").
	// Every field below is read as EnvPrefix+FIELD — no shared/global names.
	EnvPrefix string

	// ProjectID is the GCP project ID
	ProjectID string

	// CredentialsPath is the path to the service-account JSON file
	// (from EnvPrefix+"CREDENTIALS_FILE", e.g. AUTH_FIREBASE_CREDENTIALS_FILE)
	CredentialsPath string

	// UseServiceAccountJSON determines if service account JSON should be
	// constructed from environment variables
	UseServiceAccountJSON bool

	// ServiceAccountKeyPath is an alternative path to service account JSON file
	ServiceAccountKeyPath string
}

// DefaultCredentialConfig creates a CredentialConfig from environment variables
// with the given fully-explicit prefix (e.g., "AUTH_FIREBASE_" or "STORAGE_GCS_").
func DefaultCredentialConfig(envPrefix string) *CredentialConfig {
	// ProjectID falls back to the service-account's own project_id
	// ({prefix}SA_PROJECT_ID) when {prefix}PROJECT_ID is not set — providing a
	// service-account JSON (which embeds its project) is sufficient; you don't
	// have to also repeat the project id in a separate var. Same-concern derive,
	// not a cross-concern fallback.
	projectID := os.Getenv(envPrefix + "PROJECT_ID")
	if projectID == "" {
		projectID = os.Getenv(envPrefix + "SA_PROJECT_ID")
	}
	return &CredentialConfig{
		EnvPrefix:             envPrefix,
		ProjectID:             projectID,
		CredentialsPath:       os.Getenv(envPrefix + "CREDENTIALS_FILE"),
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
		Type:                    os.Getenv(prefix + "SA_TYPE"),
		ProjectID:               os.Getenv(prefix + "SA_PROJECT_ID"),
		PrivateKeyID:            os.Getenv(prefix + "SA_PRIVATE_KEY_ID"),
		PrivateKey:              strings.ReplaceAll(os.Getenv(prefix+"SA_PRIVATE_KEY"), "\\n", "\n"),
		ClientEmail:             os.Getenv(prefix + "SA_CLIENT_EMAIL"),
		ClientID:                os.Getenv(prefix + "SA_CLIENT_ID"),
		AuthURI:                 os.Getenv(prefix + "SA_AUTH_URI"),
		TokenURI:                os.Getenv(prefix + "SA_TOKEN_URI"),
		AuthProviderX509CertURL: os.Getenv(prefix + "SA_AUTH_PROVIDER_CERT_URL"),
		ClientX509CertURL:       os.Getenv(prefix + "SA_CLIENT_CERT_URL"),
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

	// Scenario 3: Use service account key file.
	// Pass the scoped path DIRECTLY to the SDK; never write the process-global
	// GOOGLE_APPLICATION_CREDENTIALS (that would let one concern's credentials
	// clobber another's — the per-concern {CONCERN}_{PROVIDER}_ split forbids it).
	if config.ServiceAccountKeyPath != "" {
		return option.WithCredentialsFile(config.ServiceAccountKeyPath), nil
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
