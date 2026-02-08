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
