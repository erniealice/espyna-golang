//go:build sqlserver

package tenancy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	tenantsubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tenancy/tenant_subscription"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.TenantSubscription, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver tenant_subscription repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerTenantSubscriptionRepository(dbOps, tableName), nil
	})
}

// SQLServerTenantSubscriptionRepository implements tenant subscription CRUD operations using SQL Server.
type SQLServerTenantSubscriptionRepository struct {
	tenantsubscriptionpb.UnimplementedTenantSubscriptionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerTenantSubscriptionRepository creates a new SQL Server tenant_subscription repository.
func NewSQLServerTenantSubscriptionRepository(dbOps interfaces.DatabaseOperation, tableName string) tenantsubscriptionpb.TenantSubscriptionDomainServiceServer {
	if tableName == "" {
		tableName = "tenant_subscription"
	}
	return &SQLServerTenantSubscriptionRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateTenantSubscription creates a new tenant subscription record.
func (r *SQLServerTenantSubscriptionRepository) CreateTenantSubscription(ctx context.Context, req *tenantsubscriptionpb.CreateTenantSubscriptionRequest) (*tenantsubscriptionpb.CreateTenantSubscriptionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("tenant_subscription data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant_subscription: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	sub := &tenantsubscriptionpb.TenantSubscription{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sub); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &tenantsubscriptionpb.CreateTenantSubscriptionResponse{Data: []*tenantsubscriptionpb.TenantSubscription{sub}}, nil
}

// ReadTenantSubscription retrieves a tenant subscription by ID.
func (r *SQLServerTenantSubscriptionRepository) ReadTenantSubscription(ctx context.Context, req *tenantsubscriptionpb.ReadTenantSubscriptionRequest) (*tenantsubscriptionpb.ReadTenantSubscriptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tenant_subscription ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read tenant_subscription: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	sub := &tenantsubscriptionpb.TenantSubscription{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sub); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &tenantsubscriptionpb.ReadTenantSubscriptionResponse{Data: []*tenantsubscriptionpb.TenantSubscription{sub}}, nil
}

// UpdateTenantSubscription updates an existing tenant subscription record.
func (r *SQLServerTenantSubscriptionRepository) UpdateTenantSubscription(ctx context.Context, req *tenantsubscriptionpb.UpdateTenantSubscriptionRequest) (*tenantsubscriptionpb.UpdateTenantSubscriptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tenant_subscription ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update tenant_subscription: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	sub := &tenantsubscriptionpb.TenantSubscription{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sub); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &tenantsubscriptionpb.UpdateTenantSubscriptionResponse{Data: []*tenantsubscriptionpb.TenantSubscription{sub}}, nil
}

// DeleteTenantSubscription soft-deletes a tenant subscription.
func (r *SQLServerTenantSubscriptionRepository) DeleteTenantSubscription(ctx context.Context, req *tenantsubscriptionpb.DeleteTenantSubscriptionRequest) (*tenantsubscriptionpb.DeleteTenantSubscriptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tenant_subscription ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete tenant_subscription: %w", err)
	}
	return &tenantsubscriptionpb.DeleteTenantSubscriptionResponse{Success: true}, nil
}

// ListTenantSubscriptions lists tenant subscriptions matching optional filters.
func (r *SQLServerTenantSubscriptionRepository) ListTenantSubscriptions(ctx context.Context, req *tenantsubscriptionpb.ListTenantSubscriptionsRequest) (*tenantsubscriptionpb.ListTenantSubscriptionsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant_subscriptions: %w", err)
	}
	var subs []*tenantsubscriptionpb.TenantSubscription
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		sub := &tenantsubscriptionpb.TenantSubscription{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sub); err != nil {
			continue
		}
		subs = append(subs, sub)
	}
	return &tenantsubscriptionpb.ListTenantSubscriptionsResponse{Data: subs}, nil
}
