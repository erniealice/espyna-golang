package tabular

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// =============================================================================
// CONTAINER INTERFACE
// =============================================================================

// Container defines the interface for container configuration
type Container interface {
	GetConfig() interface{}
	SetConfig(interface{})
}

// ContainerOption defines a function that can configure a Container
type ContainerOption func(Container) error

// TabularConfigSetter defines methods for setting tabular configuration
type TabularConfigSetter interface {
	SetTabularConfig(config interface{})
}

// =============================================================================
// TABULAR CONFIGURATION TYPES
// =============================================================================

// TabularConfig holds configuration for tabular providers
type TabularConfig struct {
	GoogleSheets *GoogleSheetsConfig `json:"google_sheets,omitempty"`
	CSV          *CSVConfig          `json:"csv,omitempty"`
	Mock         bool                `json:"mock,omitempty"`
}

// GoogleSheetsConfig holds Google Sheets specific configuration
type GoogleSheetsConfig struct {
	DelegateEmail         string        `json:"delegate_email"`
	ServiceAccountKeyPath string        `json:"service_account_key_path"`
	SecretManagerPath     string        `json:"secret_manager_path,omitempty"`
	UseSecretManager      bool          `json:"use_secret_manager,omitempty"`
	ProjectID             string        `json:"project_id,omitempty"`
	Timeout               time.Duration `json:"timeout,omitempty"`
}

// Validate validates the Google Sheets configuration
func (c GoogleSheetsConfig) Validate() error {
	if c.DelegateEmail == "" {
		return fmt.Errorf("delegate_email is required")
	}
	if c.ServiceAccountKeyPath == "" && !c.UseSecretManager {
		return fmt.Errorf("service_account_key_path is required when not using secret manager")
	}
	if c.UseSecretManager && c.SecretManagerPath == "" {
		return fmt.Errorf("secret_manager_path is required when using secret manager")
	}
	return nil
}

// CSVConfig holds CSV specific configuration
type CSVConfig struct {
	BasePath  string `json:"base_path"`
	Delimiter string `json:"delimiter,omitempty"`
	Encoding  string `json:"encoding,omitempty"`
	ReadOnly  bool   `json:"read_only,omitempty"`
}

// Validate validates the CSV configuration
func (c CSVConfig) Validate() error {
	if c.BasePath == "" {
		return fmt.Errorf("base_path is required for CSV provider")
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

func createGoogleSheetsConfigFromEnv() GoogleSheetsConfig {
	timeout := 30 * time.Second
	if t := os.Getenv("LEAPFOR_INTEGRATION_TABULAR_GOOGLESHEETS_TIMEOUT"); t != "" {
		if d, err := time.ParseDuration(t + "s"); err == nil {
			timeout = d
		}
	}

	return GoogleSheetsConfig{
		DelegateEmail:         os.Getenv("LEAPFOR_INTEGRATION_TABULAR_GOOGLESHEETS_DELEGATE_EMAIL"),
		ServiceAccountKeyPath: os.Getenv("LEAPFOR_INTEGRATION_TABULAR_GOOGLESHEETS_SERVICE_ACCOUNT_KEY_PATH"),
		SecretManagerPath:     os.Getenv("LEAPFOR_INTEGRATION_TABULAR_GOOGLESHEETS_SECRET_MANAGER_PATH"),
		UseSecretManager:      os.Getenv("LEAPFOR_INTEGRATION_TABULAR_GOOGLESHEETS_USE_SECRET_MANAGER") == "true",
		ProjectID:             os.Getenv("LEAPFOR_INTEGRATION_TABULAR_GOOGLESHEETS_PROJECT_ID"),
		Timeout:               timeout,
	}
}

func createCSVConfigFromEnv() CSVConfig {
	return CSVConfig{
		BasePath:  os.Getenv("LEAPFOR_INTEGRATION_TABULAR_CSV_BASE_PATH"),
		Delimiter: getEnv("LEAPFOR_INTEGRATION_TABULAR_CSV_DELIMITER", ","),
		Encoding:  getEnv("LEAPFOR_INTEGRATION_TABULAR_CSV_ENCODING", "utf-8"),
		ReadOnly:  os.Getenv("LEAPFOR_INTEGRATION_TABULAR_CSV_READ_ONLY") == "true",
	}
}

// =============================================================================
// TABULAR PROVIDER OPTIONS
// =============================================================================

// WithTabularFromEnv dynamically selects tabular provider based on CONFIG_TABULAR_PROVIDER
func WithTabularFromEnv() ContainerOption {
	return func(c Container) error {
		tabularProvider := strings.ToLower(getEnv("CONFIG_TABULAR_PROVIDER", "mock"))

		switch tabularProvider {
		case "googlesheets", "google_sheets":
			return WithGoogleSheets(createGoogleSheetsConfigFromEnv())(c)
		case "csv":
			return WithCSV(createCSVConfigFromEnv())(c)
		case "mock", "mock_tabular", "":
			return WithMockTabular()(c)
		default:
			return fmt.Errorf("unsupported tabular provider: %s", tabularProvider)
		}
	}
}

// WithGoogleSheets configures Google Sheets as tabular provider
func WithGoogleSheets(config GoogleSheetsConfig) ContainerOption {
	return func(c Container) error {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid google sheets configuration: %w", err)
		}

		if setter, ok := c.(TabularConfigSetter); ok {
			setter.SetTabularConfig(TabularConfig{GoogleSheets: &config})
		} else {
			return fmt.Errorf("container does not implement SetTabularConfig method")
		}

		fmt.Printf("Configured Google Sheets tabular provider: %s\n", config.DelegateEmail)
		return nil
	}
}

// WithCSV configures CSV as tabular provider
func WithCSV(config CSVConfig) ContainerOption {
	return func(c Container) error {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid csv configuration: %w", err)
		}

		if setter, ok := c.(TabularConfigSetter); ok {
			setter.SetTabularConfig(TabularConfig{CSV: &config})
		} else {
			return fmt.Errorf("container does not implement SetTabularConfig method")
		}

		mode := "read-write"
		if config.ReadOnly {
			mode = "read-only"
		}
		fmt.Printf("Configured CSV tabular provider: %s (%s)\n", config.BasePath, mode)
		return nil
	}
}

// WithMockTabular configures mock tabular for testing/development
func WithMockTabular() ContainerOption {
	return func(c Container) error {
		if setter, ok := c.(TabularConfigSetter); ok {
			setter.SetTabularConfig(TabularConfig{Mock: true})
		} else {
			return fmt.Errorf("container does not implement SetTabularConfig method")
		}

		fmt.Printf("Configured mock tabular provider\n")
		return nil
	}
}
