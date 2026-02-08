//go:build mock_db

package integration

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/payment"
)

func init() {
	registry.RegisterRepositoryFactory("mock", "integration_payment", func(conn any, tableName string) (any, error) {
		return NewMockIntegrationPaymentRepository(), nil
	})
}

// MockIntegrationPaymentRepository implements IntegrationPaymentRepository with in-memory storage
type MockIntegrationPaymentRepository struct {
	webhooks map[string]map[string]any
	mutex    sync.RWMutex
}

// NewMockIntegrationPaymentRepository creates a new mock integration payment repository
func NewMockIntegrationPaymentRepository() *MockIntegrationPaymentRepository {
	return &MockIntegrationPaymentRepository{
		webhooks: make(map[string]map[string]any),
	}
}

// LogWebhook saves parsed webhook data to in-memory storage
func (r *MockIntegrationPaymentRepository) LogWebhook(ctx context.Context, req *paymentpb.LogWebhookRequest) (*paymentpb.LogWebhookResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("log webhook data is required")
	}

	data := req.Data

	id := data.ExecutionId
	if id == "" {
		id = uuid.New().String()
	}

	r.mutex.Lock()
	r.webhooks[id] = map[string]any{
		"id":             id,
		"payment_id":     data.PaymentId,
		"provider_id":    data.ProviderId,
		"payment_status": data.PaymentStatus,
		"amount":         data.Amount,
		"currency":       data.Currency,
	}
	r.mutex.Unlock()

	log.Printf("[mock] Logged integration payment webhook: id=%s, provider=%s", id, data.ProviderId)

	return &paymentpb.LogWebhookResponse{
		Success: true,
		Id:      id,
	}, nil
}
