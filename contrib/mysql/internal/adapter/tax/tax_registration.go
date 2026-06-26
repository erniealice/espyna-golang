//go:build mysql

package tax

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	taxregistrationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.TaxRegistration, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql tax_registration repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLTaxRegistrationRepository(db, dbOps, tableName), nil
	})
}

// MySQLTaxRegistrationRepository implements tax_registration CRUD using MySQL 8.0+.
// NOTE: "Update" means Supersede — a new immutable row is inserted and the prior
// ACTIVE row is marked SUPERSEDED with effective_to set. "Delete" means Revoke.
type MySQLTaxRegistrationRepository struct {
	taxregistrationpb.UnimplementedTaxRegistrationDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLTaxRegistrationRepository creates a new MySQL tax_registration repository.
func NewMySQLTaxRegistrationRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) taxregistrationpb.TaxRegistrationDomainServiceServer {
	if tableName == "" {
		tableName = entityid.TaxRegistration
	}
	return &MySQLTaxRegistrationRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func unmarshalTaxRegistration(raw map[string]any) (*taxregistrationpb.TaxRegistration, error) {
	js, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal raw: %w", err)
	}
	r := &taxregistrationpb.TaxRegistration{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(js, r); err != nil {
		return nil, fmt.Errorf("unmarshal proto: %w", err)
	}
	return r, nil
}

// CreateTaxRegistration inserts a new tax_registration row.
func (r *MySQLTaxRegistrationRepository) CreateTaxRegistration(ctx context.Context, req *taxregistrationpb.CreateTaxRegistrationRequest) (*taxregistrationpb.CreateTaxRegistrationResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("tax_registration data is required")
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
		return nil, fmt.Errorf("failed to create tax_registration: %w", err)
	}
	reg, err := unmarshalTaxRegistration(result)
	if err != nil {
		return nil, err
	}
	return &taxregistrationpb.CreateTaxRegistrationResponse{Success: true, Data: []*taxregistrationpb.TaxRegistration{reg}}, nil
}

// ReadTaxRegistration retrieves a tax_registration record by ID.
func (r *MySQLTaxRegistrationRepository) ReadTaxRegistration(ctx context.Context, req *taxregistrationpb.ReadTaxRegistrationRequest) (*taxregistrationpb.ReadTaxRegistrationResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tax_registration ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read tax_registration: %w", err)
	}
	reg, err := unmarshalTaxRegistration(result)
	if err != nil {
		return nil, err
	}
	return &taxregistrationpb.ReadTaxRegistrationResponse{Success: true, Data: []*taxregistrationpb.TaxRegistration{reg}}, nil
}

// UpdateTaxRegistration stores a direct update to a tax_registration record.
func (r *MySQLTaxRegistrationRepository) UpdateTaxRegistration(ctx context.Context, req *taxregistrationpb.UpdateTaxRegistrationRequest) (*taxregistrationpb.UpdateTaxRegistrationResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tax_registration ID is required")
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
		return nil, fmt.Errorf("failed to update tax_registration: %w", err)
	}
	reg, err := unmarshalTaxRegistration(result)
	if err != nil {
		return nil, err
	}
	return &taxregistrationpb.UpdateTaxRegistrationResponse{Success: true, Data: []*taxregistrationpb.TaxRegistration{reg}}, nil
}

// DeleteTaxRegistration soft-deletes a tax_registration record.
func (r *MySQLTaxRegistrationRepository) DeleteTaxRegistration(ctx context.Context, req *taxregistrationpb.DeleteTaxRegistrationRequest) (*taxregistrationpb.DeleteTaxRegistrationResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tax_registration ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete tax_registration: %w", err)
	}
	return &taxregistrationpb.DeleteTaxRegistrationResponse{Success: true}, nil
}

// ListTaxRegistrations lists tax_registration records.
func (r *MySQLTaxRegistrationRepository) ListTaxRegistrations(ctx context.Context, req *taxregistrationpb.ListTaxRegistrationsRequest) (*taxregistrationpb.ListTaxRegistrationsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list tax_registrations: %w", err)
	}
	var items []*taxregistrationpb.TaxRegistration
	for _, raw := range listResult.Data {
		reg, err := unmarshalTaxRegistration(raw)
		if err != nil {
			log.Printf("WARN: unmarshal tax_registration: %v", err)
			continue
		}
		items = append(items, reg)
	}
	return &taxregistrationpb.ListTaxRegistrationsResponse{Success: true, Data: items}, nil
}

// TaxRegistrationQueries provides additional query methods beyond the proto interface.
type TaxRegistrationQueries interface {
	FindActive(ctx context.Context, partyType, partyID string, asOf time.Time) ([]*taxregistrationpb.TaxRegistration, error)
	FindActiveByComputePath(ctx context.Context, partyType, partyID, computePath, jurisdiction string, asOf time.Time) (*taxregistrationpb.TaxRegistration, error)
}

