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
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	taxregistrationkindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration_kind"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.TaxRegistrationKind, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql tax_registration_kind repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLTaxRegistrationKindRepository(db, dbOps, tableName), nil
	})
}

// MySQLTaxRegistrationKindRepository implements tax_registration_kind read operations using MySQL 8.0+.
type MySQLTaxRegistrationKindRepository struct {
	taxregistrationkindpb.UnimplementedTaxRegistrationKindDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLTaxRegistrationKindRepository creates a new MySQL tax_registration_kind repository.
func NewMySQLTaxRegistrationKindRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) taxregistrationkindpb.TaxRegistrationKindDomainServiceServer {
	if tableName == "" {
		tableName = entityid.TaxRegistrationKind
	}
	return &MySQLTaxRegistrationKindRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func unmarshalTaxRegistrationKind(raw map[string]any) (*taxregistrationkindpb.TaxRegistrationKind, error) {
	js, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal raw: %w", err)
	}
	k := &taxregistrationkindpb.TaxRegistrationKind{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(js, k); err != nil {
		return nil, fmt.Errorf("unmarshal proto: %w", err)
	}
	return k, nil
}

// ReadTaxRegistrationKind retrieves a tax_registration_kind record by ID.
func (r *MySQLTaxRegistrationKindRepository) ReadTaxRegistrationKind(ctx context.Context, req *taxregistrationkindpb.ReadTaxRegistrationKindRequest) (*taxregistrationkindpb.ReadTaxRegistrationKindResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tax_registration_kind ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read tax_registration_kind: %w", err)
	}
	k, err := unmarshalTaxRegistrationKind(result)
	if err != nil {
		return nil, err
	}
	return &taxregistrationkindpb.ReadTaxRegistrationKindResponse{Success: true, Data: []*taxregistrationkindpb.TaxRegistrationKind{k}}, nil
}

// ListTaxRegistrationKinds lists all tax_registration_kind records.
func (r *MySQLTaxRegistrationKindRepository) ListTaxRegistrationKinds(ctx context.Context, req *taxregistrationkindpb.ListTaxRegistrationKindsRequest) (*taxregistrationkindpb.ListTaxRegistrationKindsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list tax_registration_kinds: %w", err)
	}
	var items []*taxregistrationkindpb.TaxRegistrationKind
	for _, raw := range listResult.Data {
		k, err := unmarshalTaxRegistrationKind(raw)
		if err != nil {
			log.Printf("WARN: unmarshal tax_registration_kind: %v", err)
			continue
		}
		items = append(items, k)
	}
	return &taxregistrationkindpb.ListTaxRegistrationKindsResponse{Success: true, Data: items}, nil
}

// FindByPartyTypeQueries is the interface consumed by FindByPartyType.
type FindByPartyTypeQueries interface {
	FindByPartyType(ctx context.Context, partyType string) ([]*taxregistrationkindpb.TaxRegistrationKind, error)
}

// FindByPartyType returns kinds where applicable_party_types contains partyType.
//
// Dialect changes from postgres gold standard:
//   - $1 = ANY(applicable_party_types) → JSON_CONTAINS(applicable_party_types, JSON_QUOTE(?), '$')
//     MySQL stores the array as JSON; JSON_CONTAINS checks element membership.
//   - active = true → active = 1 (MySQL TINYINT boolean)
//   - ? placeholder replaces $1.
func (r *MySQLTaxRegistrationKindRepository) FindByPartyType(ctx context.Context, partyType string) ([]*taxregistrationkindpb.TaxRegistrationKind, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindByPartyType requires raw *sql.DB")
	}
	// Dialect: JSON_CONTAINS replaces postgres ANY(array); active = 1 for TINYINT boolean.
	// applicable_party_types is stored as a JSON array in MySQL (e.g. ["CLIENT","WORKSPACE"]).
	rows, err := r.db.QueryContext(ctx,
		`SELECT k.id FROM tax_registration_kind k
		 WHERE active = 1
		   AND JSON_CONTAINS(k.applicable_party_types, JSON_QUOTE(?), '$')
		 ORDER BY k.name`,
		partyType,
	)
	if err != nil {
		return nil, fmt.Errorf("FindByPartyType query failed: %w", err)
	}
	defer rows.Close()

	var items []*taxregistrationkindpb.TaxRegistrationKind
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("FindByPartyType scan: %w", err)
		}
		raw, err := r.dbOps.Read(ctx, r.tableName, id)
		if err != nil {
			log.Printf("WARN: FindByPartyType read id=%s: %v", id, err)
			continue
		}
		k, err := unmarshalTaxRegistrationKind(raw)
		if err != nil {
			log.Printf("WARN: FindByPartyType unmarshal id=%s: %v", id, err)
			continue
		}
		items = append(items, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("FindByPartyType rows error: %w", err)
	}
	return items, nil
}

var _ FindByPartyTypeQueries = (*MySQLTaxRegistrationKindRepository)(nil)
