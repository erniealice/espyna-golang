package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/payment"
	"github.com/google/uuid"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.IntegrationPayment, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres integration_payment repository requires *sql.DB, got %T", conn)
		}
		return NewPostgresIntegrationPaymentRepository(db, tableName), nil
	})
}

// PostgresIntegrationPaymentRepository implements IntegrationPaymentRepository using PostgreSQL
type PostgresIntegrationPaymentRepository struct {
	db           *sql.DB
	tableName    string
	auditService infraports.AuditService
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

// WithAuditService returns a copy of the repository with an audit service attached.
func (r *PostgresIntegrationPaymentRepository) WithAuditService(svc infraports.AuditService) *PostgresIntegrationPaymentRepository {
	r.auditService = svc
	return r
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

	if r.auditService != nil {
		_ = infraports.DiffAndLog(ctx, r.auditService, infraports.DiffAndLogRequest{
			EntityType:     "integration_payment",
			EntityID:       id,
			Domain:         "centymo",
			Action:         1, // INSERT
			PermissionCode: "payment:create",
			UseCase:        "LogWebhook",
			MethodName:     "LogWebhook",
			NewData: map[string]any{
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
				"content_type":         data.ContentType,
				"action":               data.Action,
			},
		})
	}

	return &paymentpb.LogWebhookResponse{
		Success: true,
		Id:      id,
	}, nil
}
