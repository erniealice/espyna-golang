package integration

import (
	"fmt"
	"sync"

	"leapfor.xyz/espyna/internal/application/ports"
)

// Registry manages integration provider instances (email, payment, etc.)
type Registry struct {
	mu sync.RWMutex

	email   ports.EmailProvider
	payment ports.PaymentProvider
}

// NewRegistry creates a new integration provider registry
func NewRegistry() *Registry {
	return &Registry{}
}

// InitializeAll creates and initializes all integration providers from environment.
// Each provider reads its own configuration from environment variables.
func (r *Registry) InitializeAll() error {
	// Initialize email provider
	emailProvider, err := CreateEmailProvider()
	if err != nil {
		return fmt.Errorf("failed to create email provider: %w", err)
	}
	r.SetEmail(emailProvider)

	// Initialize payment provider
	paymentProvider, err := CreatePaymentProvider()
	if err != nil {
		return fmt.Errorf("failed to create payment provider: %w", err)
	}
	r.SetPayment(paymentProvider)

	return nil
}

// SetEmail sets the email provider
func (r *Registry) SetEmail(provider ports.EmailProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.email = provider
}

// GetEmail returns the email provider
func (r *Registry) GetEmail() ports.EmailProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.email
}

// SetPayment sets the payment provider
func (r *Registry) SetPayment(provider ports.PaymentProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.payment = provider
}

// GetPayment returns the payment provider
func (r *Registry) GetPayment() ports.PaymentProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.payment
}

// Close closes all integration providers
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error

	if r.email != nil {
		if closer, ok := r.email.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				errs = append(errs, fmt.Errorf("email: %w", err))
			}
		}
	}

	if r.payment != nil {
		if closer, ok := r.payment.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				errs = append(errs, fmt.Errorf("payment: %w", err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing integration providers: %v", errs)
	}
	return nil
}
