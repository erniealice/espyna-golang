//go:build postgresql

package expenditure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.AccruedExpenseSettlement, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres accrued_expense_settlement repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresAccruedExpenseSettlementRepository(dbOps, tableName), nil
	})
}

// PostgresAccruedExpenseSettlementRepository implements the join-table CRUD between
// AccruedExpense and Expenditure.
//
// HIGH-3: this is the canonical settlement model — N:M between AccruedExpense and
// Expenditure with per-row amount, currency, FX, and reversal tracking. Replaces
// the original single `settling_expenditure_id` FK design.
type PostgresAccruedExpenseSettlementRepository struct {
	accruedexpensepb.UnimplementedAccruedExpenseSettlementDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresAccruedExpenseSettlementRepository creates a new PostgreSQL settlement repository.
func NewPostgresAccruedExpenseSettlementRepository(dbOps interfaces.DatabaseOperation, tableName string) accruedexpensepb.AccruedExpenseSettlementDomainServiceServer {
	if tableName == "" {
		tableName = "accrued_expense_settlement"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresAccruedExpenseSettlementRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateAccruedExpenseSettlement creates a new settlement row.
func (r *PostgresAccruedExpenseSettlementRepository) CreateAccruedExpenseSettlement(ctx context.Context, req *accruedexpensepb.CreateAccruedExpenseSettlementRequest) (*accruedexpensepb.CreateAccruedExpenseSettlementResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("accrued expense settlement data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create accrued_expense_settlement: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &accruedexpensepb.AccruedExpenseSettlement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &accruedexpensepb.CreateAccruedExpenseSettlementResponse{Success: true, Data: []*accruedexpensepb.AccruedExpenseSettlement{row}}, nil
}

// ReadAccruedExpenseSettlement retrieves a settlement by ID.
func (r *PostgresAccruedExpenseSettlementRepository) ReadAccruedExpenseSettlement(ctx context.Context, req *accruedexpensepb.ReadAccruedExpenseSettlementRequest) (*accruedexpensepb.ReadAccruedExpenseSettlementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("accrued expense settlement ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read accrued_expense_settlement: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &accruedexpensepb.AccruedExpenseSettlement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &accruedexpensepb.ReadAccruedExpenseSettlementResponse{Success: true, Data: []*accruedexpensepb.AccruedExpenseSettlement{row}}, nil
}

// UpdateAccruedExpenseSettlement updates a settlement row.
func (r *PostgresAccruedExpenseSettlementRepository) UpdateAccruedExpenseSettlement(ctx context.Context, req *accruedexpensepb.UpdateAccruedExpenseSettlementRequest) (*accruedexpensepb.UpdateAccruedExpenseSettlementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("accrued expense settlement ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update accrued_expense_settlement: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &accruedexpensepb.AccruedExpenseSettlement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &accruedexpensepb.UpdateAccruedExpenseSettlementResponse{Success: true, Data: []*accruedexpensepb.AccruedExpenseSettlement{row}}, nil
}

// DeleteAccruedExpenseSettlement soft-deletes a settlement row.
func (r *PostgresAccruedExpenseSettlementRepository) DeleteAccruedExpenseSettlement(ctx context.Context, req *accruedexpensepb.DeleteAccruedExpenseSettlementRequest) (*accruedexpensepb.DeleteAccruedExpenseSettlementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("accrued expense settlement ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete accrued_expense_settlement: %w", err)
	}
	return &accruedexpensepb.DeleteAccruedExpenseSettlementResponse{Success: true}, nil
}

// ListAccruedExpenseSettlements lists settlements with optional filters.
func (r *PostgresAccruedExpenseSettlementRepository) ListAccruedExpenseSettlements(ctx context.Context, req *accruedexpensepb.ListAccruedExpenseSettlementsRequest) (*accruedexpensepb.ListAccruedExpenseSettlementsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list accrued_expense_settlements: %w", err)
	}
	var rows []*accruedexpensepb.AccruedExpenseSettlement
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal accrued_expense_settlement row: %v", err)
			continue
		}
		row := &accruedexpensepb.AccruedExpenseSettlement{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
			log.Printf("WARN: protojson unmarshal accrued_expense_settlement: %v", err)
			continue
		}
		rows = append(rows, row)
	}
	return &accruedexpensepb.ListAccruedExpenseSettlementsResponse{Success: true, Data: rows}, nil
}

// GetAccruedExpenseSettlementListPageData returns a paginated list page.
func (r *PostgresAccruedExpenseSettlementRepository) GetAccruedExpenseSettlementListPageData(ctx context.Context, req *accruedexpensepb.GetAccruedExpenseSettlementListPageDataRequest) (*accruedexpensepb.GetAccruedExpenseSettlementListPageDataResponse, error) {
	listResp, err := r.ListAccruedExpenseSettlements(ctx, &accruedexpensepb.ListAccruedExpenseSettlementsRequest{Filters: req.GetFilters(), Pagination: req.GetPagination(), Sort: req.GetSort(), Search: req.GetSearch()})
	if err != nil {
		return nil, err
	}
	return &accruedexpensepb.GetAccruedExpenseSettlementListPageDataResponse{
		AccruedExpenseSettlementList: listResp.Data,
		Success:                      true,
	}, nil
}

// GetAccruedExpenseSettlementItemPageData returns a single settlement row.
func (r *PostgresAccruedExpenseSettlementRepository) GetAccruedExpenseSettlementItemPageData(ctx context.Context, req *accruedexpensepb.GetAccruedExpenseSettlementItemPageDataRequest) (*accruedexpensepb.GetAccruedExpenseSettlementItemPageDataResponse, error) {
	if req == nil || req.GetAccruedExpenseSettlementId() == "" {
		return nil, fmt.Errorf("accrued expense settlement ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetAccruedExpenseSettlementId())
	if err != nil {
		return nil, fmt.Errorf("failed to read accrued_expense_settlement item: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &accruedexpensepb.AccruedExpenseSettlement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &accruedexpensepb.GetAccruedExpenseSettlementItemPageDataResponse{
		AccruedExpenseSettlement: row,
		Success:                  true,
	}, nil
}
