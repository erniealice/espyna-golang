//go:build firestore

package integration

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	integrationPorts "github.com/erniealice/espyna-golang/internal/application/ports/integration"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	firestoreCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/payment"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "integration_payment", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore integration_payment repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreIntegrationPaymentRepository(dbOps, collectionName), nil
	})
}

// FirestoreIntegrationPaymentRepository implements IntegrationPaymentRepository using Firestore
type FirestoreIntegrationPaymentRepository struct {
	dbOps          interfaces.DatabaseOperation
	collectionName string
}

// NewFirestoreIntegrationPaymentRepository creates a new Firestore integration payment repository
func NewFirestoreIntegrationPaymentRepository(dbOps interfaces.DatabaseOperation, collectionName string) integrationPorts.IntegrationPaymentRepository {
	if collectionName == "" {
		collectionName = "integration_payment"
	}
	return &FirestoreIntegrationPaymentRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
	}
}

// LogWebhook saves parsed webhook data to the integration_payment collection
func (r *FirestoreIntegrationPaymentRepository) LogWebhook(ctx context.Context, req *paymentpb.LogWebhookRequest) (*paymentpb.LogWebhookResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("log webhook data is required")
	}

	data := req.Data

	// Generate ID if not provided
	id := data.ExecutionId
	if id == "" {
		id = uuid.New().String()
	}

	// Build document for Firestore
	now := time.Now()
	doc := map[string]any{
		"id":                   id,
		"payment_id":           data.PaymentId,
		"provider_id":          data.ProviderId,
		"provider_ref":         data.ProviderRef,
		"provider_payment_ref": data.ProviderPaymentRef,
		"payment_status":       data.PaymentStatus,
		"amount":               data.Amount,
		"currency":             data.Currency,
		"payment_method":       data.PaymentMethod,
		"response_code":        data.ResponseCode,
		"order_ref":            data.OrderRef,
		"raw_data":             data.RawData,
		"content_type":         data.ContentType,
		"action":               data.Action,
		"active":               true,
		"date_created":         now.Unix(),
		"received_at":          now,
	}

	// Create document using common operations
	_, err := r.dbOps.Create(ctx, r.collectionName, doc)
	if err != nil {
		return nil, fmt.Errorf("failed to log webhook: %w", err)
	}

	return &paymentpb.LogWebhookResponse{
		Success: true,
		Id:      id,
	}, nil
}

// Compile-time interface check
var _ integrationPorts.IntegrationPaymentRepository = (*FirestoreIntegrationPaymentRepository)(nil)
