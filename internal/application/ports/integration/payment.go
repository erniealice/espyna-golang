package integration

import (
	"context"

	paymentpb "leapfor.xyz/esqyma/golang/v1/integration/payment"
)

// IntegrationPaymentRepository defines the interface for integration payment operations.
// Database adapters (firestore, postgres, mock) implement this interface behind build tags.
type IntegrationPaymentRepository interface {
	// LogWebhook saves parsed webhook data to the integration_payment table/collection
	LogWebhook(ctx context.Context, req *paymentpb.LogWebhookRequest) (*paymentpb.LogWebhookResponse, error)
}

// PaymentProvider defines the contract for payment providers
// This interface abstracts payment services like AsiaPay, Stripe, PayPal, GCash, PayMaya, etc.
// following the hexagonal architecture pattern established for EmailProvider.
type PaymentProvider interface {
	// Name returns the name of the payment provider (e.g., "asiapay", "stripe", "mock")
	Name() string

	// Initialize sets up the payment provider with the given configuration
	Initialize(config *paymentpb.PaymentProviderConfig) error

	// CreateCheckoutSession creates a new payment checkout session
	// Returns a session with checkout URL where customer can complete payment
	CreateCheckoutSession(ctx context.Context, req *paymentpb.CreateCheckoutSessionRequest) (*paymentpb.CreateCheckoutSessionResponse, error)

	// ProcessWebhook processes incoming webhook/datafeed from payment provider
	// Parses and validates webhook data, returns transaction details
	ProcessWebhook(ctx context.Context, req *paymentpb.ProcessWebhookRequest) (*paymentpb.ProcessWebhookResponse, error)

	// GetPaymentStatus retrieves the status of a payment from the provider
	GetPaymentStatus(ctx context.Context, req *paymentpb.GetPaymentStatusRequest) (*paymentpb.GetPaymentStatusResponse, error)

	// RefundPayment initiates a refund for a payment
	RefundPayment(ctx context.Context, req *paymentpb.RefundPaymentRequest) (*paymentpb.RefundPaymentResponse, error)

	// IsHealthy checks if the payment service is available
	IsHealthy(ctx context.Context) error

	// Close cleans up payment provider resources
	Close() error

	// IsEnabled returns whether this provider is currently enabled
	IsEnabled() bool

	// GetCapabilities returns the capabilities supported by this provider
	GetCapabilities() []paymentpb.PaymentCapability

	// GetSupportedCurrencies returns the currency codes supported by this provider
	GetSupportedCurrencies() []string
}

// PaymentWebhookResult represents the result of processing a payment webhook
// This is a convenience type for use cases that need to act on webhook results
type PaymentWebhookResult struct {
	// Success indicates if webhook processing was successful
	Success bool

	// PaymentID is the internal payment document ID
	PaymentID string

	// SubscriptionID is the subscription associated with this payment (if any)
	SubscriptionID string

	// Status is the current payment status
	Status paymentpb.PaymentStatus

	// IsFirstPayment indicates if this is the first payment for a subscription
	IsFirstPayment bool

	// Action describes what action was taken (success, failure, pending, cancelled)
	Action string

	// Transaction contains the full transaction details
	Transaction *paymentpb.PaymentTransaction

	// Error contains any error that occurred during processing
	Error error
}

// CheckoutSessionParams provides a simplified parameter structure for creating checkout sessions
// Consumer adapters can use this for convenience instead of the full protobuf request
type CheckoutSessionParams struct {
	// Amount in base currency units (e.g., dollars, not cents)
	// Payment providers typically convert to cents internally
	Amount float64

	// Currency code (ISO 4217: USD, PHP, HKD)
	Currency string

	// Description of the payment
	Description string

	// PaymentID is the internal payment document ID
	PaymentID string

	// SubscriptionID is the subscription this payment is for (optional)
	SubscriptionID string

	// ClientID is the client making the payment
	ClientID string

	// OrderRef is a merchant reference for the order
	OrderRef string

	// CustomerEmail is the customer's email address
	CustomerEmail string

	// CustomerName is the customer's name
	CustomerName string

	// CustomerPhone is the customer's phone number
	CustomerPhone string

	// Metadata contains additional key-value data
	Metadata map[string]string

	// SuccessURL overrides the default success redirect URL
	SuccessURL string

	// FailureURL overrides the default failure redirect URL
	FailureURL string

	// CancelURL overrides the default cancel redirect URL
	CancelURL string

	// ExpiresInMinutes sets the session expiration time
	ExpiresInMinutes int32
}

// ToProtoRequest converts CheckoutSessionParams to the protobuf request type
func (p *CheckoutSessionParams) ToProtoRequest() *paymentpb.CreateCheckoutSessionRequest {
	return &paymentpb.CreateCheckoutSessionRequest{
		Data: &paymentpb.CheckoutSessionData{
			Amount:           p.Amount,
			Currency:         p.Currency,
			Description:      p.Description,
			PaymentId:        p.PaymentID,
			SubscriptionId:   p.SubscriptionID,
			ClientId:         p.ClientID,
			OrderRef:         p.OrderRef,
			SuccessUrl:       p.SuccessURL,
			FailureUrl:       p.FailureURL,
			CancelUrl:        p.CancelURL,
			Metadata:         p.Metadata,
			ExpiresInMinutes: p.ExpiresInMinutes,
			Customer: &paymentpb.CustomerInfo{
				Email: p.CustomerEmail,
				Name:  p.CustomerName,
				Phone: p.CustomerPhone,
			},
		},
	}
}
