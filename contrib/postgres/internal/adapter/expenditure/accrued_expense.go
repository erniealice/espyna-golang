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
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.AccruedExpense, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres accrued_expense repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresAccruedExpenseRepository(dbOps, tableName), nil
	})
}

// PostgresAccruedExpenseRepository implements accrued expense CRUD using PostgreSQL.
//
// Single-write boundary discipline (plan §10 R2/R3): only the AccruedExpense
// + AccruedExpenseSettlement use cases write `settled_amount` and
// `remaining_amount`. The Expenditure use case never touches them. Settle /
// reverse paths are routed through the use case layer (settle_accrual.go /
// reverse_accrual.go); this adapter exposes raw CRUD plus ListOutstanding.
type PostgresAccruedExpenseRepository struct {
	accruedexpensepb.UnimplementedAccruedExpenseDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresAccruedExpenseRepository creates a new PostgreSQL accrued expense repository.
func NewPostgresAccruedExpenseRepository(dbOps interfaces.DatabaseOperation, tableName string) accruedexpensepb.AccruedExpenseDomainServiceServer {
	if tableName == "" {
		tableName = "accrued_expense"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresAccruedExpenseRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateAccruedExpense creates a new accrued-expense row.
func (r *PostgresAccruedExpenseRepository) CreateAccruedExpense(ctx context.Context, req *accruedexpensepb.CreateAccruedExpenseRequest) (*accruedexpensepb.CreateAccruedExpenseResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("accrued expense data is required")
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
	// status is INTEGER; protojson emitted the enum string. Convert to int32.
	if v, ok := data["status"].(string); ok {
		if num, ok := accruedexpensepb.AccruedExpenseStatus_value[v]; ok {
			data["status"] = int32(num)
		}
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create accrued_expense: %w", err)
	}
	postgresCore.ConvertMillisToRFC3339(result, "period_start", "period_end", "recognition_date", "settled_at")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &accruedexpensepb.AccruedExpense{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &accruedexpensepb.CreateAccruedExpenseResponse{Success: true, Data: []*accruedexpensepb.AccruedExpense{row}}, nil
}

// ReadAccruedExpense retrieves an accrued-expense row by ID.
func (r *PostgresAccruedExpenseRepository) ReadAccruedExpense(ctx context.Context, req *accruedexpensepb.ReadAccruedExpenseRequest) (*accruedexpensepb.ReadAccruedExpenseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("accrued expense ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read accrued_expense: %w", err)
	}
	postgresCore.ConvertMillisToRFC3339(result, "period_start", "period_end", "recognition_date", "settled_at")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &accruedexpensepb.AccruedExpense{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &accruedexpensepb.ReadAccruedExpenseResponse{Success: true, Data: []*accruedexpensepb.AccruedExpense{row}}, nil
}

// UpdateAccruedExpense updates an accrued-expense row.
func (r *PostgresAccruedExpenseRepository) UpdateAccruedExpense(ctx context.Context, req *accruedexpensepb.UpdateAccruedExpenseRequest) (*accruedexpensepb.UpdateAccruedExpenseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("accrued expense ID is required")
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
	if v, ok := data["status"].(string); ok {
		if num, ok := accruedexpensepb.AccruedExpenseStatus_value[v]; ok {
			data["status"] = int32(num)
		}
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update accrued_expense: %w", err)
	}
	postgresCore.ConvertMillisToRFC3339(result, "period_start", "period_end", "recognition_date", "settled_at")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &accruedexpensepb.AccruedExpense{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &accruedexpensepb.UpdateAccruedExpenseResponse{Success: true, Data: []*accruedexpensepb.AccruedExpense{row}}, nil
}

// DeleteAccruedExpense soft-deletes an accrued-expense row.
func (r *PostgresAccruedExpenseRepository) DeleteAccruedExpense(ctx context.Context, req *accruedexpensepb.DeleteAccruedExpenseRequest) (*accruedexpensepb.DeleteAccruedExpenseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("accrued expense ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete accrued_expense: %w", err)
	}
	return &accruedexpensepb.DeleteAccruedExpenseResponse{Success: true}, nil
}

// ListAccruedExpenses lists accrued-expenses with optional filters.
func (r *PostgresAccruedExpenseRepository) ListAccruedExpenses(ctx context.Context, req *accruedexpensepb.ListAccruedExpensesRequest) (*accruedexpensepb.ListAccruedExpensesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list accrued_expenses: %w", err)
	}
	var rows []*accruedexpensepb.AccruedExpense
	for _, result := range listResult.Data {
		postgresCore.ConvertMillisToRFC3339(result, "period_start", "period_end", "recognition_date", "settled_at")
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal accrued_expense row: %v", err)
			continue
		}
		row := &accruedexpensepb.AccruedExpense{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
			log.Printf("WARN: protojson unmarshal accrued_expense: %v", err)
			continue
		}
		rows = append(rows, row)
	}
	return &accruedexpensepb.ListAccruedExpensesResponse{Success: true, Data: rows}, nil
}

// GetAccruedExpenseListPageData returns a paginated list page (basic CRUD form).
func (r *PostgresAccruedExpenseRepository) GetAccruedExpenseListPageData(ctx context.Context, req *accruedexpensepb.GetAccruedExpenseListPageDataRequest) (*accruedexpensepb.GetAccruedExpenseListPageDataResponse, error) {
	listResp, err := r.ListAccruedExpenses(ctx, &accruedexpensepb.ListAccruedExpensesRequest{Filters: req.GetFilters(), Pagination: req.GetPagination(), Sort: req.GetSort(), Search: req.GetSearch()})
	if err != nil {
		return nil, err
	}
	return &accruedexpensepb.GetAccruedExpenseListPageDataResponse{
		AccruedExpenseList: listResp.Data,
		Success:            true,
	}, nil
}

// GetAccruedExpenseItemPageData returns a single accrued-expense row.
func (r *PostgresAccruedExpenseRepository) GetAccruedExpenseItemPageData(ctx context.Context, req *accruedexpensepb.GetAccruedExpenseItemPageDataRequest) (*accruedexpensepb.GetAccruedExpenseItemPageDataResponse, error) {
	if req == nil || req.GetAccruedExpenseId() == "" {
		return nil, fmt.Errorf("accrued expense ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetAccruedExpenseId())
	if err != nil {
		return nil, fmt.Errorf("failed to read accrued_expense item: %w", err)
	}
	postgresCore.ConvertMillisToRFC3339(result, "period_start", "period_end", "recognition_date", "settled_at")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &accruedexpensepb.AccruedExpense{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &accruedexpensepb.GetAccruedExpenseItemPageDataResponse{
		AccruedExpense: row,
		Success:        true,
	}, nil
}

// AccrueFromContract / SettleAccrual / ReverseAccrual are routed through the
// use case layer (accrue_from_contract.go, settle_accrual.go [Opus], reverse_accrual.go).
// The adapter implements the interface contract via the embedded
// UnimplementedAccruedExpenseDomainServiceServer (returning "unimplemented")
// and the use cases call lower-level CRUD adapter methods directly.

// ListOutstanding returns AccruedExpense rows whose status is OUTSTANDING (1)
// or PARTIAL (2), scoped to the given workspace. Used by the AP team review queue
// and the Procurement Operations app's "Open Accruals" surface.
func (r *PostgresAccruedExpenseRepository) ListOutstanding(ctx context.Context, workspaceID string) ([]*accruedexpensepb.AccruedExpense, error) {
	const outstanding = int32(accruedexpensepb.AccruedExpenseStatus_ACCRUED_EXPENSE_STATUS_OUTSTANDING)
	const partial = int32(accruedexpensepb.AccruedExpenseStatus_ACCRUED_EXPENSE_STATUS_PARTIAL)

	query := `
		SELECT id
		FROM accrued_expense
		WHERE active = true
		  AND ($1::text IS NULL OR $1::text = '' OR workspace_id = $1)
		  AND status IN ($2, $3)
		ORDER BY recognition_date DESC
		LIMIT 500
	`
	rows, err := r.db.QueryContext(ctx, query, workspaceID, outstanding, partial)
	if err != nil {
		return nil, fmt.Errorf("failed to list outstanding accruals: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan outstanding accrual row: %w", err)
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating outstanding accrual rows: %w", err)
	}

	out := make([]*accruedexpensepb.AccruedExpense, 0, len(ids))
	for _, id := range ids {
		readResp, err := r.ReadAccruedExpense(ctx, &accruedexpensepb.ReadAccruedExpenseRequest{Data: &accruedexpensepb.AccruedExpense{Id: id}})
		if err != nil {
			log.Printf("WARN: ReadAccruedExpense %s: %v", id, err)
			continue
		}
		if len(readResp.Data) == 0 {
			continue
		}
		out = append(out, readResp.Data[0])
	}
	return out, nil
}
