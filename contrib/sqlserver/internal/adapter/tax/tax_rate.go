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
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	taxratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_rate"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.TaxRate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver tax_rate repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerTaxRateRepository(db, dbOps, tableName), nil
	})
}

// SQLServerTaxRateRepository implements tax_rate read operations using SQL Server.
type SQLServerTaxRateRepository struct {
	taxratepb.UnimplementedTaxRateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerTaxRateRepository creates a new SQL Server tax_rate repository.
func NewSQLServerTaxRateRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) taxratepb.TaxRateDomainServiceServer {
	if tableName == "" {
		tableName = entityid.TaxRate
	}
	return &SQLServerTaxRateRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func unmarshalTaxRate(raw map[string]any) (*taxratepb.TaxRate, error) {
	js, err := json.Marshal(sqlserverCore.DenormalizeKeys(raw))
	if err != nil {
		return nil, fmt.Errorf("marshal raw: %w", err)
	}
	r := &taxratepb.TaxRate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(js, r); err != nil {
		return nil, fmt.Errorf("unmarshal proto: %w", err)
	}
	return r, nil
}

// ReadTaxRate retrieves a tax_rate record by ID.
func (r *SQLServerTaxRateRepository) ReadTaxRate(ctx context.Context, req *taxratepb.ReadTaxRateRequest) (*taxratepb.ReadTaxRateResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tax_rate ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read tax_rate: %w", err)
	}
	rate, err := unmarshalTaxRate(result)
	if err != nil {
		return nil, err
	}
	return &taxratepb.ReadTaxRateResponse{Success: true, Data: []*taxratepb.TaxRate{rate}}, nil
}

// ListTaxRates lists all tax_rate records.
func (r *SQLServerTaxRateRepository) ListTaxRates(ctx context.Context, req *taxratepb.ListTaxRatesRequest) (*taxratepb.ListTaxRatesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list tax_rates: %w", err)
	}
	var items []*taxratepb.TaxRate
	for _, raw := range listResult.Data {
		rate, err := unmarshalTaxRate(raw)
		if err != nil {
			log.Printf("WARN: unmarshal tax_rate: %v", err)
			continue
		}
		items = append(items, rate)
	}
	return &taxratepb.ListTaxRatesResponse{Success: true, Data: items}, nil
}

// FindApplicableQueries is the interface consumed by the compute use case.
type FindApplicableQueries interface {
	// FindApplicable returns the most-specific ACTIVE or SUPERSEDED tax_rate row
	// valid on asOf for the given (jurisdiction, authority_code, kind, treatment,
	// direction). Workspace-scoped rows take precedence over global rows.
	FindApplicable(ctx context.Context, workspaceID, jurisdiction, authorityCode, kind, treatment, direction string, asOf time.Time) (*taxratepb.TaxRate, error)
}

// FindApplicable implements the asOf-pinned lookup described in plan.md § "The asOf rule".
//
// SQL Server differences from the postgres gold standard:
//   - $1..$7 → @p1..@p7.
//   - row_to_json() not available — columns selected explicitly and scanned directly.
//   - LIMIT 1 → SELECT TOP 1 (applied on the outer query with ORDER BY).
//   - CASE expression for workspace-precedence ordering (no syntax change needed).
//   - active = true → status IN (2, 3) (no change, these are integers).
func (r *SQLServerTaxRateRepository) FindApplicable(ctx context.Context, workspaceID, jurisdiction, authorityCode, kind, treatment, direction string, asOf time.Time) (*taxratepb.TaxRate, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindApplicable requires raw *sql.DB")
	}

	row := r.db.QueryRowContext(ctx,
		`SELECT TOP 1
			tr.id, tr.jurisdiction, tr.authority_code, tr.kind, tr.treatment_code,
			tr.direction, tr.rate_bps, tr.status, tr.effective_from, tr.effective_to,
			tr.workspace_id
		 FROM tax_rate tr
		 WHERE tr.jurisdiction = @p1
		   AND tr.authority_code = @p2
		   AND tr.kind = @p3
		   AND (@p4 = '' OR tr.treatment_code IS NULL OR tr.treatment_code = @p4)
		   AND tr.direction = @p5
		   AND tr.status IN (2, 3)
		   AND tr.effective_from <= @p6
		   AND (tr.effective_to IS NULL OR tr.effective_to > @p6)
		   AND (tr.workspace_id = @p7 OR tr.workspace_id IS NULL)
		 ORDER BY
		   CASE WHEN tr.workspace_id = @p7 THEN 0 ELSE 1 END,
		   tr.effective_from DESC`,
		jurisdiction, authorityCode, kind, treatment, direction, asOf, workspaceID,
	)

	var (
		id            string
		jurisdictionV string
		authCode      string
		kindV         string
		treatmentCode *string
		directionV    string
		rateBps       int64
		status        int32
		effectiveFrom time.Time
		effectiveTo   *time.Time
		workspaceIDV  *string
	)
	if err := row.Scan(&id, &jurisdictionV, &authCode, &kindV, &treatmentCode,
		&directionV, &rateBps, &status, &effectiveFrom, &effectiveTo, &workspaceIDV); err == sql.ErrNoRows {
		return nil, nil // caller treats nil as "no applicable rate"
	} else if err != nil {
		return nil, fmt.Errorf("FindApplicable query: %w", err)
	}

	rate := &taxratepb.TaxRate{
		Id:              id,
		Jurisdiction:    jurisdictionV,
		AuthorityCode:   authCode,
		Kind:            kindV,
		RateBasisPoints: int32(rateBps),
		Status:          taxratepb.TaxRateStatus(status),
	}
	if val, ok := taxratepb.TaxRateDirection_value[directionV]; ok {
		rate.Direction = taxratepb.TaxRateDirection(val)
	}
	if treatmentCode != nil {
		rate.TreatmentCode = treatmentCode
	}
	if workspaceIDV != nil {
		rate.WorkspaceId = workspaceIDV
	}
	return rate, nil
}

var _ FindApplicableQueries = (*SQLServerTaxRateRepository)(nil)
