//go:build postgresql

package treasury

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	withholdingcertificatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/withholding_certificate"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.WithholdingCertificate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres withholding_certificate repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresWithholdingCertificateRepository(dbOps, tableName), nil
	})
}

// PostgresWithholdingCertificateRepository implements withholding_certificate CRUD using PostgreSQL.
type PostgresWithholdingCertificateRepository struct {
	withholdingcertificatepb.UnimplementedWithholdingCertificateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresWithholdingCertificateRepository creates a new PostgreSQL withholding_certificate repository.
func NewPostgresWithholdingCertificateRepository(dbOps interfaces.DatabaseOperation, tableName string) withholdingcertificatepb.WithholdingCertificateDomainServiceServer {
	if tableName == "" {
		tableName = entityid.WithholdingCertificate
	}
	return &PostgresWithholdingCertificateRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func unmarshalWithholdingCertificate(raw map[string]any) (*withholdingcertificatepb.WithholdingCertificate, error) {
	js, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal raw: %w", err)
	}
	c := &withholdingcertificatepb.WithholdingCertificate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(js, c); err != nil {
		return nil, fmt.Errorf("unmarshal proto: %w", err)
	}
	return c, nil
}

// CreateWithholdingCertificate creates a new withholding_certificate record.
func (r *PostgresWithholdingCertificateRepository) CreateWithholdingCertificate(ctx context.Context, req *withholdingcertificatepb.CreateWithholdingCertificateRequest) (*withholdingcertificatepb.CreateWithholdingCertificateResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("withholding_certificate data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("marshal proto: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal to map: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create withholding_certificate: %w", err)
	}
	cert, err := unmarshalWithholdingCertificate(result)
	if err != nil {
		return nil, err
	}
	return &withholdingcertificatepb.CreateWithholdingCertificateResponse{Success: true, Data: []*withholdingcertificatepb.WithholdingCertificate{cert}}, nil
}

// ReadWithholdingCertificate retrieves a withholding_certificate record by ID.
func (r *PostgresWithholdingCertificateRepository) ReadWithholdingCertificate(ctx context.Context, req *withholdingcertificatepb.ReadWithholdingCertificateRequest) (*withholdingcertificatepb.ReadWithholdingCertificateResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("withholding_certificate ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read withholding_certificate: %w", err)
	}
	cert, err := unmarshalWithholdingCertificate(result)
	if err != nil {
		return nil, err
	}
	return &withholdingcertificatepb.ReadWithholdingCertificateResponse{Success: true, Data: []*withholdingcertificatepb.WithholdingCertificate{cert}}, nil
}

// UpdateWithholdingCertificate updates a withholding_certificate record.
func (r *PostgresWithholdingCertificateRepository) UpdateWithholdingCertificate(ctx context.Context, req *withholdingcertificatepb.UpdateWithholdingCertificateRequest) (*withholdingcertificatepb.UpdateWithholdingCertificateResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("withholding_certificate ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("marshal proto: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal to map: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update withholding_certificate: %w", err)
	}
	cert, err := unmarshalWithholdingCertificate(result)
	if err != nil {
		return nil, err
	}
	return &withholdingcertificatepb.UpdateWithholdingCertificateResponse{Success: true, Data: []*withholdingcertificatepb.WithholdingCertificate{cert}}, nil
}

// DeleteWithholdingCertificate soft-deletes a withholding_certificate record.
func (r *PostgresWithholdingCertificateRepository) DeleteWithholdingCertificate(ctx context.Context, req *withholdingcertificatepb.DeleteWithholdingCertificateRequest) (*withholdingcertificatepb.DeleteWithholdingCertificateResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("withholding_certificate ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete withholding_certificate: %w", err)
	}
	return &withholdingcertificatepb.DeleteWithholdingCertificateResponse{Success: true}, nil
}

// ListWithholdingCertificates lists withholding_certificate records.
func (r *PostgresWithholdingCertificateRepository) ListWithholdingCertificates(ctx context.Context, req *withholdingcertificatepb.ListWithholdingCertificatesRequest) (*withholdingcertificatepb.ListWithholdingCertificatesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list withholding_certificates: %w", err)
	}
	var items []*withholdingcertificatepb.WithholdingCertificate
	for _, raw := range listResult.Data {
		cert, err := unmarshalWithholdingCertificate(raw)
		if err != nil {
			log.Printf("WARN: unmarshal withholding_certificate: %v", err)
			continue
		}
		items = append(items, cert)
	}
	return &withholdingcertificatepb.ListWithholdingCertificatesResponse{Success: true, Data: items}, nil
}
