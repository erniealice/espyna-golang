package gcp

import (
	"strings"
	"testing"
)

func TestLoadCredentialConfigRequiresExplicitProjectID(t *testing.T) {
	_, err := LoadCredentialConfig("TEST_EXPLICIT_", "")
	if err == nil || !strings.Contains(err.Error(), "TEST_EXPLICIT_PROJECT_ID is required") {
		t.Fatalf("LoadCredentialConfig() error = %v, want explicit project error", err)
	}
}

func TestLoadCredentialConfigUsesADCByDefault(t *testing.T) {
	config, err := LoadCredentialConfig("TEST_ADC_", "resource-project")
	if err != nil {
		t.Fatalf("LoadCredentialConfig() error = %v", err)
	}
	if config.ProjectID != "resource-project" {
		t.Fatalf("ProjectID = %q, want resource-project", config.ProjectID)
	}
	if config.CredentialsPath != "" {
		t.Fatalf("CredentialsPath = %q, want empty ADC path", config.CredentialsPath)
	}
	opt, err := GetClientOption(config)
	if err != nil {
		t.Fatalf("GetClientOption() error = %v", err)
	}
	if opt != nil {
		t.Fatal("GetClientOption() returned an option, want nil for ADC")
	}
}

func TestLoadCredentialConfigUsesScopedCredentialsFile(t *testing.T) {
	t.Setenv("TEST_FILE_CREDENTIALS_FILE", "/tmp/test-google-credentials.json")
	config, err := LoadCredentialConfig("TEST_FILE_", "resource-project")
	if err != nil {
		t.Fatalf("LoadCredentialConfig() error = %v", err)
	}
	if config.CredentialsPath != "/tmp/test-google-credentials.json" {
		t.Fatalf("CredentialsPath = %q", config.CredentialsPath)
	}
	opt, err := GetClientOption(config)
	if err != nil {
		t.Fatalf("GetClientOption() error = %v", err)
	}
	if opt == nil {
		t.Fatal("GetClientOption() = nil, want scoped file option")
	}
}

func TestLoadCredentialConfigRejectsRetiredVariables(t *testing.T) {
	tests := []struct {
		name string
		env  string
	}{
		{name: "boolean even when false", env: "TEST_RETIRED_USE_SERVICE_ACCOUNT"},
		{name: "key path alias", env: "TEST_RETIRED_SERVICE_ACCOUNT_KEY_PATH"},
		{name: "inline project", env: "TEST_RETIRED_SA_PROJECT_ID"},
		{name: "inline private key", env: "TEST_RETIRED_SA_PRIVATE_KEY"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secretValue := "must-not-appear-in-error"
			t.Setenv(tt.env, secretValue)
			_, err := LoadCredentialConfig("TEST_RETIRED_", "resource-project")
			if err == nil {
				t.Fatal("LoadCredentialConfig() error = nil, want retired-name error")
			}
			if !strings.Contains(err.Error(), tt.env) {
				t.Fatalf("error %q does not name %s", err, tt.env)
			}
			if strings.Contains(err.Error(), secretValue) {
				t.Fatalf("error leaked retired variable value: %q", err)
			}
		})
	}
}

func TestLoadCredentialConfigDoesNotDeriveTargetFromSAProjectID(t *testing.T) {
	t.Setenv("TEST_NO_FALLBACK_SA_PROJECT_ID", "credential-owner-project")
	_, err := LoadCredentialConfig("TEST_NO_FALLBACK_", "")
	if err == nil {
		t.Fatal("LoadCredentialConfig() error = nil, want retired/missing explicit project error")
	}
}

func TestCredentialConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *CredentialConfig
		wantErr bool
	}{
		{name: "valid", config: &CredentialConfig{EnvPrefix: "TEST_", ProjectID: "project"}},
		{name: "nil", config: nil, wantErr: true},
		{name: "missing prefix", config: &CredentialConfig{ProjectID: "project"}, wantErr: true},
		{name: "missing project", config: &CredentialConfig{EnvPrefix: "TEST_"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
