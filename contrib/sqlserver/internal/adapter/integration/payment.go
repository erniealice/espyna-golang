//go:build sqlserver

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
	registry.RegisterRepositoryFactory("sqlserver", entityid.IntegrationPayment, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver integration_payment repository requires *sql.DB, got %T", conn)
		}
		return NewSQLServerIntegrationPaymentRepository(db, tableName), nil
	})
}

// SQLServerIntegrationPaymentRepository implements IntegrationPaymentRepository using SQL Server.
//
// SQL Server dialect differences from the postgres gold standard:
//   - Placeholders: @p1, @p2, … (not $1, $2, …).
//   - No RETURNING clause — INSERT uses OUTPUT inserted.* if needed; here we
//     only need the app-supplied id back so no OUTPUT clause is required.
//   - active = 1 (BIT) instead of active = true.
//   - JSON column raw_data stored as NVARCHAR(MAX) with JSON string.
type SQLServerIntegrationPaymentRepository struct {
	db           *sql.DB
	tableName    string
	auditService infraports.AuditService
}

// NewSQLServerIntegrationPaymentRepository creates a new SQL Server integration payment repository.
func NewSQLServerIntegrationPaymentRepository(db *sql.DB, tableName string) *SQLServerIntegrationPaymentRepository {
	if tableName == "" {
		tableName = "integration_payment"
	}
	return &SQLServerIntegrationPaymentRepository{
		db:        db,
		tableName: tableName,
	}
}

// WithAuditService returns a copy of the repository with an audit service attached.
func (r *SQLServerIntegrationPaymentRepository) WithAuditService(svc infraports.AuditService) *SQLServerIntegrationPaymentRepository {
	r.auditService = svc
	return r
}

// LogWebhook saves parsed webhook data to the integration_payment table.
//
// SQL Server differences from the postgres gold standard:
//   - INSERT placeholders: @p1, @p2, … instead of $1, $2, ….
//   - active = 1 (BIT) instead of active = true.
//   - raw_data stored as NVARCHAR(MAX) JSON string — SQL Server lacks a native
//     JSON type; the column accepts the marshalled JSON string directly.
func (r *SQLServerIntegrationPaymentRepository) LogWebhook(ctx context.Context, req *paymentpb.LogWebhookRequest) (*paymentpb.LogWebhookResponse, error) {
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

	// SQL Server INSERT uses @p1, @p2, … placeholders.
	// active = 1 (SQL Server BIT) instead of active = true.
	query := fmt.Sprintf(`INSERT INTO %s (
		id, payment_id, provider_id, provider_ref, provider_payment_ref,
		payment_status, amount, currency, payment_method, response_code,
		order_ref, raw_data, content_type, action, active, date_created, received_at
	) VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9, @p10, @p11, @p12, @p13, @p14, 1, @p15, @p16)`, r.tableName)

	_, err := r.db.ExecContext(ctx, query,
		id, data.PaymentId, data.ProviderId, data.ProviderRef, data.ProviderPaymentRef,
		data.PaymentStatus, data.Amount, data.Currency, data.PaymentMethod, data.ResponseCode,
		data.OrderRef, string(rawDataJSON), data.ContentType, data.Action, now.Unix(), now,
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
