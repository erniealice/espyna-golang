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
	taxratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_rate"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.TaxRate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres tax_rate repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresTaxRateRepository(db, dbOps, tableName), nil
	})
}

// PostgresTaxRateRepository implements tax_rate read operations using PostgreSQL.
type PostgresTaxRateRepository struct {
	taxratepb.UnimplementedTaxRateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresTaxRateRepository creates a new PostgreSQL tax_rate repository.
func NewPostgresTaxRateRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) taxratepb.TaxRateDomainServiceServer {
	if tableName == "" {
		tableName = entityid.TaxRate
	}
	return &PostgresTaxRateRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func unmarshalTaxRate(raw map[string]any) (*taxratepb.TaxRate, error) {
	js, err := json.Marshal(raw)
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
func (r *PostgresTaxRateRepository) ReadTaxRate(ctx context.Context, req *taxratepb.ReadTaxRateRequest) (*taxratepb.ReadTaxRateResponse, error) {
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
func (r *PostgresTaxRateRepository) ListTaxRates(ctx context.Context, req *taxratepb.ListTaxRatesRequest) (*taxratepb.ListTaxRatesResponse, error) {
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
// Lookup precedence:
//  1. workspace_id = $workspaceID (workspace override)
//  2. workspace_id IS NULL (global fallback)
//
// Both restricted to (jurisdiction, authority_code, kind, direction, asOf window) match.
// treatment may be "" to match rows with treatment_code IS NULL (any-treatment rows).
func (r *PostgresTaxRateRepository) FindApplicable(ctx context.Context, workspaceID, jurisdiction, authorityCode, kind, treatment, direction string, asOf time.Time) (*taxratepb.TaxRate, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindApplicable requires raw *sql.DB")
	}

	row := r.db.QueryRowContext(ctx,
		`SELECT row_to_json(tr) FROM tax_rate tr
		 WHERE tr.jurisdiction = $1
		   AND tr.authority_code = $2
		   AND tr.kind = $3
		   AND ($4 = '' OR tr.treatment_code IS NULL OR tr.treatment_code = $4)
		   AND tr.direction = $5
		   AND tr.status IN (2, 3) -- ACTIVE=2, SUPERSEDED=3
		   AND tr.effective_from <= $6
		   AND (tr.effective_to IS NULL OR tr.effective_to > $6)
		   AND (tr.workspace_id = $7 OR tr.workspace_id IS NULL)
		 ORDER BY
		   CASE WHEN tr.workspace_id = $7 THEN 0 ELSE 1 END,
		   tr.effective_from DESC
		 LIMIT 1`,
		jurisdiction, authorityCode, kind, treatment, direction, asOf, workspaceID,
	)

	var rawJSON []byte
	if err := row.Scan(&rawJSON); err == sql.ErrNoRows {
		return nil, nil // caller treats nil as "no applicable rate"
	} else if err != nil {
		return nil, fmt.Errorf("FindApplicable query: %w", err)
	}

	rate := &taxratepb.TaxRate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawJSON, rate); err != nil {
		return nil, fmt.Errorf("FindApplicable unmarshal: %w", err)
	}
	return rate, nil
}

var _ FindApplicableQueries = (*PostgresTaxRateRepository)(nil)
