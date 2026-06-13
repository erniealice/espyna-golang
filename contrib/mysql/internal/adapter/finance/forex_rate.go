//go:build mysql

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
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	forexratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/finance/forex_rate"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.ForexRate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql forex_rate repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLForexRateRepository(db, dbOps, tableName), nil
	})
}

// MySQLForexRateRepository implements forex_rate read/append operations using MySQL 8.0+.
// ForexRate is read-only in the UI; rows are appended only via RecordOperatorRate use case.
type MySQLForexRateRepository struct {
	forexratepb.UnimplementedForexRateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLForexRateRepository creates a new MySQL forex_rate repository.
func NewMySQLForexRateRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) forexratepb.ForexRateDomainServiceServer {
	if tableName == "" {
		tableName = entityid.ForexRate
	}
	return &MySQLForexRateRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func unmarshalForexRate(raw map[string]any) (*forexratepb.ForexRate, error) {
	js, err := json.Marshal(raw)
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
func (r *MySQLForexRateRepository) ReadForexRate(ctx context.Context, req *forexratepb.ReadForexRateRequest) (*forexratepb.ReadForexRateResponse, error) {
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
func (r *MySQLForexRateRepository) ListForexRates(ctx context.Context, req *forexratepb.ListForexRatesRequest) (*forexratepb.ListForexRatesResponse, error) {
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
// Dialect changes from postgres gold standard:
//   - row_to_json(fr) → id-only scan + dbOps.Read
//   - $1..$3 → ? (positional, re-sequenced in same order)
func (r *MySQLForexRateRepository) FindMostRecent(ctx context.Context, workspaceID, fromCurrency, toCurrency string) (*forexratepb.ForexRate, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindMostRecent requires raw *sql.DB")
	}
	var id string
	row := r.db.QueryRowContext(ctx,
		`SELECT fr.id FROM forex_rate fr
		 WHERE fr.workspace_id = ?
		   AND fr.from_currency = ?
		   AND fr.to_currency = ?
		   AND fr.status = 2
		 ORDER BY fr.effective_from DESC
		 LIMIT 1`,
		workspaceID, fromCurrency, toCurrency,
	)
	if err := row.Scan(&id); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("FindMostRecent query: %w", err)
	}
	raw, err := r.dbOps.Read(ctx, r.tableName, id)
	if err != nil {
		return nil, fmt.Errorf("FindMostRecent read: %w", err)
	}
	return unmarshalForexRate(raw)
}

// FindActive returns the ACTIVE forex_rate for the workspace + currency pair at asOf.
//
// Dialect changes from postgres gold standard:
//   - row_to_json(fr) → id-only scan + dbOps.Read
//   - $1..$4 → ? (positional, asOf repeated for both effective_from and effective_to)
func (r *MySQLForexRateRepository) FindActive(ctx context.Context, workspaceID, fromCurrency, toCurrency string, asOf time.Time) (*forexratepb.ForexRate, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindActive requires raw *sql.DB")
	}
	var id string
	row := r.db.QueryRowContext(ctx,
		`SELECT fr.id FROM forex_rate fr
		 WHERE fr.workspace_id = ?
		   AND fr.from_currency = ?
		   AND fr.to_currency = ?
		   AND fr.status IN (2, 3)
		   AND fr.effective_from <= ?
		   AND (fr.effective_to IS NULL OR fr.effective_to > ?)
		 ORDER BY fr.effective_from DESC
		 LIMIT 1`,
		workspaceID, fromCurrency, toCurrency, asOf, asOf,
	)
	if err := row.Scan(&id); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("FindActive query: %w", err)
	}
	raw, err := r.dbOps.Read(ctx, r.tableName, id)
	if err != nil {
		return nil, fmt.Errorf("FindActive read: %w", err)
	}
	return unmarshalForexRate(raw)
}

// Insert appends a new forex_rate row.
func (r *MySQLForexRateRepository) Insert(ctx context.Context, rate *forexratepb.ForexRate) error {
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
// Dialect changes from postgres gold standard:
//   - $1..$3 → ? (positional, re-sequenced: effectiveTo, priorID, workspaceID)
//   - No RETURNING clause — two-step: UPDATE then SELECT if caller needs the row.
func (r *MySQLForexRateRepository) SupersedePrior(ctx context.Context, priorID string, effectiveTo time.Time) error {
	if r.db == nil {
		return fmt.Errorf("SupersedePrior requires raw *sql.DB")
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE forex_rate SET status = 3, effective_to = ?
		 WHERE id = ?
		   AND workspace_id = ?
		   AND status = 2`,
		effectiveTo, priorID, identity.Must(ctx).WorkspaceID,
	)
	if err != nil {
		return fmt.Errorf("SupersedePrior update: %w", err)
	}
	return nil
}

var _ ForexRateQueries = (*MySQLForexRateRepository)(nil)
