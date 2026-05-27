//go:build mysql

package tax

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	taxtreatmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_treatment"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.TaxTreatment, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql tax_treatment repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLTaxTreatmentRepository(dbOps, tableName), nil
	})
}

// MySQLTaxTreatmentRepository implements tax_treatment read operations using MySQL 8.0+.
type MySQLTaxTreatmentRepository struct {
	taxtreatmentpb.UnimplementedTaxTreatmentDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLTaxTreatmentRepository creates a new MySQL tax_treatment repository.
func NewMySQLTaxTreatmentRepository(dbOps interfaces.DatabaseOperation, tableName string) taxtreatmentpb.TaxTreatmentDomainServiceServer {
	if tableName == "" {
		tableName = entityid.TaxTreatment
	}
	return &MySQLTaxTreatmentRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func unmarshalTaxTreatment(raw map[string]any) (*taxtreatmentpb.TaxTreatment, error) {
	js, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal raw: %w", err)
	}
	t := &taxtreatmentpb.TaxTreatment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(js, t); err != nil {
		return nil, fmt.Errorf("unmarshal proto: %w", err)
	}
	return t, nil
}

// ReadTaxTreatment retrieves a tax_treatment record by ID.
func (r *MySQLTaxTreatmentRepository) ReadTaxTreatment(ctx context.Context, req *taxtreatmentpb.ReadTaxTreatmentRequest) (*taxtreatmentpb.ReadTaxTreatmentResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tax_treatment ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read tax_treatment: %w", err)
	}
	t, err := unmarshalTaxTreatment(result)
	if err != nil {
		return nil, err
	}
	return &taxtreatmentpb.ReadTaxTreatmentResponse{Success: true, Data: []*taxtreatmentpb.TaxTreatment{t}}, nil
}

// ListTaxTreatments lists all tax_treatment records.
func (r *MySQLTaxTreatmentRepository) ListTaxTreatments(ctx context.Context, req *taxtreatmentpb.ListTaxTreatmentsRequest) (*taxtreatmentpb.ListTaxTreatmentsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list tax_treatments: %w", err)
	}
	var items []*taxtreatmentpb.TaxTreatment
	for _, raw := range listResult.Data {
		t, err := unmarshalTaxTreatment(raw)
		if err != nil {
			log.Printf("WARN: unmarshal tax_treatment: %v", err)
			continue
		}
		items = append(items, t)
	}
	return &taxtreatmentpb.ListTaxTreatmentsResponse{Success: true, Data: items}, nil
}
