package integration

import (
	"fmt"
	"os"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// CreatePaymentProvider creates a payment provider using provider self-configuration.
// The provider reads its own environment variables - composition layer is provider-agnostic.
//
// Uses CONFIG_PAYMENT_PROVIDER environment variable to select which provider to use:
//   - "asiapay" → AsiaPay payment gateway
//   - "stripe" → Stripe payment gateway
//   - "mock_payment", "mock", or "" → Mock payment provider (default)
func CreatePaymentProvider() (ports.PaymentProvider, error) {
	providerName := strings.ToLower(os.Getenv("CONFIG_PAYMENT_PROVIDER"))

	// Normalize provider names
	switch providerName {
	case "mock", "":
		providerName = "mock_payment"
	}

	// Let the provider build and configure itself from environment
	providerInstance, err := registry.BuildPaymentProviderFromEnv(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment provider '%s': %w", providerName, err)
	}

	return providerInstance, nil
}

// CreatePaymentProviders creates all payment providers specified in CONFIG_PAYMENT_PROVIDER.
// Supports comma-separated values (e.g., "asiapay,maya,paypal").
// All providers are active simultaneously — the domain layer picks per-operation.
// Returns a map keyed by provider name.
func CreatePaymentProviders() (map[string]ports.PaymentProvider, error) {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("CONFIG_PAYMENT_PROVIDER")))
	if raw == "" || raw == "mock" {
		raw = "mock_payment"
	}

	names := strings.Split(raw, ",")
	providers := make(map[string]ports.PaymentProvider)

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		// Normalize
		if name == "mock" {
			name = "mock_payment"
		}

		provider, err := registry.BuildPaymentProviderFromEnv(name)
		if err != nil {
			fmt.Printf("⚠️ Failed to initialize payment provider '%s': %v\n", name, err)
			continue
		}
		if provider != nil {
			providers[name] = provider
		}
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no payment providers could be initialized from CONFIG_PAYMENT_PROVIDER=%s", raw)
	}

	return providers, nil
}
