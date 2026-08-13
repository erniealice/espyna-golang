package gcp

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"google.golang.org/api/option"
)

// CredentialConfig separates resource selection from credential selection.
// ProjectID is supplied explicitly by the owning provider config. CredentialsPath
// is an optional, concern-scoped local escape hatch; an empty path means ADC.
type CredentialConfig struct {
	EnvPrefix       string
	ProjectID       string
	CredentialsPath string
}

// LoadCredentialConfig loads the strict Google credential contract for one
// concern. The caller supplies the already-resolved resource project so a
// credential document's project_id can never silently become the target.
//
// Managed Google runtimes should leave {prefix}CREDENTIALS_FILE unset and use
// Application Default Credentials from the attached workload identity. A
// credentials file remains available for local/non-managed environments.
func LoadCredentialConfig(envPrefix, projectID string) (*CredentialConfig, error) {
	prefix := strings.TrimSpace(envPrefix)
	if prefix == "" {
		return nil, fmt.Errorf("environment prefix is required")
	}

	if retired := retiredCredentialEnvironmentNames(prefix); len(retired) > 0 {
		return nil, fmt.Errorf(
			"retired credential environment variables are set: %s; use %sPROJECT_ID with ADC or %sCREDENTIALS_FILE",
			strings.Join(retired, ", "), prefix, prefix,
		)
	}

	config := &CredentialConfig{
		EnvPrefix:       prefix,
		ProjectID:       strings.TrimSpace(projectID),
		CredentialsPath: strings.TrimSpace(os.Getenv(prefix + "CREDENTIALS_FILE")),
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return config, nil
}

// retiredCredentialEnvironmentNames checks names, not values. Even an empty or
// "false" legacy variable is rejected so deployments have one unambiguous
// credential path and cannot appear migrated while still shipping stale keys.
func retiredCredentialEnvironmentNames(prefix string) []string {
	seen := make(map[string]struct{})
	for _, suffix := range []string{"USE_SERVICE_ACCOUNT", "SERVICE_ACCOUNT_KEY_PATH"} {
		name := prefix + suffix
		if _, ok := os.LookupEnv(name); ok {
			seen[name] = struct{}{}
		}
	}

	saPrefix := prefix + "SA_"
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, saPrefix) {
			seen[name] = struct{}{}
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetClientOption returns a concern-scoped credentials-file option or nil for
// ADC. It never mutates GOOGLE_APPLICATION_CREDENTIALS or another process-global
// credential source.
func GetClientOption(config *CredentialConfig) (option.ClientOption, error) {
	if config == nil {
		return nil, fmt.Errorf("credential config is required")
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if config.CredentialsPath != "" {
		return option.WithCredentialsFile(config.CredentialsPath), nil
	}
	return nil, nil
}

// Validate checks the target/credential configuration without opening a file or
// contacting Google. SDK construction owns those provider-specific checks.
func (c *CredentialConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("credential config is required")
	}
	if strings.TrimSpace(c.EnvPrefix) == "" {
		return fmt.Errorf("environment prefix is required")
	}
	if strings.TrimSpace(c.ProjectID) == "" {
		return fmt.Errorf("%sPROJECT_ID is required", c.EnvPrefix)
	}
	return nil
}
