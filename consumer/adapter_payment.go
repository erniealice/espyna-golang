package consumer

import (
	"context"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	paymentpb "leapfor.xyz/esqyma/golang/v1/integration/payment"
)

/*
 ESPYNA CONSUMER APP - Technology-Agnostic Payment Adapter

Provides direct access to payment gateway operations without requiring
the full use cases/provider initialization chain.

This adapter works with ANY payment provider (AsiaPay, Stripe, PayMaya, GCash, etc.)
based on your CONFIG_PAYMENT_PROVIDER environment variable.

Usage:

	// Option 1: Get from container (recommended)
	container := consumer.NewContainerFromEnv()
	adapter := consumer.NewPaymentAdapterFromContainer(container)

	// Option 2: Standalone (legacy compatibility - requires build tags)
	// adapter, err := consumer.NewAsiaPayAdapterFromEnv(ctx)

	// Create checkout session
	session, err := adapter.CreateCheckoutSession(ctx, consumer.CheckoutSessionParams{
	    Amount:      76752,  // Amount in cents
	    Currency:    "PHP",
	    Description: "Subscription Payment",
	    PaymentID:   "pay_123",
	    OrderRef:    "order_456",
	})

	// Process webhook
	result, err := adapter.ProcessWebhook(ctx, payload, "application/x-www-form-urlencoded", headers)
*/

// PaymentAdapter provides technology-agnostic access to payment gateways.
// It wraps the PaymentProvider interface and works with AsiaPay, Stripe, PayMaya, etc.
type PaymentAdapter struct {
	provider  ports.PaymentProvider
	container *Container
}

// NewPaymentAdapterFromContainer creates a PaymentAdapter from an existing container.
// This is the recommended way to create the adapter as it reuses the container's provider.
func NewPaymentAdapterFromContainer(container *Container) *PaymentAdapter {
	if container == nil {
		return nil
	}

	provider := container.GetPaymentProvider()
	if provider == nil {
		return nil
	}

	return &PaymentAdapter{
		provider:  provider,
		container: container,
	}
}

// Close closes the payment adapter.
// Note: If created from container, this does NOT close the container.
func (a *PaymentAdapter) Close() error {
	// Don't close the container here - let the caller manage it
	return nil
}

// GetProvider returns the underlying PaymentProvider for advanced operations.
func (a *PaymentAdapter) GetProvider() ports.PaymentProvider {
	return a.provider
}

// Name returns the name of the underlying payment provider (e.g., "asiapay", "stripe", "mock")
func (a *PaymentAdapter) Name() string {
	if a.provider == nil {
		return ""
	}
	return a.provider.Name()
}

// IsEnabled returns whether the payment provider is enabled
func (a *PaymentAdapter) IsEnabled() bool {
	return a.provider != nil && a.provider.IsEnabled()
}

// --- Payment Operations ---

// CreateCheckoutSession creates a new payment checkout session.
// Returns a session with the checkout URL where customers can complete payment.
func (a *PaymentAdapter) CreateCheckoutSession(ctx context.Context, params CheckoutSessionParams) (*paymentpb.CheckoutSession, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("payment provider not initialized")
	}

	req := params.ToProtoRequest()
	resp, err := a.provider.CreateCheckoutSession(ctx, req)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		errMsg := "unknown error"
		if resp.Error != nil {
			errMsg = resp.Error.Message
		}
		return nil, fmt.Errorf("failed to create checkout session: %s", errMsg)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no checkout session data returned")
	}
	return resp.Data[0], nil
}

// ProcessWebhook processes an incoming payment webhook/datafeed.
// Returns the parsed transaction and payment status.
func (a *PaymentAdapter) ProcessWebhook(ctx context.Context, payload []byte, contentType string, headers map[string]string) (*PaymentWebhookResult, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("payment provider not initialized")
	}

	req := &paymentpb.ProcessWebhookRequest{
		Data: &paymentpb.WebhookData{
			ProviderId:  a.provider.Name(),
			Payload:     payload,
			Headers:     headers,
			ContentType: contentType,
		},
	}

	resp, err := a.provider.ProcessWebhook(ctx, req)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		errMsg := "unknown error"
		if resp.Error != nil {
			errMsg = resp.Error.Message
		}
		return &PaymentWebhookResult{
			Success: false,
			Action:  "error",
			Error:   fmt.Errorf("%s", errMsg),
		}, nil
	}

	if len(resp.Data) == 0 {
		return &PaymentWebhookResult{
			Success: false,
			Action:  "error",
			Error:   fmt.Errorf("no webhook data returned"),
		}, nil
	}

	data := resp.Data[0]

	// Map protobuf status to string action
	action := "unknown"
	switch data.Status {
	case paymentpb.PaymentStatus_PAYMENT_STATUS_SUCCESS:
		action = "success"
	case paymentpb.PaymentStatus_PAYMENT_STATUS_FAILED:
		action = "failure"
	case paymentpb.PaymentStatus_PAYMENT_STATUS_CANCELLED:
		action = "cancelled"
	case paymentpb.PaymentStatus_PAYMENT_STATUS_PENDING:
		action = "pending"
	}

	return &PaymentWebhookResult{
		Success:     true,
		PaymentID:   data.PaymentId,
		Status:      data.Status,
		Action:      action,
		Transaction: data.Transaction,
	}, nil
}

