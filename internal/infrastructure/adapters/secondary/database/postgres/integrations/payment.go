//go:build postgres

package integrations

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/payment"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "integration_payment", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres integration_payment repository requires *sql.DB, got %T", conn)
		}
		return NewPostgresIntegrationPaymentRepository(db, tableName), nil
	})
}

// PostgresIntegrationPaymentRepository implements IntegrationPaymentRepository using PostgreSQL
type PostgresIntegrationPaymentRepository struct {
	db        *sql.DB
	tableName string
}

// NewPostgresIntegrationPaymentRepository creates a new Postgres integration payment repository
func NewPostgresIntegrationPaymentRepository(db *sql.DB, tableName string) *PostgresIntegrationPaymentRepository {
	if tableName == "" {
		tableName = "integration_payment"
	}
	return &PostgresIntegrationPaymentRepository{
		db:        db,
		tableName: tableName,
	}
}

// LogWebhook saves parsed webhook data to the integration_payment table
func (r *PostgresIntegrationPaymentRepository) LogWebhook(ctx context.Context, req *paymentpb.LogWebhookRequest) (*paymentpb.LogWebhookResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("log webhook data is required")
	}

	data := req.Data

	id := data.ExecutionId
	if id == "" {
		id = uuid.New().String()
	}

	now := time.Now()

	rawDataJSON, _ := json.Marshal(data.RawData)

	query := fmt.Sprintf(`INSERT INTO %s (
		id, payment_id, provider_id, provider_ref, provider_payment_ref,
		payment_status, amount, currency, payment_method, response_code,
		order_ref, raw_data, content_type, action, active, date_created, received_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`, r.tableName)

	_, err := r.db.ExecContext(ctx, query,
		id, data.PaymentId, data.ProviderId, data.ProviderRef, data.ProviderPaymentRef,
		data.PaymentStatus, data.Amount, data.Currency, data.PaymentMethod, data.ResponseCode,
		data.OrderRef, rawDataJSON, data.ContentType, data.Action, true, now.Unix(), now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to log webhook: %w", err)
	}

	return &paymentpb.LogWebhookResponse{
		Success: true,
		Id:      id,
	}, nil
}