// FindActive returns all ACTIVE tax_registration rows for the given party at asOf.
//
// Dialect changes from postgres gold standard:
//   - row_to_json(tr) → id-only scan + dbOps.Read per row
//   - $1..$3 → ? (positional, re-sequenced)
//   - tr.status IN (2, 3, 4) stays identical
func (r *MySQLTaxRegistrationRepository) FindActive(ctx context.Context, partyType, partyID string, asOf time.Time) ([]*taxregistrationpb.TaxRegistration, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindActive requires raw *sql.DB")
	}
	// Dialect: ? placeholders; no casts needed in MySQL.
	rows, err := r.db.QueryContext(ctx,
		`SELECT tr.id FROM tax_registration tr
		 WHERE tr.party_type = ?
		   AND tr.party_id = ?
		   AND tr.status IN (2, 3, 4)
		   AND tr.effective_from <= ?
		   AND (tr.effective_to IS NULL OR tr.effective_to > ?)
		 ORDER BY tr.effective_from DESC`,
		partyType, partyID, asOf, asOf,
	)
	if err != nil {
		return nil, fmt.Errorf("FindActive query: %w", err)
	}
	defer rows.Close()

	var items []*taxregistrationpb.TaxRegistration
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("FindActive scan: %w", err)
		}
		raw, err := r.dbOps.Read(ctx, r.tableName, id)
		if err != nil {
			log.Printf("WARN: FindActive read id=%s: %v", id, err)
			continue
		}
		reg, err := unmarshalTaxRegistration(raw)
		if err != nil {
			log.Printf("WARN: FindActive unmarshal id=%s: %v", id, err)
			continue
		}
		items = append(items, reg)
	}
	return items, rows.Err()
}

// FindActiveByComputePath returns the ACTIVE registration for (party, computePath) at asOf.
//
// Dialect changes from postgres gold standard:
//   - row_to_json(tr) → id-only scan + dbOps.Read
//   - $1..$5 → ? (positional, re-sequenced)
//   - $1 = ANY(applicable_party_types) → JSON_CONTAINS not needed here (no array column)
//   - jurisdiction join uses identical SQL structure; LIKE replaces no ILIKE needed
func (r *MySQLTaxRegistrationRepository) FindActiveByComputePath(ctx context.Context, partyType, partyID, computePath, jurisdiction string, asOf time.Time) (*taxregistrationpb.TaxRegistration, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindActiveByComputePath requires raw *sql.DB")
	}

	var id string
	var err error

	if jurisdiction != "" {
		row := r.db.QueryRowContext(ctx,
			`SELECT tr.id
			 FROM tax_registration tr
			 JOIN tax_registration_kind trk ON trk.id = tr.tax_registration_kind_id
			 WHERE tr.party_type = ?
			   AND tr.party_id = ?
			   AND tr.compute_path_snapshot = ?
			   AND trk.jurisdiction = ?
			   AND tr.status IN (2, 3, 4)
			   AND tr.effective_from <= ?
			   AND (tr.effective_to IS NULL OR tr.effective_to > ?)
			 ORDER BY tr.effective_from DESC
			 LIMIT 1`,
			partyType, partyID, computePath, jurisdiction, asOf, asOf,
		)
		if err = row.Scan(&id); err == sql.ErrNoRows {
			return nil, nil
		} else if err != nil {
			return nil, fmt.Errorf("FindActiveByComputePath (with jurisdiction) query: %w", err)
		}
	} else {
		row := r.db.QueryRowContext(ctx,
			`SELECT tr.id
			 FROM tax_registration tr
			 WHERE tr.party_type = ?
			   AND tr.party_id = ?
			   AND tr.compute_path_snapshot = ?
			   AND tr.status IN (2, 3, 4)
			   AND tr.effective_from <= ?
			   AND (tr.effective_to IS NULL OR tr.effective_to > ?)
			 ORDER BY tr.effective_from DESC
			 LIMIT 1`,
			partyType, partyID, computePath, asOf, asOf,
		)
		if err = row.Scan(&id); err == sql.ErrNoRows {
			return nil, nil
		} else if err != nil {
			return nil, fmt.Errorf("FindActiveByComputePath (no jurisdiction) query: %w", err)
		}
	}

	raw, err := r.dbOps.Read(ctx, r.tableName, id)
	if err != nil {
		return nil, fmt.Errorf("FindActiveByComputePath read: %w", err)
	}
	return unmarshalTaxRegistration(raw)
}

var _ TaxRegistrationQueries = (*MySQLTaxRegistrationRepository)(nil)
