//go:build google

package gcp

import (
	"os"
	"testing"
)

func TestDefaultCredentialConfig(t *testing.T) {
	// Set up test environment
	os.Setenv("GOOGLE_PROJECT_ID", "test-project")
	os.Setenv("GOOGLE_USE_SERVICE_ACCOUNT", "true")
	defer os.Unsetenv("GOOGLE_PROJECT_ID")
	defer os.Unsetenv("GOOGLE_USE_SERVICE_ACCOUNT")

	config := DefaultCredentialConfig("GOOGLE_")

	if config.ProjectID != "test-project" {
		t.Errorf("Expected project ID 'test-project', got '%s'", config.ProjectID)
	}

	if !config.UseServiceAccountJSON {
		t.Error("Expected UseServiceAccountJSON to be true")
	}

	if config.EnvPrefix != "GOOGLE_" {
		t.Errorf("Expected prefix 'GOOGLE_', got '%s'", config.EnvPrefix)
	}
}

func TestGetServiceAccountJSON(t *testing.T) {
	// Set up test service account environment
	testEnv := map[string]string{
		"TEST_TYPE":                   "service_account",
		"TEST_PROJECT_ID":             "test-project",
		"TEST_PRIVATE_KEY_ID":         "key123",
		"TEST_PRIVATE_KEY":            "-----BEGIN PRIVATE KEY-----\\ntest\\n-----END PRIVATE KEY-----",
		"TEST_CLIENT_EMAIL":           "test@test-project.iam.gserviceaccount.com",
		"TEST_CLIENT_ID":              "12345",
		"TEST_AUTH_URI":               "https://accounts.google.com/o/oauth2/auth",
		"TEST_TOKEN_URI":              "https://oauth2.googleapis.com/token",
		"TEST_AUTH_PROVIDER_CERT_URL": "https://www.googleapis.com/oauth2/v1/certs",
		"TEST_CLIENT_CERT_URL":        "https://www.googleapis.com/robot/v1/metadata/x509/test",
	}

	for key, value := range testEnv {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	config := &CredentialConfig{
		EnvPrefix: "TEST_",
		ProjectID: "test-project",
	}

	jsonBytes, err := GetServiceAccountJSON(config)
	if err != nil {
		t.Fatalf("GetServiceAccountJSON failed: %v", err)
	}

	if len(jsonBytes) == 0 {
		t.Error("Expected non-empty JSON bytes")
	}

	// Verify JSON structure is valid and contains expected fields
	jsonStr := string(jsonBytes)
	if !contains(jsonStr, "-----BEGIN PRIVATE KEY-----") {
		t.Error("Expected private key BEGIN marker in JSON")
	}
	if !contains(jsonStr, "test@test-project.iam.gserviceaccount.com") {
		t.Error("Expected client email in JSON")
	}
}

func TestGetServiceAccountJSON_MissingFields(t *testing.T) {
	config := &CredentialConfig{
		EnvPrefix: "MISSING_",
		ProjectID: "test-project",
	}

	_, err := GetServiceAccountJSON(config)
	if err == nil {
		t.Error("Expected error for missing service account fields")
	}
}

func TestCredentialConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  CredentialConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: CredentialConfig{
				EnvPrefix: "GOOGLE_",
				ProjectID: "test-project",
			},
			wantErr: false,
		},
		{
			name: "missing prefix",
			config: CredentialConfig{
				ProjectID: "test-project",
			},
			wantErr: true,
		},
		{
			name: "missing project ID",
			config: CredentialConfig{
				EnvPrefix: "GOOGLE_",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
