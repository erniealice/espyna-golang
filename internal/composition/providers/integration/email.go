package integration

import (
	"fmt"
	"os"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// CreateEmailProvider creates an email provider using provider self-configuration.
// The provider reads its own environment variables - composition layer is provider-agnostic.
//
// Uses CONFIG_EMAIL_PROVIDER environment variable to select which provider to use:
//   - "google_email" → Gmail provider
//   - "microsoft_email" → Microsoft 365/Outlook provider
//   - "mock_email" → Mock email provider
//
// No aliases — the canonical token must be used verbatim. Unknown values produce an error.
func CreateEmailProvider() (ports.EmailProvider, error) {
	providerName := strings.ToLower(os.Getenv("CONFIG_EMAIL_PROVIDER"))

	// Debug: Log what we're trying to create
	fmt.Printf("[CreateEmailProvider] CONFIG_EMAIL_PROVIDER=%q\n", providerName)
	fmt.Printf("[CreateEmailProvider] Available providers: %v\n", registry.ListAvailableEmailBuildFromEnv())

	// Let the provider build and configure itself from environment
	providerInstance, err := registry.BuildEmailProviderFromEnv(providerName)
	if err != nil {
		fmt.Printf("[CreateEmailProvider] ERROR: %v\n", err)
		return nil, fmt.Errorf("failed to create email provider '%s': %w", providerName, err)
	}

	fmt.Printf("[CreateEmailProvider] SUCCESS: Created provider %q\n", providerInstance.Name())
	return providerInstance, nil
}
