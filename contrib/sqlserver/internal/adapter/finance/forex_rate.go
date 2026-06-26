//go:build sqlserver

package finance

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/shared/identity"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	forexratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/finance/forex_rate"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.ForexRate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver forex_rate repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerForexRateRepository(db, dbOps, tableName), nil
	})
}

// SQLServerForexRateRepository implements forex_rate read/append operations using SQL Server.
// ForexRate is read-only in the UI; rows are appended only via RecordOperatorRate use case.
type SQLServerForexRateRepository struct {
	forexratepb.UnimplementedForexRateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerForexRateRepository creates a new SQL Server forex_rate repository.
func NewSQLServerForexRateRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) forexratepb.ForexRateDomainServiceServer {
	if tableName == "" {
		tableName = entityid.ForexRate
	}
	return &SQLServerForexRateRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func unmarshalForexRate(raw map[string]any) (*forexratepb.ForexRate, error) {
	js, err := json.Marshal(sqlserverCore.DenormalizeKeys(raw))
	if err != nil {
		return nil, fmt.Errorf("marshal raw: %w", err)
	}
	r := &forexratepb.ForexRate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(js, r); err != nil {
		return nil, fmt.Errorf("unmarshal proto: %w", err)
	}
	return r, nil
}

// ReadForexRate retrieves a forex_rate record by ID.
func (r *SQLServerForexRateRepository) ReadForexRate(ctx context.Context, req *forexratepb.ReadForexRateRequest) (*forexratepb.ReadForexRateResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("forex_rate ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read forex_rate: %w", err)
	}
	rate, err := unmarshalForexRate(result)
	if err != nil {
		return nil, err
	}
	return &forexratepb.ReadForexRateResponse{Success: true, Data: []*forexratepb.ForexRate{rate}}, nil
}

// ListForexRates lists forex_rate records.
func (r *SQLServerForexRateRepository) ListForexRates(ctx context.Context, req *forexratepb.ListForexRatesRequest) (*forexratepb.ListForexRatesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list forex_rates: %w", err)
	}
	var items []*forexratepb.ForexRate
	for _, raw := range listResult.Data {
		rate, err := unmarshalForexRate(raw)
		if err != nil {
			log.Printf("WARN: unmarshal forex_rate: %v", err)
			continue
		}
		items = append(items, rate)
	}
	return &forexratepb.ListForexRatesResponse{Success: true, Data: items}, nil
}

// ForexRateQueries provides additional operations used by RecordOperatorRate.
type ForexRateQueries interface {
	FindMostRecent(ctx context.Context, workspaceID, fromCurrency, toCurrency string) (*forexratepb.ForexRate, error)
	FindActive(ctx context.Context, workspaceID, fromCurrency, toCurrency string, asOf time.Time) (*forexratepb.ForexRate, error)
	Insert(ctx context.Context, rate *forexratepb.ForexRate) error
	SupersedePrior(ctx context.Context, priorID string, effectiveTo time.Time) error
}

// FindMostRecent returns the most recent ACTIVE forex_rate for the workspace + currency pair.
//
// SQL Server differences from the postgres gold standard:
//   - $1/$2/$3 → @p1/@p2/@p3.
//   - row_to_json() not available — explicit column scan.
//   - LIMIT 1 → SELECT TOP 1 with ORDER BY.
func (r *SQLServerForexRateRepository) FindMostRecent(ctx context.Context, workspaceID, fromCurrency, toCurrency string) (*forexratepb.ForexRate, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindMostRecent requires raw *sql.DB")
	}
	row := r.db.QueryRowContext(ctx,
		`SELECT TOP 1 fr.id, fr.workspace_id, fr.from_currency, fr.to_currency,
		        fr.rate_bps, fr.status, fr.effective_from, fr.effective_to
		 FROM forex_rate fr
		 WHERE fr.workspace_id = @p1
		   AND fr.from_currency = @p2
		   AND fr.to_currency = @p3
		   AND fr.status = 2
		 ORDER BY fr.effective_from DESC`,
		workspaceID, fromCurrency, toCurrency,
	)
	return r.scanForexRateRow(row)
}

