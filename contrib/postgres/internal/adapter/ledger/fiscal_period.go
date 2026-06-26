//go:build postgresql

package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	fiscalperiodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/fiscal_period"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.FiscalPeriod, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres fiscal_period repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresFiscalPeriodRepository(dbOps, tableName), nil
	})
}

// PostgresFiscalPeriodRepository implements fiscal_period CRUD and lifecycle operations using PostgreSQL.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_fiscal_period_status ON fiscal_period(status)
//   - CREATE INDEX idx_fiscal_period_start_date ON fiscal_period(start_date)
//   - CREATE INDEX idx_fiscal_period_end_date ON fiscal_period(end_date)
//   - CREATE INDEX idx_fiscal_period_active ON fiscal_period(active)
//
// TODO Phase 2: Implement GetFiscalPeriodListPageData with date range filters and journal entry counts
// TODO Phase 2: Implement GetFiscalPeriodItemPageData with journal entry summary
// TODO Phase 2: Implement CloseFiscalPeriod — validate no DRAFT entries, set status=CLOSED
type PostgresFiscalPeriodRepository struct {
	fiscalperiodpb.UnimplementedFiscalPeriodDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresFiscalPeriodRepository creates a new PostgreSQL fiscal_period repository.
func NewPostgresFiscalPeriodRepository(dbOps interfaces.DatabaseOperation, tableName string) fiscalperiodpb.FiscalPeriodDomainServiceServer {
	if tableName == "" {
		tableName = "fiscal_period"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresFiscalPeriodRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateFiscalPeriod creates a new fiscal_period using common PostgreSQL operations.
func (r *PostgresFiscalPeriodRepository) CreateFiscalPeriod(ctx context.Context, req *fiscalperiodpb.CreateFiscalPeriodRequest) (*fiscalperiodpb.CreateFiscalPeriodResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("fiscal_period data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create fiscal_period: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "start_date", "end_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	fiscalPeriod := &fiscalperiodpb.FiscalPeriod{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fiscalPeriod); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &fiscalperiodpb.CreateFiscalPeriodResponse{
		Data: []*fiscalperiodpb.FiscalPeriod{fiscalPeriod},
	}, nil
}

// ReadFiscalPeriod retrieves a fiscal_period by ID using common PostgreSQL operations.
func (r *PostgresFiscalPeriodRepository) ReadFiscalPeriod(ctx context.Context, req *fiscalperiodpb.ReadFiscalPeriodRequest) (*fiscalperiodpb.ReadFiscalPeriodResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fiscal_period ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read fiscal_period: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "start_date", "end_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	fiscalPeriod := &fiscalperiodpb.FiscalPeriod{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fiscalPeriod); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &fiscalperiodpb.ReadFiscalPeriodResponse{
		Data: []*fiscalperiodpb.FiscalPeriod{fiscalPeriod},
	}, nil
}

// UpdateFiscalPeriod updates a fiscal_period using common PostgreSQL operations.
func (r *PostgresFiscalPeriodRepository) UpdateFiscalPeriod(ctx context.Context, req *fiscalperiodpb.UpdateFiscalPeriodRequest) (*fiscalperiodpb.UpdateFiscalPeriodResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fiscal_period ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update fiscal_period: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "start_date", "end_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	fiscalPeriod := &fiscalperiodpb.FiscalPeriod{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fiscalPeriod); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &fiscalperiodpb.UpdateFiscalPeriodResponse{
		Data: []*fiscalperiodpb.FiscalPeriod{fiscalPeriod},
	}, nil
}

// DeleteFiscalPeriod soft-deletes a fiscal_period using common PostgreSQL operations.
func (r *PostgresFiscalPeriodRepository) DeleteFiscalPeriod(ctx context.Context, req *fiscalperiodpb.DeleteFiscalPeriodRequest) (*fiscalperiodpb.DeleteFiscalPeriodResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fiscal_period ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete fiscal_period: %w", err)
	}

	return &fiscalperiodpb.DeleteFiscalPeriodResponse{
		Success: true,
	}, nil
}

// ListFiscalPeriods lists fiscal_periods using common PostgreSQL operations.
func (r *PostgresFiscalPeriodRepository) ListFiscalPeriods(ctx context.Context, req *fiscalperiodpb.ListFiscalPeriodsRequest) (*fiscalperiodpb.ListFiscalPeriodsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list fiscal_periods: %w", err)
	}

	var fiscalPeriods []*fiscalperiodpb.FiscalPeriod
	for _, result := range listResult.Data {
		postgresCore.ConvertMillisToDateStr(result, "start_date", "end_date")
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		fiscalPeriod := &fiscalperiodpb.FiscalPeriod{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fiscalPeriod); err != nil {
			continue
		}
		fiscalPeriods = append(fiscalPeriods, fiscalPeriod)
	}

	return &fiscalperiodpb.ListFiscalPeriodsResponse{
		Data: fiscalPeriods,
	}, nil
}

// GetFiscalPeriodListPageData - TODO Phase 2: CTE with journal entry count per period, date range filter.
func (r *PostgresFiscalPeriodRepository) GetFiscalPeriodListPageData(ctx context.Context, req *fiscalperiodpb.GetFiscalPeriodListPageDataRequest) (*fiscalperiodpb.GetFiscalPeriodListPageDataResponse, error) {
	// TODO Phase 2: CTE with COUNT(journal_entry), status filter, sort by start_date DESC
	return nil, fmt.Errorf("GetFiscalPeriodListPageData not yet implemented — Phase 2")
}

// GetFiscalPeriodItemPageData - TODO Phase 2: implement with journal entry summary statistics.
func (r *PostgresFiscalPeriodRepository) GetFiscalPeriodItemPageData(ctx context.Context, req *fiscalperiodpb.GetFiscalPeriodItemPageDataRequest) (*fiscalperiodpb.GetFiscalPeriodItemPageDataResponse, error) {
	// TODO Phase 2: fetch period + total debits/credits, entry count by status
	return nil, fmt.Errorf("GetFiscalPeriodItemPageData not yet implemented — Phase 2")
}

// CloseFiscalPeriod - TODO Phase 2: validate no DRAFT entries exist, set status=CLOSED.
func (r *PostgresFiscalPeriodRepository) CloseFiscalPeriod(ctx context.Context, req *fiscalperiodpb.CloseFiscalPeriodRequest) (*fiscalperiodpb.CloseFiscalPeriodResponse, error) {
	// TODO Phase 2: check COUNT(journal_entry WHERE status=DRAFT AND fiscal_period_id=?) = 0,
	// then UPDATE fiscal_period SET status='closed' WHERE id=?
	return nil, fmt.Errorf("CloseFiscalPeriod not yet implemented — Phase 2")
}
