//go:build mysql

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
	registry.RegisterRepositoryFactory("mysql", entityid.IntegrationPayment, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql integration_payment repository requires *sql.DB, got %T", conn)
		}
		return NewMySQLIntegrationPaymentRepository(db, tableName), nil
	})
}

// MySQLIntegrationPaymentRepository implements IntegrationPaymentRepository using MySQL 8.0+.
//
// Dialect translation from postgres gold standard:
//   - $1,$2,... → ? (MySQL positional placeholders, args in same left-to-right order)
//   - "ident" → `ident` (backtick-quoted identifiers)
//   - INSERT ... RETURNING → app-supplied UUID + SELECT after insert (no RETURNING in MySQL)
type MySQLIntegrationPaymentRepository struct {
	db           *sql.DB
	tableName    string
	auditService infraports.AuditService
}

// NewMySQLIntegrationPaymentRepository creates a new MySQL integration payment repository.
func NewMySQLIntegrationPaymentRepository(db *sql.DB, tableName string) *MySQLIntegrationPaymentRepository {
	if tableName == "" {
		tableName = "integration_payment"
	}
	return &MySQLIntegrationPaymentRepository{
		db:        db,
		tableName: tableName,
	}
}

// WithAuditService returns a copy of the repository with an audit service attached.
func (r *MySQLIntegrationPaymentRepository) WithAuditService(svc infraports.AuditService) *MySQLIntegrationPaymentRepository {
	r.auditService = svc
	return r
}

// LogWebhook saves parsed webhook data to the integration_payment table.
//
// Dialect translation from postgres gold standard:
//   - $1..$17 → ? (positional, same left-to-right order)
//   - Backtick-quoted table name via fmt.Sprintf
//   - No RETURNING — id is generated app-side before INSERT
func (r *MySQLIntegrationPaymentRepository) LogWebhook(ctx context.Context, req *paymentpb.LogWebhookRequest) (*paymentpb.LogWebhookResponse, error) {
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

	// Dialect: $1..$17 → ? (positional), backtick-quoted table name.
	query := fmt.Sprintf("INSERT INTO `%s` ("+
		"`id`, `payment_id`, `provider_id`, `provider_ref`, `provider_payment_ref`,"+
		"`payment_status`, `amount`, `currency`, `payment_method`, `response_code`,"+
		"`order_ref`, `raw_data`, `content_type`, `action`, `active`, `date_created`, `received_at`"+
		") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", r.tableName)

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
