//go:build mysql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	invoiceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice_attribute"
	"google.golang.org/protobuf/encoding/protojson"
)

// MySQLInvoiceAttributeRepository implements invoice_attribute CRUD using MySQL 8.0+.
type MySQLInvoiceAttributeRepository struct {
	invoiceattributepb.UnimplementedInvoiceAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.InvoiceAttribute, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql invoice_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLInvoiceAttributeRepository(dbOps, tableName), nil
	})
}

// NewMySQLInvoiceAttributeRepository creates a new MySQL invoice_attribute repository.
func NewMySQLInvoiceAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) invoiceattributepb.InvoiceAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "invoice_attribute"
	}
	return &MySQLInvoiceAttributeRepository{dbOps: dbOps, tableName: tableName}
}

func (r *MySQLInvoiceAttributeRepository) CreateInvoiceAttribute(ctx context.Context, req *invoiceattributepb.CreateInvoiceAttributeRequest) (*invoiceattributepb.CreateInvoiceAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("invoice_attribute data is required")
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
		return nil, fmt.Errorf("failed to create invoice_attribute: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ia := &invoiceattributepb.InvoiceAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ia); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &invoiceattributepb.CreateInvoiceAttributeResponse{Data: []*invoiceattributepb.InvoiceAttribute{ia}}, nil
}

func (r *MySQLInvoiceAttributeRepository) ReadInvoiceAttribute(ctx context.Context, req *invoiceattributepb.ReadInvoiceAttributeRequest) (*invoiceattributepb.ReadInvoiceAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice_attribute ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read invoice_attribute: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ia := &invoiceattributepb.InvoiceAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ia); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &invoiceattributepb.ReadInvoiceAttributeResponse{Data: []*invoiceattributepb.InvoiceAttribute{ia}}, nil
}

func (r *MySQLInvoiceAttributeRepository) UpdateInvoiceAttribute(ctx context.Context, req *invoiceattributepb.UpdateInvoiceAttributeRequest) (*invoiceattributepb.UpdateInvoiceAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice_attribute ID is required")
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
		return nil, fmt.Errorf("failed to update invoice_attribute: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ia := &invoiceattributepb.InvoiceAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ia); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &invoiceattributepb.UpdateInvoiceAttributeResponse{Data: []*invoiceattributepb.InvoiceAttribute{ia}}, nil
}

func (r *MySQLInvoiceAttributeRepository) DeleteInvoiceAttribute(ctx context.Context, req *invoiceattributepb.DeleteInvoiceAttributeRequest) (*invoiceattributepb.DeleteInvoiceAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("invoice_attribute ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete invoice_attribute: %w", err)
	}
	return &invoiceattributepb.DeleteInvoiceAttributeResponse{Success: true}, nil
}

func (r *MySQLInvoiceAttributeRepository) ListInvoiceAttributes(ctx context.Context, req *invoiceattributepb.ListInvoiceAttributesRequest) (*invoiceattributepb.ListInvoiceAttributesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list invoice_attributes: %w", err)
	}
	var ias []*invoiceattributepb.InvoiceAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		ia := &invoiceattributepb.InvoiceAttribute{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ia); err != nil {
			continue
		}
		ias = append(ias, ia)
	}
	return &invoiceattributepb.ListInvoiceAttributesResponse{Data: ias, Success: true}, nil
}
