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
	taxratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_rate"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.TaxRate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql tax_rate repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLTaxRateRepository(db, dbOps, tableName), nil
	})
}

// MySQLTaxRateRepository implements tax_rate read operations using MySQL 8.0+.
type MySQLTaxRateRepository struct {
	taxratepb.UnimplementedTaxRateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLTaxRateRepository creates a new MySQL tax_rate repository.
func NewMySQLTaxRateRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) taxratepb.TaxRateDomainServiceServer {
	if tableName == "" {
		tableName = entityid.TaxRate
	}
	return &MySQLTaxRateRepository{
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
func (r *MySQLTaxRateRepository) ReadTaxRate(ctx context.Context, req *taxratepb.ReadTaxRateRequest) (*taxratepb.ReadTaxRateResponse, error) {
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
func (r *MySQLTaxRateRepository) ListTaxRates(ctx context.Context, req *taxratepb.ListTaxRatesRequest) (*taxratepb.ListTaxRatesResponse, error) {
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
	FindApplicable(ctx context.Context, workspaceID, jurisdiction, authorityCode, kind, treatment, direction string, asOf time.Time) (*taxratepb.TaxRate, error)
}

// FindApplicable implements the asOf-pinned lookup for MySQL.
//
// Dialect changes from postgres gold standard:
//   - row_to_json(tr) → id-only scan + dbOps.Read (avoids enumerating all columns)
//   - $1..$7 → ? (re-sequenced in same positional order)
//   - $4 = ” → ? = ” (no cast needed in MySQL)
//   - tr.workspace_id = $7 OR tr.workspace_id IS NULL stays identical
//   - CASE WHEN tr.workspace_id = $7 THEN 0 ELSE 1 END → same (MySQL supports CASE)
//   - Tax rates are basis-points — untouched.
func (r *MySQLTaxRateRepository) FindApplicable(ctx context.Context, workspaceID, jurisdiction, authorityCode, kind, treatment, direction string, asOf time.Time) (*taxratepb.TaxRate, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindApplicable requires raw *sql.DB")
	}

	// Dialect: ? placeholders; ? = '' instead of $4 = ''; CASE preserved.
	// Arg order: jurisdiction, authorityCode, kind, treatment (x2), direction, asOf (x2), workspaceID (x3)
	// Re-sequenced from postgres $1..$7 → positional ? order matching arg slice.
	var id string
	row := r.db.QueryRowContext(ctx,
		`SELECT tr.id FROM tax_rate tr
		 WHERE tr.jurisdiction = ?
		   AND tr.authority_code = ?
		   AND tr.kind = ?
		   AND (? = '' OR tr.treatment_code IS NULL OR tr.treatment_code = ?)
		   AND tr.direction = ?
		   AND tr.status IN (2, 3)
		   AND tr.effective_from <= ?
		   AND (tr.effective_to IS NULL OR tr.effective_to > ?)
		   AND (tr.workspace_id = ? OR tr.workspace_id IS NULL)
		 ORDER BY
		   CASE WHEN tr.workspace_id = ? THEN 0 ELSE 1 END,
		   tr.effective_from DESC
		 LIMIT 1`,
		jurisdiction, authorityCode, kind, treatment, treatment, direction, asOf, asOf, workspaceID, workspaceID,
	)
	if err := row.Scan(&id); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("FindApplicable query: %w", err)
	}

	raw, err := r.dbOps.Read(ctx, r.tableName, id)
	if err != nil {
		return nil, fmt.Errorf("FindApplicable read: %w", err)
	}
	return unmarshalTaxRate(raw)
}

var _ FindApplicableQueries = (*MySQLTaxRateRepository)(nil)
