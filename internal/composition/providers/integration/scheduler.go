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
//   - "mock_scheduler" → Mock scheduler provider
//
// No aliases — the canonical token must be used verbatim. Unknown or empty values
// produce an error directing the user to the canonical token.
func CreateSchedulerProvider() (ports.SchedulerProvider, error) {
	providerName := strings.ToLower(os.Getenv("CONFIG_SCHEDULER_PROVIDER"))

	// Reject retired aliases with a clear migration message.
	switch providerName {
	case "mock":
		return nil, fmt.Errorf("scheduler provider 'mock' is retired - use CONFIG_SCHEDULER_PROVIDER=mock_scheduler")
	case "":
		// Scheduler is optional — not configured means skip.
		return nil, nil
	}

	// Let the provider build and configure itself from environment
	providerInstance, err := registry.BuildSchedulerProviderFromEnv(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler provider '%s': %w", providerName, err)
	}

	return providerInstance, nil
}

// CreateSchedulerProviders creates all scheduler providers specified in CONFIG_SCHEDULER_PROVIDER.
// Supports comma-separated values (e.g., "calendly,google_calendar").
// All providers are active simultaneously — the domain layer picks per-operation.
// Returns a map keyed by provider name.
func CreateSchedulerProviders() (map[string]ports.SchedulerProvider, error) {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("CONFIG_SCHEDULER_PROVIDER")))
	if raw == "" {
		// Scheduler is optional — not configured means skip.
		return nil, nil
	}

	names := strings.Split(raw, ",")
	providers := make(map[string]ports.SchedulerProvider)

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if name == "mock" {
			return nil, fmt.Errorf("scheduler provider 'mock' is retired - use mock_scheduler")
		}

		provider, err := registry.BuildSchedulerProviderFromEnv(name)
		if err != nil {
			fmt.Printf("⚠️ Failed to initialize scheduler provider '%s': %v\n", name, err)
			continue
		}
		if provider != nil {
			providers[name] = provider
		}
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no scheduler providers could be initialized from CONFIG_SCHEDULER_PROVIDER=%s", raw)
	}

	return providers, nil
}
