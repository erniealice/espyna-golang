//go:build sqlserver

package tenancy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	tenantinvoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tenancy/tenant_invoice"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.TenantInvoice, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver tenant_invoice repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerTenantInvoiceRepository(dbOps, tableName), nil
	})
}

// SQLServerTenantInvoiceRepository implements tenant invoice CRUD operations using SQL Server.
// SQL dialect differences from the postgres gold standard are encapsulated in
// sqlserverCore.WorkspaceAwareOperations (placeholders @pN, [bracket] quoting,
// OUTPUT inserted.* instead of RETURNING, OFFSET/FETCH pagination).
type SQLServerTenantInvoiceRepository struct {
	tenantinvoicepb.UnimplementedTenantInvoiceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerTenantInvoiceRepository creates a new SQL Server tenant_invoice repository.
func NewSQLServerTenantInvoiceRepository(dbOps interfaces.DatabaseOperation, tableName string) tenantinvoicepb.TenantInvoiceDomainServiceServer {
	if tableName == "" {
		tableName = "tenant_invoice"
	}
	return &SQLServerTenantInvoiceRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateTenantInvoice creates a new tenant invoice record.
func (r *SQLServerTenantInvoiceRepository) CreateTenantInvoice(ctx context.Context, req *tenantinvoicepb.CreateTenantInvoiceRequest) (*tenantinvoicepb.CreateTenantInvoiceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("tenant_invoice data is required")
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
		return nil, fmt.Errorf("failed to create tenant_invoice: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	inv := &tenantinvoicepb.TenantInvoice{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &tenantinvoicepb.CreateTenantInvoiceResponse{Data: []*tenantinvoicepb.TenantInvoice{inv}}, nil
}

// ReadTenantInvoice retrieves a tenant invoice by ID.
func (r *SQLServerTenantInvoiceRepository) ReadTenantInvoice(ctx context.Context, req *tenantinvoicepb.ReadTenantInvoiceRequest) (*tenantinvoicepb.ReadTenantInvoiceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tenant_invoice ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read tenant_invoice: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	inv := &tenantinvoicepb.TenantInvoice{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &tenantinvoicepb.ReadTenantInvoiceResponse{Data: []*tenantinvoicepb.TenantInvoice{inv}}, nil
}

// UpdateTenantInvoice updates an existing tenant invoice record.
func (r *SQLServerTenantInvoiceRepository) UpdateTenantInvoice(ctx context.Context, req *tenantinvoicepb.UpdateTenantInvoiceRequest) (*tenantinvoicepb.UpdateTenantInvoiceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tenant_invoice ID is required")
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
		return nil, fmt.Errorf("failed to update tenant_invoice: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	inv := &tenantinvoicepb.TenantInvoice{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &tenantinvoicepb.UpdateTenantInvoiceResponse{Data: []*tenantinvoicepb.TenantInvoice{inv}}, nil
}

// DeleteTenantInvoice soft-deletes a tenant invoice.
func (r *SQLServerTenantInvoiceRepository) DeleteTenantInvoice(ctx context.Context, req *tenantinvoicepb.DeleteTenantInvoiceRequest) (*tenantinvoicepb.DeleteTenantInvoiceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tenant_invoice ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete tenant_invoice: %w", err)
	}
	return &tenantinvoicepb.DeleteTenantInvoiceResponse{Success: true}, nil
}

// ListTenantInvoices lists tenant invoices matching optional filters.
func (r *SQLServerTenantInvoiceRepository) ListTenantInvoices(ctx context.Context, req *tenantinvoicepb.ListTenantInvoicesRequest) (*tenantinvoicepb.ListTenantInvoicesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant_invoices: %w", err)
	}
	var invoices []*tenantinvoicepb.TenantInvoice
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		inv := &tenantinvoicepb.TenantInvoice{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inv); err != nil {
			continue
		}
		invoices = append(invoices, inv)
	}
	return &tenantinvoicepb.ListTenantInvoicesResponse{Data: invoices}, nil
}
