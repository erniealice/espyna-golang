//go:build sqlserver

package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	fiscalperiodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/fiscal_period"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.FiscalPeriod, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver fiscal_period repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerFiscalPeriodRepository(dbOps, tableName), nil
	})
}

// SQLServerFiscalPeriodRepository implements fiscal_period CRUD using SQL Server.
type SQLServerFiscalPeriodRepository struct {
	fiscalperiodpb.UnimplementedFiscalPeriodDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerFiscalPeriodRepository creates a new SQL Server fiscal_period repository.
func NewSQLServerFiscalPeriodRepository(dbOps interfaces.DatabaseOperation, tableName string) fiscalperiodpb.FiscalPeriodDomainServiceServer {
	if tableName == "" {
		tableName = "fiscal_period"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &SQLServerFiscalPeriodRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func (r *SQLServerFiscalPeriodRepository) CreateFiscalPeriod(ctx context.Context, req *fiscalperiodpb.CreateFiscalPeriodRequest) (*fiscalperiodpb.CreateFiscalPeriodResponse, error) {
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
	sqlserverCore.ConvertMillisToDateStr(result, "start_date", "end_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	fiscalPeriod := &fiscalperiodpb.FiscalPeriod{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fiscalPeriod); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &fiscalperiodpb.CreateFiscalPeriodResponse{Data: []*fiscalperiodpb.FiscalPeriod{fiscalPeriod}}, nil
}

func (r *SQLServerFiscalPeriodRepository) ReadFiscalPeriod(ctx context.Context, req *fiscalperiodpb.ReadFiscalPeriodRequest) (*fiscalperiodpb.ReadFiscalPeriodResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fiscal_period ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read fiscal_period: %w", err)
	}
	sqlserverCore.ConvertMillisToDateStr(result, "start_date", "end_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	fiscalPeriod := &fiscalperiodpb.FiscalPeriod{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fiscalPeriod); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &fiscalperiodpb.ReadFiscalPeriodResponse{Data: []*fiscalperiodpb.FiscalPeriod{fiscalPeriod}}, nil
}

func (r *SQLServerFiscalPeriodRepository) UpdateFiscalPeriod(ctx context.Context, req *fiscalperiodpb.UpdateFiscalPeriodRequest) (*fiscalperiodpb.UpdateFiscalPeriodResponse, error) {
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
	sqlserverCore.ConvertMillisToDateStr(result, "start_date", "end_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	fiscalPeriod := &fiscalperiodpb.FiscalPeriod{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fiscalPeriod); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &fiscalperiodpb.UpdateFiscalPeriodResponse{Data: []*fiscalperiodpb.FiscalPeriod{fiscalPeriod}}, nil
}

func (r *SQLServerFiscalPeriodRepository) DeleteFiscalPeriod(ctx context.Context, req *fiscalperiodpb.DeleteFiscalPeriodRequest) (*fiscalperiodpb.DeleteFiscalPeriodResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fiscal_period ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete fiscal_period: %w", err)
	}
	return &fiscalperiodpb.DeleteFiscalPeriodResponse{Success: true}, nil
}

func (r *SQLServerFiscalPeriodRepository) ListFiscalPeriods(ctx context.Context, req *fiscalperiodpb.ListFiscalPeriodsRequest) (*fiscalperiodpb.ListFiscalPeriodsResponse, error) {
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
		sqlserverCore.ConvertMillisToDateStr(result, "start_date", "end_date")
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
	return &fiscalperiodpb.ListFiscalPeriodsResponse{Data: fiscalPeriods}, nil
}

func (r *SQLServerFiscalPeriodRepository) GetFiscalPeriodListPageData(ctx context.Context, req *fiscalperiodpb.GetFiscalPeriodListPageDataRequest) (*fiscalperiodpb.GetFiscalPeriodListPageDataResponse, error) {
	return nil, fmt.Errorf("GetFiscalPeriodListPageData not yet implemented — Phase 2")
}

func (r *SQLServerFiscalPeriodRepository) GetFiscalPeriodItemPageData(ctx context.Context, req *fiscalperiodpb.GetFiscalPeriodItemPageDataRequest) (*fiscalperiodpb.GetFiscalPeriodItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetFiscalPeriodItemPageData not yet implemented — Phase 2")
}

func (r *SQLServerFiscalPeriodRepository) CloseFiscalPeriod(ctx context.Context, req *fiscalperiodpb.CloseFiscalPeriodRequest) (*fiscalperiodpb.CloseFiscalPeriodResponse, error) {
	return nil, fmt.Errorf("CloseFiscalPeriod not yet implemented — Phase 2")
}
