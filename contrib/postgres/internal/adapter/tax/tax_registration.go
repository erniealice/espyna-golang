//go:build postgresql

package tax

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	taxregistrationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.TaxRegistration, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres tax_registration repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresTaxRegistrationRepository(db, dbOps, tableName), nil
	})
}

// PostgresTaxRegistrationRepository implements tax_registration CRUD using PostgreSQL.
// NOTE: "Update" in this domain means Supersede — a new immutable row is inserted and
// the prior ACTIVE row is marked SUPERSEDED with effective_to set. "Delete" means Revoke
// — similar mutation setting status=CANCELLED.
type PostgresTaxRegistrationRepository struct {
	taxregistrationpb.UnimplementedTaxRegistrationDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresTaxRegistrationRepository creates a new PostgreSQL tax_registration repository.
func NewPostgresTaxRegistrationRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) taxregistrationpb.TaxRegistrationDomainServiceServer {
	if tableName == "" {
		tableName = entityid.TaxRegistration
	}
	return &PostgresTaxRegistrationRepository{
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
// The use case is responsible for setting compute_path_snapshot and party_role_snapshot
// before calling Create (denormed from the kind row).
func (r *PostgresTaxRegistrationRepository) CreateTaxRegistration(ctx context.Context, req *taxregistrationpb.CreateTaxRegistrationRequest) (*taxregistrationpb.CreateTaxRegistrationResponse, error) {
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
func (r *PostgresTaxRegistrationRepository) ReadTaxRegistration(ctx context.Context, req *taxregistrationpb.ReadTaxRegistrationRequest) (*taxregistrationpb.ReadTaxRegistrationResponse, error) {
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
// In normal use the Supersede use case handles supersession semantics; this
// method provides the raw persistence layer for status transitions.
func (r *PostgresTaxRegistrationRepository) UpdateTaxRegistration(ctx context.Context, req *taxregistrationpb.UpdateTaxRegistrationRequest) (*taxregistrationpb.UpdateTaxRegistrationResponse, error) {
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
func (r *PostgresTaxRegistrationRepository) DeleteTaxRegistration(ctx context.Context, req *taxregistrationpb.DeleteTaxRegistrationRequest) (*taxregistrationpb.DeleteTaxRegistrationResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tax_registration ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete tax_registration: %w", err)
	}
	return &taxregistrationpb.DeleteTaxRegistrationResponse{Success: true}, nil
}

// ListTaxRegistrations lists tax_registration records.
func (r *PostgresTaxRegistrationRepository) ListTaxRegistrations(ctx context.Context, req *taxregistrationpb.ListTaxRegistrationsRequest) (*taxregistrationpb.ListTaxRegistrationsResponse, error) {
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
	// FindActive returns all ACTIVE tax_registration rows for the given party at asOf.
	FindActive(ctx context.Context, partyType, partyID string, asOf time.Time) ([]*taxregistrationpb.TaxRegistration, error)
	// FindActiveByComputePath returns the ACTIVE registration for the given party,
	// compute_path_snapshot, and jurisdiction at asOf.  Used by ComputeTaxesForRevenue.
	FindActiveByComputePath(ctx context.Context, partyType, partyID, computePath, jurisdiction string, asOf time.Time) (*taxregistrationpb.TaxRegistration, error)
}

// FindActive returns all ACTIVE tax_registration rows for the given party at asOf.
func (r *PostgresTaxRegistrationRepository) FindActive(ctx context.Context, partyType, partyID string, asOf time.Time) ([]*taxregistrationpb.TaxRegistration, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindActive requires raw *sql.DB")
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT row_to_json(tr) FROM tax_registration tr
		 WHERE tr.party_type = $1
		   AND tr.party_id = $2
		   AND tr.status IN (2, 3, 4)  -- ACTIVE=2, SUPERSEDED=3, CANCELLED=4
		   AND tr.effective_from <= $3
		   AND (tr.effective_to IS NULL OR tr.effective_to > $3)
		 ORDER BY tr.effective_from DESC`,
		partyType, partyID, asOf,
	)
	if err != nil {
		return nil, fmt.Errorf("FindActive query: %w", err)
	}
	defer rows.Close()

	var items []*taxregistrationpb.TaxRegistration
	for rows.Next() {
		var rawJSON []byte
		if err := rows.Scan(&rawJSON); err != nil {
			return nil, fmt.Errorf("FindActive scan: %w", err)
		}
		reg := &taxregistrationpb.TaxRegistration{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawJSON, reg); err != nil {
			log.Printf("WARN: FindActive unmarshal: %v", err)
			continue
		}
		items = append(items, reg)
	}
	return items, rows.Err()
}

// FindActiveByComputePath returns the ACTIVE registration for (party, computePath) at asOf,
// optionally filtered by jurisdiction via a join to tax_authority.
// The jurisdiction parameter corresponds to the workspace's home_jurisdiction (e.g. "PH-NATIONAL").
// NOTE: tax_registration has NO compliance_region column — jurisdiction is stored on the
// linked tax_registration_kind row. We join tax_registration_kind to filter by jurisdiction.
// Passing an empty jurisdiction string skips the jurisdiction filter (returns any matching registration).
func (r *PostgresTaxRegistrationRepository) FindActiveByComputePath(ctx context.Context, partyType, partyID, computePath, jurisdiction string, asOf time.Time) (*taxregistrationpb.TaxRegistration, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindActiveByComputePath requires raw *sql.DB")
	}

	var (
		rawJSON []byte
		err     error
	)

	if jurisdiction != "" {
		// Join to tax_registration_kind to apply the jurisdiction predicate.
		row := r.db.QueryRowContext(ctx,
			`SELECT row_to_json(tr)
			 FROM tax_registration tr
			 JOIN tax_registration_kind trk ON trk.id = tr.tax_registration_kind_id
			 WHERE tr.party_type = $1
			   AND tr.party_id = $2
			   AND tr.compute_path_snapshot = $3
			   AND trk.jurisdiction = $4
			   AND tr.status IN (2, 3, 4)  -- ACTIVE=2, SUPERSEDED=3, CANCELLED=4
			   AND tr.effective_from <= $5
			   AND (tr.effective_to IS NULL OR tr.effective_to > $5)
			 ORDER BY tr.effective_from DESC
			 LIMIT 1`,
			partyType, partyID, computePath, jurisdiction, asOf,
		)
		if err = row.Scan(&rawJSON); err == sql.ErrNoRows {
			return nil, nil
		} else if err != nil {
			return nil, fmt.Errorf("FindActiveByComputePath (with jurisdiction) query: %w", err)
		}
	} else {
		// No jurisdiction filter — return the most-recent matching registration.
		row := r.db.QueryRowContext(ctx,
			`SELECT row_to_json(tr)
			 FROM tax_registration tr
			 WHERE tr.party_type = $1
			   AND tr.party_id = $2
			   AND tr.compute_path_snapshot = $3
			   AND tr.status IN (2, 3, 4)  -- ACTIVE=2, SUPERSEDED=3, CANCELLED=4
			   AND tr.effective_from <= $4
			   AND (tr.effective_to IS NULL OR tr.effective_to > $4)
			 ORDER BY tr.effective_from DESC
			 LIMIT 1`,
			partyType, partyID, computePath, asOf,
		)
		if err = row.Scan(&rawJSON); err == sql.ErrNoRows {
			return nil, nil
		} else if err != nil {
			return nil, fmt.Errorf("FindActiveByComputePath (no jurisdiction) query: %w", err)
		}
	}

	reg := &taxregistrationpb.TaxRegistration{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawJSON, reg); err != nil {
		return nil, fmt.Errorf("FindActiveByComputePath unmarshal: %w", err)
	}
	return reg, nil
}

var _ TaxRegistrationQueries = (*PostgresTaxRegistrationRepository)(nil)
