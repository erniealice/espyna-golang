//go:build mysql

package tenancy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	tenantpaymentmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tenancy/tenant_payment_method"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.TenantPaymentMethod, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql tenant_payment_method repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLTenantPaymentMethodRepository(dbOps, tableName), nil
	})
}

// MySQLTenantPaymentMethodRepository implements tenant payment method CRUD operations using MySQL 8.0+.
type MySQLTenantPaymentMethodRepository struct {
	tenantpaymentmethodpb.UnimplementedTenantPaymentMethodDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLTenantPaymentMethodRepository creates a new MySQL tenant_payment_method repository.
func NewMySQLTenantPaymentMethodRepository(dbOps interfaces.DatabaseOperation, tableName string) tenantpaymentmethodpb.TenantPaymentMethodDomainServiceServer {
	if tableName == "" {
		tableName = "tenant_payment_method"
	}
	return &MySQLTenantPaymentMethodRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateTenantPaymentMethod creates a new tenant payment method record.
func (r *MySQLTenantPaymentMethodRepository) CreateTenantPaymentMethod(ctx context.Context, req *tenantpaymentmethodpb.CreateTenantPaymentMethodRequest) (*tenantpaymentmethodpb.CreateTenantPaymentMethodResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("tenant_payment_method data is required")
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
		return nil, fmt.Errorf("failed to create tenant_payment_method: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pm := &tenantpaymentmethodpb.TenantPaymentMethod{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &tenantpaymentmethodpb.CreateTenantPaymentMethodResponse{Data: []*tenantpaymentmethodpb.TenantPaymentMethod{pm}}, nil
}

// ReadTenantPaymentMethod retrieves a tenant payment method by ID.
func (r *MySQLTenantPaymentMethodRepository) ReadTenantPaymentMethod(ctx context.Context, req *tenantpaymentmethodpb.ReadTenantPaymentMethodRequest) (*tenantpaymentmethodpb.ReadTenantPaymentMethodResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tenant_payment_method ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read tenant_payment_method: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pm := &tenantpaymentmethodpb.TenantPaymentMethod{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &tenantpaymentmethodpb.ReadTenantPaymentMethodResponse{Data: []*tenantpaymentmethodpb.TenantPaymentMethod{pm}}, nil
}

// UpdateTenantPaymentMethod updates an existing tenant payment method record.
func (r *MySQLTenantPaymentMethodRepository) UpdateTenantPaymentMethod(ctx context.Context, req *tenantpaymentmethodpb.UpdateTenantPaymentMethodRequest) (*tenantpaymentmethodpb.UpdateTenantPaymentMethodResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tenant_payment_method ID is required")
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
		return nil, fmt.Errorf("failed to update tenant_payment_method: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pm := &tenantpaymentmethodpb.TenantPaymentMethod{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &tenantpaymentmethodpb.UpdateTenantPaymentMethodResponse{Data: []*tenantpaymentmethodpb.TenantPaymentMethod{pm}}, nil
}

// DeleteTenantPaymentMethod soft-deletes a tenant payment method.
func (r *MySQLTenantPaymentMethodRepository) DeleteTenantPaymentMethod(ctx context.Context, req *tenantpaymentmethodpb.DeleteTenantPaymentMethodRequest) (*tenantpaymentmethodpb.DeleteTenantPaymentMethodResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tenant_payment_method ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete tenant_payment_method: %w", err)
	}
	return &tenantpaymentmethodpb.DeleteTenantPaymentMethodResponse{Success: true}, nil
}

// ListTenantPaymentMethods lists tenant payment methods matching optional filters.
func (r *MySQLTenantPaymentMethodRepository) ListTenantPaymentMethods(ctx context.Context, req *tenantpaymentmethodpb.ListTenantPaymentMethodsRequest) (*tenantpaymentmethodpb.ListTenantPaymentMethodsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant_payment_methods: %w", err)
	}
	var methods []*tenantpaymentmethodpb.TenantPaymentMethod
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		pm := &tenantpaymentmethodpb.TenantPaymentMethod{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pm); err != nil {
			continue
		}
		methods = append(methods, pm)
	}
	return &tenantpaymentmethodpb.ListTenantPaymentMethodsResponse{Data: methods}, nil
}