// GetPaymentStatus retrieves the status of a payment.
// Note: Not all providers support direct status queries - use webhooks when available.
func (a *PaymentAdapter) GetPaymentStatus(ctx context.Context, paymentID string) (paymentpb.PaymentStatus, error) {
	if a.provider == nil {
		return paymentpb.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED, fmt.Errorf("payment provider not initialized")
	}

	req := &paymentpb.GetPaymentStatusRequest{
		Data: &paymentpb.PaymentStatusLookup{
			ProviderId: a.provider.Name(),
			PaymentId:  paymentID,
		},
	}

	resp, err := a.provider.GetPaymentStatus(ctx, req)
	if err != nil {
		return paymentpb.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED, err
	}
	if !resp.Success {
		errMsg := "unknown error"
		if resp.Error != nil {
			errMsg = resp.Error.Message
		}
		return paymentpb.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED, fmt.Errorf("%s", errMsg)
	}
	if len(resp.Data) == 0 {
		return paymentpb.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED, fmt.Errorf("no payment status data returned")
	}
	return resp.Data[0].Status, nil
}

// RefundPayment initiates a refund for a payment.
// transactionID is the provider's transaction reference.
func (a *PaymentAdapter) RefundPayment(ctx context.Context, transactionID string, amount int64, reason string) (*paymentpb.RefundPaymentResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("payment provider not initialized")
	}

	req := &paymentpb.RefundPaymentRequest{
		Data: &paymentpb.RefundData{
			ProviderId:    a.provider.Name(),
			TransactionId: transactionID,
			Amount:        amount,
			Reason:        reason,
		},
	}

	return a.provider.RefundPayment(ctx, req)
}

// IsHealthy checks if the payment provider is healthy and available.
func (a *PaymentAdapter) IsHealthy(ctx context.Context) error {
	if a.provider == nil {
		return fmt.Errorf("payment provider not initialized")
	}
	return a.provider.IsHealthy(ctx)
}

// GetCapabilities returns the capabilities supported by the payment provider.
func (a *PaymentAdapter) GetCapabilities() []paymentpb.PaymentCapability {
	if a.provider == nil {
		return nil
	}
	return a.provider.GetCapabilities()
}

// GetSupportedCurrencies returns currencies supported by the payment provider.
func (a *PaymentAdapter) GetSupportedCurrencies() []string {
	if a.provider == nil {
		return nil
	}
	return a.provider.GetSupportedCurrencies()
}

// --- Convenience Methods ---

// CreateQuickCheckout creates a checkout session with minimal parameters.
func (a *PaymentAdapter) CreateQuickCheckout(ctx context.Context, amount float64, currency, paymentID, description string) (*paymentpb.CheckoutSession, error) {
	return a.CreateCheckoutSession(ctx, CheckoutSessionParams{
		Amount:      amount,
		Currency:    currency,
		PaymentID:   paymentID,
		OrderRef:    paymentID,
		Description: description,
	})
}

// CreateSubscriptionCheckout creates a checkout for a subscription payment.
func (a *PaymentAdapter) CreateSubscriptionCheckout(ctx context.Context, amount float64, currency, paymentID, subscriptionID, clientID, description string) (*paymentpb.CheckoutSession, error) {
	return a.CreateCheckoutSession(ctx, CheckoutSessionParams{
		Amount:         amount,
		Currency:       currency,
		PaymentID:      paymentID,
		SubscriptionID: subscriptionID,
		ClientID:       clientID,
		OrderRef:       paymentID,
		Description:    description,
	})
}

// --- Re-export types for consumer convenience ---

// CheckoutSessionParams re-exports the CheckoutSessionParams type for consumer convenience
type CheckoutSessionParams = ports.CheckoutSessionParams

// PaymentWebhookResult re-exports the PaymentWebhookResult type for consumer convenience
type PaymentWebhookResult = ports.PaymentWebhookResult