// FindActive returns the ACTIVE forex_rate for the workspace + currency pair at asOf.
//
// SQL Server differences: same as FindMostRecent plus @p4 for asOf.
func (r *SQLServerForexRateRepository) FindActive(ctx context.Context, workspaceID, fromCurrency, toCurrency string, asOf time.Time) (*forexratepb.ForexRate, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindActive requires raw *sql.DB")
	}
	row := r.db.QueryRowContext(ctx,
		`SELECT TOP 1 fr.id, fr.workspace_id, fr.from_currency, fr.to_currency,
		        fr.rate_bps, fr.status, fr.effective_from, fr.effective_to
		 FROM forex_rate fr
		 WHERE fr.workspace_id = @p1
		   AND fr.from_currency = @p2
		   AND fr.to_currency = @p3
		   AND fr.status IN (2, 3)
		   AND fr.effective_from <= @p4
		   AND (fr.effective_to IS NULL OR fr.effective_to > @p4)
		 ORDER BY fr.effective_from DESC`,
		workspaceID, fromCurrency, toCurrency, asOf,
	)
	return r.scanForexRateRow(row)
}

func (r *SQLServerForexRateRepository) scanForexRateRow(row *sql.Row) (*forexratepb.ForexRate, error) {
	var (
		id             string
		workspaceIDV   string
		fromCurrency   string
		toCurrency     string
		rateMicroUnits int64
		status         int32
		effectiveFrom  time.Time
		effectiveTo    *time.Time
	)
	if err := row.Scan(&id, &workspaceIDV, &fromCurrency, &toCurrency,
		&rateMicroUnits, &status, &effectiveFrom, &effectiveTo); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("scanForexRateRow: %w", err)
	}
	rate := &forexratepb.ForexRate{
		Id:             id,
		WorkspaceId:    workspaceIDV,
		FromCurrency:   fromCurrency,
		ToCurrency:     toCurrency,
		RateMicroUnits: rateMicroUnits,
		Status:         forexratepb.ForexRateStatus(status),
	}
	return rate, nil
}

// Insert appends a new forex_rate row.
func (r *SQLServerForexRateRepository) Insert(ctx context.Context, rate *forexratepb.ForexRate) error {
	jsonData, err := protojson.Marshal(rate)
	if err != nil {
		return fmt.Errorf("marshal proto: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return fmt.Errorf("unmarshal to map: %w", err)
	}
	if _, err := r.dbOps.Create(ctx, r.tableName, data); err != nil {
		return fmt.Errorf("insert forex_rate: %w", err)
	}
	return nil
}

// SupersedePrior marks the prior ACTIVE forex_rate row as SUPERSEDED with effective_to set.
// A1: workspace_id guard added — prevents cross-tenant mutation if priorID is guessed.
//
// SQL Server differences from the postgres gold standard:
//   - $1/$2/$3 → @p1/@p2/@p3.
//   - No RETURNING clause — SQL Server uses OUTPUT inserted.*, but here we only
//     need success/failure so RowsAffected is sufficient.
func (r *SQLServerForexRateRepository) SupersedePrior(ctx context.Context, priorID string, effectiveTo time.Time) error {
	if r.db == nil {
		return fmt.Errorf("SupersedePrior requires raw *sql.DB")
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE forex_rate SET status = 3, effective_to = @p1
		 WHERE id = @p2
		   AND workspace_id = @p3
		   AND status = 2`,
		effectiveTo, priorID, r.workspaceIDFromCtx(ctx),
	)
	if err != nil {
		return fmt.Errorf("SupersedePrior update: %w", err)
	}
	return nil
}

// workspaceIDFromCtx extracts the workspace ID from the context.
func (r *SQLServerForexRateRepository) workspaceIDFromCtx(ctx context.Context) string {
	return identity.Must(ctx).WorkspaceID
}

var _ ForexRateQueries = (*SQLServerForexRateRepository)(nil)
