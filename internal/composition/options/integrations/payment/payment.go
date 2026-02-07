package payment

import (
	"fmt"
	"os"
	"strings"
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

// PaymentConfigSetter defines methods for setting payment configuration
type PaymentConfigSetter interface {
	SetPaymentConfig(config interface{})
}

// =============================================================================
// PAYMENT CONFIGURATION TYPES
// =============================================================================

// PaymentConfig holds configuration for payment providers
type PaymentConfig struct {
	AsiaPay *AsiaPayConfig `json:"asiapay,omitempty"`
	Stripe  *StripeConfig  `json:"stripe,omitempty"`
	Mock    bool           `json:"mock,omitempty"`
}

// AsiaPayConfig holds configuration for AsiaPay payment provider
type AsiaPayConfig struct {
	MerchantID   string `json:"merchant_id"`
	SecureSecret string `json:"secure_secret"`
	CurrencyCode string `json:"currency_code"`
	SandboxMode  bool   `json:"sandbox_mode"`
	BaseURL      string `json:"base_url,omitempty"`
	SuccessPath  string `json:"success_path,omitempty"`
	FailurePath  string `json:"failure_path,omitempty"`
	CancelPath   string `json:"cancel_path,omitempty"`
	WebhookPath  string `json:"webhook_path,omitempty"`
}

// Validate validates the AsiaPay configuration
func (c AsiaPayConfig) Validate() error {
	if c.MerchantID == "" {
		return fmt.Errorf("asiapay merchant ID is required")
	}
	if c.SecureSecret == "" {
		return fmt.Errorf("asiapay secure secret is required")
	}
	return nil
}

// StripeConfig holds configuration for Stripe payment provider
type StripeConfig struct {
	APIKey         string `json:"api_key"`
	WebhookSecret  string `json:"webhook_secret"`
	PublishableKey string `json:"publishable_key"`
}

// Validate validates the Stripe configuration
func (c StripeConfig) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("stripe API key is required")
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

func createAsiaPayConfigFromEnv() AsiaPayConfig {
	return AsiaPayConfig{
		MerchantID:   getEnv("ASIAPAY_MERCHANT_ID", ""),
		SecureSecret: getEnv("ASIAPAY_SECURE_SECRET", ""),
		CurrencyCode: getEnv("ASIAPAY_CURRENCY_CODE", "608"), // PHP default
		SandboxMode:  getEnv("ASIAPAY_SANDBOX_MODE", "true") == "true",
		BaseURL:      getEnv("ASIAPAY_BASE_URL", ""),
		SuccessPath:  getEnv("ASIAPAY_SUCCESS_PATH", "/payment/success"),
		FailurePath:  getEnv("ASIAPAY_FAILURE_PATH", "/payment/failure"),
		CancelPath:   getEnv("ASIAPAY_CANCEL_PATH", "/payment/cancel"),
		WebhookPath:  getEnv("ASIAPAY_WEBHOOK_PATH", "/webhooks/asiapay"),
	}
}

func createStripeConfigFromEnv() StripeConfig {
	return StripeConfig{
		APIKey:         getEnv("STRIPE_API_KEY", ""),
		WebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
		PublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
	}
}

// =============================================================================
// PAYMENT PROVIDER OPTIONS
// =============================================================================

// WithPaymentFromEnv dynamically selects payment provider based on CONFIG_PAYMENT_PROVIDER
func WithPaymentFromEnv() ContainerOption {
	return func(c Container) error {
		paymentProvider := strings.ToLower(getEnv("CONFIG_PAYMENT_PROVIDER", "mock"))

		switch paymentProvider {
		case "asiapay":
			return WithAsiaPay(createAsiaPayConfigFromEnv())(c)
		case "stripe":
			return WithStripe(createStripeConfigFromEnv())(c)
		case "mock", "":
			return WithMockPayment()(c)
		default:
			return fmt.Errorf("unsupported payment provider: %s", paymentProvider)
		}
	}
}

// WithAsiaPay configures AsiaPay as payment provider
func WithAsiaPay(config AsiaPayConfig) ContainerOption {
	return func(c Container) error {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid asiapay configuration: %w", err)
		}

		if setter, ok := c.(PaymentConfigSetter); ok {
			setter.SetPaymentConfig(PaymentConfig{AsiaPay: &config})
		} else {
			return fmt.Errorf("container does not implement SetPaymentConfig method")
		}

		mode := "production"
		if config.SandboxMode {
			mode = "sandbox"
		}
		fmt.Printf("💳 Configured AsiaPay: %s (%s)\n", config.MerchantID, mode)
		return nil
	}
}

// WithStripe configures Stripe as payment provider
func WithStripe(config StripeConfig) ContainerOption {
	return func(c Container) error {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid stripe configuration: %w", err)
		}

		if setter, ok := c.(PaymentConfigSetter); ok {
			setter.SetPaymentConfig(PaymentConfig{Stripe: &config})
		} else {
			return fmt.Errorf("container does not implement SetPaymentConfig method")
		}

		fmt.Printf("💳 Configured Stripe\n")
		return nil
	}
}

// WithMockPayment configures mock payment for testing/development
func WithMockPayment() ContainerOption {
	return func(c Container) error {
		if setter, ok := c.(PaymentConfigSetter); ok {
			setter.SetPaymentConfig(PaymentConfig{Mock: true})
		} else {
			return fmt.Errorf("container does not implement SetPaymentConfig method")
		}

		fmt.Printf("🧪 Configured mock payment\n")
		return nil
	}
}
