//go:build sqlserver

package tax

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	taxregistrationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.TaxRegistration, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver tax_registration repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerTaxRegistrationRepository(db, dbOps, tableName), nil
	})
}

// SQLServerTaxRegistrationRepository implements tax_registration CRUD using SQL Server.
// NOTE: "Update" in this domain means Supersede — a new immutable row is inserted and
// the prior ACTIVE row is marked SUPERSEDED with effective_to set. "Delete" means Revoke
// — similar mutation setting status=CANCELLED.
type SQLServerTaxRegistrationRepository struct {
	taxregistrationpb.UnimplementedTaxRegistrationDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerTaxRegistrationRepository creates a new SQL Server tax_registration repository.
func NewSQLServerTaxRegistrationRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) taxregistrationpb.TaxRegistrationDomainServiceServer {
	if tableName == "" {
		tableName = entityid.TaxRegistration
	}
	return &SQLServerTaxRegistrationRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func unmarshalTaxRegistration(raw map[string]any) (*taxregistrationpb.TaxRegistration, error) {
	js, err := json.Marshal(sqlserverCore.DenormalizeKeys(raw))
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
func (r *SQLServerTaxRegistrationRepository) CreateTaxRegistration(ctx context.Context, req *taxregistrationpb.CreateTaxRegistrationRequest) (*taxregistrationpb.CreateTaxRegistrationResponse, error) {
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
func (r *SQLServerTaxRegistrationRepository) ReadTaxRegistration(ctx context.Context, req *taxregistrationpb.ReadTaxRegistrationRequest) (*taxregistrationpb.ReadTaxRegistrationResponse, error) {
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
func (r *SQLServerTaxRegistrationRepository) UpdateTaxRegistration(ctx context.Context, req *taxregistrationpb.UpdateTaxRegistrationRequest) (*taxregistrationpb.UpdateTaxRegistrationResponse, error) {
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
func (r *SQLServerTaxRegistrationRepository) DeleteTaxRegistration(ctx context.Context, req *taxregistrationpb.DeleteTaxRegistrationRequest) (*taxregistrationpb.DeleteTaxRegistrationResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tax_registration ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete tax_registration: %w", err)
	}
	return &taxregistrationpb.DeleteTaxRegistrationResponse{Success: true}, nil
}

// ListTaxRegistrations lists tax_registration records.
func (r *SQLServerTaxRegistrationRepository) ListTaxRegistrations(ctx context.Context, req *taxregistrationpb.ListTaxRegistrationsRequest) (*taxregistrationpb.ListTaxRegistrationsResponse, error) {
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
// SQL Server differences from the postgres gold standard:
//   - $1/$2/$3 → @p1/@p2/@p3.
//   - row_to_json() not available — columns are selected explicitly and scanned.
//   - LIMIT 1 → SELECT TOP 1 for single-row queries (FindActiveByComputePath).
//   - active = true → status IN (2, 3, 4) (integers, no change needed).
func (r *SQLServerTaxRegistrationRepository) FindActive(ctx context.Context, partyType, partyID string, asOf time.Time) ([]*taxregistrationpb.TaxRegistration, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindActive requires raw *sql.DB")
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT tr.id, tr.party_type, tr.party_id, tr.tax_registration_kind_id,
		        tr.compute_path_snapshot, tr.party_role_snapshot,
		        tr.status, tr.effective_from, tr.effective_to, tr.workspace_id
		 FROM tax_registration tr
		 WHERE tr.party_type = @p1
		   AND tr.party_id = @p2
		   AND tr.status IN (2, 3, 4)
		   AND tr.effective_from <= @p3
		   AND (tr.effective_to IS NULL OR tr.effective_to > @p3)
		 ORDER BY tr.effective_from DESC`,
		partyType, partyID, asOf,
	)
	if err != nil {
		return nil, fmt.Errorf("FindActive query: %w", err)
	}
	defer rows.Close()

	var items []*taxregistrationpb.TaxRegistration
	for rows.Next() {
		var (
			id                  string
			partyTypeV          string
			partyIDV            string
			kindID              string
			computePathSnapshot *string
			partyRoleSnapshot   *string
			status              int32
			effectiveFrom       time.Time
			effectiveTo         *time.Time
			workspaceID         *string
		)
		if err := rows.Scan(&id, &partyTypeV, &partyIDV, &kindID,
			&computePathSnapshot, &partyRoleSnapshot,
			&status, &effectiveFrom, &effectiveTo, &workspaceID); err != nil {
			return nil, fmt.Errorf("FindActive scan: %w", err)
		}
		reg := &taxregistrationpb.TaxRegistration{
			Id:                    id,
			PartyId:               partyIDV,
			TaxRegistrationKindId: kindID,
			Status:                taxregistrationpb.TaxRegistrationStatus(status),
		}
		if val, ok := taxregistrationpb.TaxRegistrationPartyType_value[partyTypeV]; ok {
			reg.PartyType = taxregistrationpb.TaxRegistrationPartyType(val)
		}
		if computePathSnapshot != nil {
			if val, ok := taxregistrationpb.TaxRegistrationComputePathSnapshot_value[*computePathSnapshot]; ok {
				reg.ComputePathSnapshot = taxregistrationpb.TaxRegistrationComputePathSnapshot(val)
			}
		}
		if partyRoleSnapshot != nil {
			if val, ok := taxregistrationpb.TaxRegistrationPartyRoleSnapshot_value[*partyRoleSnapshot]; ok {
				reg.PartyRoleSnapshot = taxregistrationpb.TaxRegistrationPartyRoleSnapshot(val)
			}
		}
		if workspaceID != nil {
			reg.WorkspaceId = *workspaceID
		}
		items = append(items, reg)
	}
	return items, rows.Err()
}

// FindActiveByComputePath returns the ACTIVE registration for (party, computePath) at asOf.
//
// SQL Server differences from the postgres gold standard:
//   - $N → @pN.
//   - LIMIT 1 → SELECT TOP 1 with ORDER BY (required for deterministic TOP).
//   - No row_to_json() — explicit column scan.
func (r *SQLServerTaxRegistrationRepository) FindActiveByComputePath(ctx context.Context, partyType, partyID, computePath, jurisdiction string, asOf time.Time) (*taxregistrationpb.TaxRegistration, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindActiveByComputePath requires raw *sql.DB")
	}

	var (
		id                  string
		partyTypeV          string
		partyIDV            string
		kindID              string
		computePathSnapshot *string
		partyRoleSnapshot   *string
		status              int32
		effectiveFrom       time.Time
		effectiveTo         *time.Time
		workspaceID         *string
		scanErr             error
	)

	if jurisdiction != "" {
		row := r.db.QueryRowContext(ctx,
			`SELECT TOP 1
				tr.id, tr.party_type, tr.party_id, tr.tax_registration_kind_id,
				tr.compute_path_snapshot, tr.party_role_snapshot,
				tr.status, tr.effective_from, tr.effective_to, tr.workspace_id
			 FROM tax_registration tr
			 JOIN tax_registration_kind trk ON trk.id = tr.tax_registration_kind_id
			 WHERE tr.party_type = @p1
			   AND tr.party_id = @p2
			   AND tr.compute_path_snapshot = @p3
			   AND trk.jurisdiction = @p4
			   AND tr.status IN (2, 3, 4)
			   AND tr.effective_from <= @p5
			   AND (tr.effective_to IS NULL OR tr.effective_to > @p5)
			 ORDER BY tr.effective_from DESC`,
			partyType, partyID, computePath, jurisdiction, asOf,
		)
		scanErr = row.Scan(&id, &partyTypeV, &partyIDV, &kindID,
			&computePathSnapshot, &partyRoleSnapshot,
			&status, &effectiveFrom, &effectiveTo, &workspaceID)
	} else {
		row := r.db.QueryRowContext(ctx,
			`SELECT TOP 1
				tr.id, tr.party_type, tr.party_id, tr.tax_registration_kind_id,
				tr.compute_path_snapshot, tr.party_role_snapshot,
				tr.status, tr.effective_from, tr.effective_to, tr.workspace_id
			 FROM tax_registration tr
			 WHERE tr.party_type = @p1
			   AND tr.party_id = @p2
			   AND tr.compute_path_snapshot = @p3
			   AND tr.status IN (2, 3, 4)
			   AND tr.effective_from <= @p4
			   AND (tr.effective_to IS NULL OR tr.effective_to > @p4)
			 ORDER BY tr.effective_from DESC`,
			partyType, partyID, computePath, asOf,
		)
		scanErr = row.Scan(&id, &partyTypeV, &partyIDV, &kindID,
			&computePathSnapshot, &partyRoleSnapshot,
			&status, &effectiveFrom, &effectiveTo, &workspaceID)
	}

	if scanErr == sql.ErrNoRows {
		return nil, nil
	}
	if scanErr != nil {
		return nil, fmt.Errorf("FindActiveByComputePath query: %w", scanErr)
	}

	reg := &taxregistrationpb.TaxRegistration{
		Id:                    id,
		PartyId:               partyIDV,
		TaxRegistrationKindId: kindID,
		Status:                taxregistrationpb.TaxRegistrationStatus(status),
	}
	if val, ok := taxregistrationpb.TaxRegistrationPartyType_value[partyTypeV]; ok {
		reg.PartyType = taxregistrationpb.TaxRegistrationPartyType(val)
	}
	if computePathSnapshot != nil {
		if val, ok := taxregistrationpb.TaxRegistrationComputePathSnapshot_value[*computePathSnapshot]; ok {
			reg.ComputePathSnapshot = taxregistrationpb.TaxRegistrationComputePathSnapshot(val)
		}
	}
	if partyRoleSnapshot != nil {
		if val, ok := taxregistrationpb.TaxRegistrationPartyRoleSnapshot_value[*partyRoleSnapshot]; ok {
			reg.PartyRoleSnapshot = taxregistrationpb.TaxRegistrationPartyRoleSnapshot(val)
		}
	}
	if workspaceID != nil {
		reg.WorkspaceId = *workspaceID
	}
	return reg, nil
}

var _ TaxRegistrationQueries = (*SQLServerTaxRegistrationRepository)(nil)
