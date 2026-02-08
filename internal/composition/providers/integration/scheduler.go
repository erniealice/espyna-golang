package integration

import (
	"fmt"
	"os"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// CreateSchedulerProvider creates a scheduler provider using provider self-configuration.
// The provider reads its own environment variables - composition layer is provider-agnostic.
//
// Uses CONFIG_SCHEDULER_PROVIDER environment variable to select which provider to use:
//   - "calendly" → Calendly scheduling service
//   - "google_calendar" → Google Calendar
//   - "mock_scheduler", "mock", or "" → Mock scheduler provider (default)
func CreateSchedulerProvider() (ports.SchedulerProvider, error) {
	providerName := strings.ToLower(os.Getenv("CONFIG_SCHEDULER_PROVIDER"))

	// Normalize provider names
	switch providerName {
	case "mock", "":
		providerName = "mock_scheduler"
	}

	// Let the provider build and configure itself from environment
	providerInstance, err := registry.BuildSchedulerProviderFromEnv(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler provider '%s': %w", providerName, err)
	}

	return providerInstance, nil
}
