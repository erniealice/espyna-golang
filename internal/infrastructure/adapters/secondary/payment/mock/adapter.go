//go:build mock_payment

package mock

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/payment"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterPaymentProvider(
		"mock_payment",
		func() ports.PaymentProvider {
			return NewMockPaymentProvider()
		},
		transformConfig,
	)
	registry.RegisterPaymentBuildFromEnv("mock_payment", buildFromEnv)
}

// buildFromEnv creates and initializes a mock payment provider.
func buildFromEnv() (ports.PaymentProvider, error) {
	protoConfig := &paymentpb.PaymentProviderConfig{
		ProviderId:   "mock",
		ProviderType: paymentpb.PaymentProviderType_PAYMENT_PROVIDER_TYPE_MOCK,
		Enabled:      true,
	}
	p := NewMockPaymentProvider()
	if err := p.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("mock_payment: failed to initialize: %w", err)
	}
	return p, nil
}

// transformConfig converts raw config map to mock payment proto config.
func transformConfig(rawConfig map[string]any) (*paymentpb.PaymentProviderConfig, error) {
	return &paymentpb.PaymentProviderConfig{
		ProviderId:   "mock",
		ProviderType: paymentpb.PaymentProviderType_PAYMENT_PROVIDER_TYPE_MOCK,
		Enabled:      true,
	}, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// MockPaymentProvider implements PaymentProvider for testing purposes
type MockPaymentProvider struct {
	enabled      bool
	sessions     map[string]*paymentpb.CheckoutSession
	transactions map[string]*paymentpb.PaymentTransaction
}

// NewMockPaymentProvider creates a new mock payment provider
func NewMockPaymentProvider() ports.PaymentProvider {
	return &MockPaymentProvider{
		enabled:      false,
		sessions:     make(map[string]*paymentpb.CheckoutSession),
		transactions: make(map[string]*paymentpb.PaymentTransaction),
	}
}

func (p *MockPaymentProvider) Name() string {
	return "mock"
}

func (p *MockPaymentProvider) Initialize(config *paymentpb.PaymentProviderConfig) error {
	p.enabled = config.Enabled
	log.Printf("✅ Mock payment provider initialized")
	return nil
}

func (p *MockPaymentProvider) CreateCheckoutSession(ctx context.Context, req *paymentpb.CreateCheckoutSessionRequest) (*paymentpb.CreateCheckoutSessionResponse, error) {
	if !p.enabled {
		return nil, fmt.Errorf("Mock payment provider is not initialized")
	}

	data := req.Data
	if data == nil {
		return &paymentpb.CreateCheckoutSessionResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:        "INVALID_REQUEST",
				Description: "Request data is required",
				Category:    commonpb.ErrorCategory_ERROR_CATEGORY_VALIDATION,
			},
		}, nil
	}

	merchantRef := data.OrderRef
	if merchantRef == "" {
		merchantRef = data.PaymentId
	}
	if merchantRef == "" {
		merchantRef = fmt.Sprintf("mock_%d", time.Now().UnixNano())
	}

	now := timestamppb.Now()
	expiresAt := timestamppb.New(time.Now().Add(30 * time.Minute))
	if data.ExpiresInMinutes > 0 {
		expiresAt = timestamppb.New(time.Now().Add(time.Duration(data.ExpiresInMinutes) * time.Minute))
	}

	session := &paymentpb.CheckoutSession{
		Id:                fmt.Sprintf("mock_%s", merchantRef),
		ProviderSessionId: merchantRef,
		ProviderId:        "mock",
		ProviderType:      paymentpb.PaymentProviderType_PAYMENT_PROVIDER_TYPE_MOCK,
		Amount:            data.Amount,
		Currency:          data.Currency,
		Status:            paymentpb.PaymentStatus_PAYMENT_STATUS_PENDING,
		CheckoutUrl:       fmt.Sprintf("http://localhost:8080/mock-payment?ref=%s&amount=%d", merchantRef, data.Amount),
		SuccessUrl:        data.SuccessUrl,
		FailureUrl:        data.FailureUrl,
		CancelUrl:         data.CancelUrl,
		PaymentId:         data.PaymentId,
		SubscriptionId:    data.SubscriptionId,
		ClientId:          data.ClientId,
		OrderRef:          merchantRef,
		Description:       data.Description,
		Metadata:          data.Metadata,
		ExpiresAt:         expiresAt,
		CreatedAt:         now,
		UpdatedAt:         now,
		ProviderData:      map[string]string{"mock": "true"},
	}

	p.sessions[session.Id] = session
	log.Printf("📦 Mock checkout session created: %s", session.Id)

	return &paymentpb.CreateCheckoutSessionResponse{Success: true, Data: []*paymentpb.CheckoutSession{session}}, nil
}

