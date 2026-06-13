//go:build postgresql

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
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	forexratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/finance/forex_rate"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ForexRate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres forex_rate repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresForexRateRepository(db, dbOps, tableName), nil
	})
}

// PostgresForexRateRepository implements forex_rate read/append operations using PostgreSQL.
// ForexRate is read-only in the UI; rows are appended only via RecordOperatorRate use case.
type PostgresForexRateRepository struct {
	forexratepb.UnimplementedForexRateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresForexRateRepository creates a new PostgreSQL forex_rate repository.
func NewPostgresForexRateRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) forexratepb.ForexRateDomainServiceServer {
	if tableName == "" {
		tableName = entityid.ForexRate
	}
	return &PostgresForexRateRepository{
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
func (r *PostgresForexRateRepository) ReadForexRate(ctx context.Context, req *forexratepb.ReadForexRateRequest) (*forexratepb.ReadForexRateResponse, error) {
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
func (r *PostgresForexRateRepository) ListForexRates(ctx context.Context, req *forexratepb.ListForexRatesRequest) (*forexratepb.ListForexRatesResponse, error) {
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
	// FindMostRecent returns the most recent ACTIVE forex_rate row for a currency pair
	// scoped to the given workspace. Returns nil if none found.
	FindMostRecent(ctx context.Context, workspaceID, fromCurrency, toCurrency string) (*forexratepb.ForexRate, error)
	// FindActive returns the ACTIVE forex_rate row for a currency pair at the given time.
	FindActive(ctx context.Context, workspaceID, fromCurrency, toCurrency string, asOf time.Time) (*forexratepb.ForexRate, error)
	// Insert appends a new forex_rate row (used by RecordOperatorRate).
	Insert(ctx context.Context, rate *forexratepb.ForexRate) error
	// SupersedePrior marks the prior ACTIVE row as SUPERSEDED with effective_to=now.
	SupersedePrior(ctx context.Context, priorID string, effectiveTo time.Time) error
}

// FindMostRecent returns the most recent ACTIVE forex_rate for the workspace + currency pair.
func (r *PostgresForexRateRepository) FindMostRecent(ctx context.Context, workspaceID, fromCurrency, toCurrency string) (*forexratepb.ForexRate, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindMostRecent requires raw *sql.DB")
	}
	row := r.db.QueryRowContext(ctx,
		`SELECT row_to_json(fr) FROM forex_rate fr
		 WHERE fr.workspace_id = $1
		   AND fr.from_currency = $2
		   AND fr.to_currency = $3
		   AND fr.status = 2  -- ACTIVE
		 ORDER BY fr.effective_from DESC
		 LIMIT 1`,
		workspaceID, fromCurrency, toCurrency,
	)
	var rawJSON []byte
	if err := row.Scan(&rawJSON); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("FindMostRecent query: %w", err)
	}
	rate := &forexratepb.ForexRate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawJSON, rate); err != nil {
		return nil, fmt.Errorf("FindMostRecent unmarshal: %w", err)
	}
	return rate, nil
}

// FindActive returns the ACTIVE forex_rate for the workspace + currency pair at asOf.
func (r *PostgresForexRateRepository) FindActive(ctx context.Context, workspaceID, fromCurrency, toCurrency string, asOf time.Time) (*forexratepb.ForexRate, error) {
	if r.db == nil {
		return nil, fmt.Errorf("FindActive requires raw *sql.DB")
	}
	row := r.db.QueryRowContext(ctx,
		`SELECT row_to_json(fr) FROM forex_rate fr
		 WHERE fr.workspace_id = $1
		   AND fr.from_currency = $2
		   AND fr.to_currency = $3
		   AND fr.status IN (2, 3)  -- ACTIVE, SUPERSEDED
		   AND fr.effective_from <= $4
		   AND (fr.effective_to IS NULL OR fr.effective_to > $4)
		 ORDER BY fr.effective_from DESC
		 LIMIT 1`,
		workspaceID, fromCurrency, toCurrency, asOf,
	)
	var rawJSON []byte
	if err := row.Scan(&rawJSON); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("FindActive query: %w", err)
	}
	rate := &forexratepb.ForexRate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawJSON, rate); err != nil {
		return nil, fmt.Errorf("FindActive unmarshal: %w", err)
	}
	return rate, nil
}

// Insert appends a new forex_rate row.
func (r *PostgresForexRateRepository) Insert(ctx context.Context, rate *forexratepb.ForexRate) error {
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
func (r *PostgresForexRateRepository) SupersedePrior(ctx context.Context, priorID string, effectiveTo time.Time) error {
	if r.db == nil {
		return fmt.Errorf("SupersedePrior requires raw *sql.DB")
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE forex_rate SET status = 3, effective_to = $1
		 WHERE id = $2
		   AND workspace_id = $3
		   AND status = 2`,
		// status 3 = SUPERSEDED
		effectiveTo, priorID, r.workspaceIDFromCtx(ctx),
	)
	if err != nil {
		return fmt.Errorf("SupersedePrior update: %w", err)
	}
	return nil
}

// workspaceIDFromCtx extracts the workspace ID from the context for inline SQL
// workspace predicates. Mirrors the same pattern used in operation/job.go.
func (r *PostgresForexRateRepository) workspaceIDFromCtx(ctx context.Context) string {
	return identity.Must(ctx).WorkspaceID
}

var _ ForexRateQueries = (*PostgresForexRateRepository)(nil)
