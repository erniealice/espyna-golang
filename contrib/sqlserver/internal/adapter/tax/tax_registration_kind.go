//go:build sqlserver

package tax

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	taxregistrationkindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration_kind"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.TaxRegistrationKind, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver tax_registration_kind repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerTaxRegistrationKindRepository(db, dbOps, tableName), nil
	})
}

// SQLServerTaxRegistrationKindRepository implements tax_registration_kind read operations.
type SQLServerTaxRegistrationKindRepository struct {
	taxregistrationkindpb.UnimplementedTaxRegistrationKindDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerTaxRegistrationKindRepository creates a new SQL Server tax_registration_kind repository.
func NewSQLServerTaxRegistrationKindRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) taxregistrationkindpb.TaxRegistrationKindDomainServiceServer {
	if tableName == "" {
		tableName = entityid.TaxRegistrationKind
	}
	return &SQLServerTaxRegistrationKindRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func unmarshalTaxRegistrationKind(raw map[string]any) (*taxregistrationkindpb.TaxRegistrationKind, error) {
	js, err := json.Marshal(sqlserverCore.DenormalizeKeys(raw))
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
func (r *SQLServerTaxRegistrationKindRepository) ReadTaxRegistrationKind(ctx context.Context, req *taxregistrationkindpb.ReadTaxRegistrationKindRequest) (*taxregistrationkindpb.ReadTaxRegistrationKindResponse, error) {
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
func (r *SQLServerTaxRegistrationKindRepository) ListTaxRegistrationKinds(ctx context.Context, req *taxregistrationkindpb.ListTaxRegistrationKindsRequest) (*taxregistrationkindpb.ListTaxRegistrationKindsResponse, error) {
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
// SQL Server differences from the postgres gold standard:
//   - $1 → @p1.
//   - active = true → active = 1 (SQL Server BIT).
//   - Postgres ANY(array_col) → SQL Server uses CHARINDEX or a JSON/string pattern.
//     applicable_party_types is stored as a delimited string or JSON on SQL Server
//     (not a native array). We use CHARINDEX to check membership in a comma-separated
//     string value. If stored as JSON ARRAY, use JSON_VALUE / OPENJSON instead.
//   - No row_to_json() — explicit column scan + DenormalizeKeys unmarshal path.
//   - LIMIT → not needed here (ORDER BY only); TOP N would be used if needed.
func (r *SQLServerTaxRegistrationKindRepository) FindByPartyType(ctx context.Context, partyType string) ([]*taxregistrationkindpb.TaxRegistrationKind, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindByPartyType requires raw *sql.DB")
	}
	// applicable_party_types is assumed to be a comma-separated string on SQL Server
	// (e.g. "CLIENT,WORKSPACE"). CHARINDEX checks whether the party-type token
	// appears in the stored value. This mirrors the postgres ANY(applicable_party_types)
	// semantic. If the column is actually stored as JSON, replace with OPENJSON.
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, code, name, description, jurisdiction,
		        applicable_party_types, active
		 FROM tax_registration_kind
		 WHERE active = 1
		   AND CHARINDEX(@p1, applicable_party_types) > 0
		 ORDER BY name`,
		partyType,
	)
	if err != nil {
		return nil, fmt.Errorf("FindByPartyType query failed: %w", err)
	}
	defer rows.Close()

	var items []*taxregistrationkindpb.TaxRegistrationKind
	for rows.Next() {
		var (
			id                   string
			code                 string
			name                 string
			description          *string
			jurisdiction         *string
			applicablePartyTypes *string
			active               bool
		)
		if err := rows.Scan(&id, &code, &name, &description, &jurisdiction, &applicablePartyTypes, &active); err != nil {
			return nil, fmt.Errorf("FindByPartyType scan: %w", err)
		}
		k := &taxregistrationkindpb.TaxRegistrationKind{
			Id:     id,
			Code:   code,
			Name:   name,
			Active: active,
		}
		if description != nil {
			k.Description = description
		}
		if jurisdiction != nil {
			k.Jurisdiction = *jurisdiction
		}
		items = append(items, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("FindByPartyType rows error: %w", err)
	}
	return items, nil
}

var _ FindByPartyTypeQueries = (*SQLServerTaxRegistrationKindRepository)(nil)
