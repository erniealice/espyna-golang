//go:build postgresql

package tax

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
	taxregistrationkindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration_kind"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.TaxRegistrationKind, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres tax_registration_kind repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresTaxRegistrationKindRepository(db, dbOps, tableName), nil
	})
}

// PostgresTaxRegistrationKindRepository implements tax_registration_kind read operations.
type PostgresTaxRegistrationKindRepository struct {
	taxregistrationkindpb.UnimplementedTaxRegistrationKindDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresTaxRegistrationKindRepository creates a new PostgreSQL tax_registration_kind repository.
func NewPostgresTaxRegistrationKindRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) taxregistrationkindpb.TaxRegistrationKindDomainServiceServer {
	if tableName == "" {
		tableName = entityid.TaxRegistrationKind
	}
	return &PostgresTaxRegistrationKindRepository{
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
func (r *PostgresTaxRegistrationKindRepository) ReadTaxRegistrationKind(ctx context.Context, req *taxregistrationkindpb.ReadTaxRegistrationKindRequest) (*taxregistrationkindpb.ReadTaxRegistrationKindResponse, error) {
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
func (r *PostgresTaxRegistrationKindRepository) ListTaxRegistrationKinds(ctx context.Context, req *taxregistrationkindpb.ListTaxRegistrationKindsRequest) (*taxregistrationkindpb.ListTaxRegistrationKindsResponse, error) {
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
// Used by the tax registration drawer Kind dropdown to show only party-applicable kinds.
// The partyType argument is the string representation stored in applicable_party_types
// (e.g. "CLIENT", "WORKSPACE").
func (r *PostgresTaxRegistrationKindRepository) FindByPartyType(ctx context.Context, partyType string) ([]*taxregistrationkindpb.TaxRegistrationKind, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindByPartyType requires raw *sql.DB")
	}
	// applicable_party_types is stored as a text[] column; each element is the
	// upper-case party-type string (e.g. "CLIENT", "WORKSPACE").
	rows, err := r.db.QueryContext(ctx,
		`SELECT row_to_json(k) FROM tax_registration_kind k
		 WHERE active = true AND $1 = ANY(applicable_party_types)
		 ORDER BY name`,
		partyType,
	)
	if err != nil {
		return nil, fmt.Errorf("FindByPartyType query failed: %w", err)
	}
	defer rows.Close()

	var items []*taxregistrationkindpb.TaxRegistrationKind
	for rows.Next() {
		var rawJSON []byte
		if err := rows.Scan(&rawJSON); err != nil {
			return nil, fmt.Errorf("FindByPartyType scan: %w", err)
		}
		k := &taxregistrationkindpb.TaxRegistrationKind{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawJSON, k); err != nil {
			log.Printf("WARN: FindByPartyType unmarshal: %v", err)
			continue
		}
		items = append(items, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("FindByPartyType rows error: %w", err)
	}
	return items, nil
}

// Ensure PostgresTaxRegistrationKindRepository satisfies FindByPartyTypeQueries.
var _ FindByPartyTypeQueries = (*PostgresTaxRegistrationKindRepository)(nil)