func (p *MockPaymentProvider) ProcessWebhook(ctx context.Context, req *paymentpb.ProcessWebhookRequest) (*paymentpb.ProcessWebhookResponse, error) {
	if !p.enabled {
		return nil, fmt.Errorf("Mock payment provider is not initialized")
	}

	data := req.Data
	if data == nil {
		return &paymentpb.ProcessWebhookResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:        "INVALID_REQUEST",
				Description: "Request data is required",
				Category:    commonpb.ErrorCategory_ERROR_CATEGORY_VALIDATION,
			},
		}, nil
	}

	transaction := &paymentpb.PaymentTransaction{
		Id:          fmt.Sprintf("txn_%d", time.Now().UnixNano()),
		ProviderId:  "mock",
		Status:      paymentpb.PaymentStatus_PAYMENT_STATUS_SUCCESS,
		ProcessedAt: timestamppb.Now(),
		RawData:     map[string]string{"mock": "true"},
	}

	p.transactions[transaction.Id] = transaction
	log.Printf("🔔 Mock webhook processed: %s", transaction.Id)

	return &paymentpb.ProcessWebhookResponse{
		Success: true,
		Data: &paymentpb.WebhookResult{
			Transaction: transaction,
			Status:      paymentpb.PaymentStatus_PAYMENT_STATUS_SUCCESS,
			Action:      "success",
		},
	}, nil
}

func (p *MockPaymentProvider) GetPaymentStatus(ctx context.Context, req *paymentpb.GetPaymentStatusRequest) (*paymentpb.GetPaymentStatusResponse, error) {
	if !p.enabled {
		return nil, fmt.Errorf("Mock payment provider is not initialized")
	}
	return &paymentpb.GetPaymentStatusResponse{
		Success: true,
		Data: []*paymentpb.PaymentStatusData{
			{
				Status: paymentpb.PaymentStatus_PAYMENT_STATUS_SUCCESS,
			},
		},
	}, nil
}

func (p *MockPaymentProvider) RefundPayment(ctx context.Context, req *paymentpb.RefundPaymentRequest) (*paymentpb.RefundPaymentResponse, error) {
	if !p.enabled {
		return nil, fmt.Errorf("Mock payment provider is not initialized")
	}
	data := req.Data
	amount := int64(0)
	if data != nil {
		amount = data.Amount
	}
	return &paymentpb.RefundPaymentResponse{
		Success: true,
		Data: []*paymentpb.RefundResponse{
			{
				Success:  true,
				RefundId: fmt.Sprintf("refund_%d", time.Now().UnixNano()),
				Status:   paymentpb.PaymentStatus_PAYMENT_STATUS_REFUNDED,
				Amount:   amount,
			},
		},
	}, nil
}

func (p *MockPaymentProvider) IsHealthy(ctx context.Context) error {
	if !p.enabled {
		return fmt.Errorf("Mock payment provider is not initialized")
	}
	return nil
}

func (p *MockPaymentProvider) Close() error {
	p.enabled = false
	p.sessions = make(map[string]*paymentpb.CheckoutSession)
	p.transactions = make(map[string]*paymentpb.PaymentTransaction)
	log.Printf("🔌 Mock payment provider closed")
	return nil
}

func (p *MockPaymentProvider) IsEnabled() bool {
	return p.enabled
}

func (p *MockPaymentProvider) GetCapabilities() []paymentpb.PaymentCapability {
	return []paymentpb.PaymentCapability{
		paymentpb.PaymentCapability_PAYMENT_CAPABILITY_ONE_TIME,
		paymentpb.PaymentCapability_PAYMENT_CAPABILITY_RECURRING,
		paymentpb.PaymentCapability_PAYMENT_CAPABILITY_REFUND,
		paymentpb.PaymentCapability_PAYMENT_CAPABILITY_WEBHOOKS,
	}
}

func (p *MockPaymentProvider) GetSupportedCurrencies() []string {
	return []string{"USD", "PHP", "EUR", "GBP", "HKD", "SGD"}
}

var _ ports.PaymentProvider = (*MockPaymentProvider)(nil)
