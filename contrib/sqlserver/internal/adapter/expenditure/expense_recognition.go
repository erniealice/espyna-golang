//go:build sqlserver

package expenditure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.ExpenseRecognition, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver expense_recognition repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerExpenseRecognitionRepository(db, dbOps, tableName), nil
	})
}

// SQLServerExpenseRecognitionRepository implements expense recognition CRUD
// using SQL Server.
//
// Idempotency: ExpenseRecognition rows carry a stable `idempotency_key` column
// with a status-independent unique index. Recurrence-engine and
// manual-recognition callers race-resolve at the use-case layer.
//
// 20260517 — `advance_disbursement_id` and `run_id` columns flow through
// transparently via protojson round-trip; no explicit SELECT/scan required.
type SQLServerExpenseRecognitionRepository struct {
	expenserecognitionpb.UnimplementedExpenseRecognitionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerExpenseRecognitionRepository creates a new SQL Server expense
// recognition repository.
func NewSQLServerExpenseRecognitionRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) expenserecognitionpb.ExpenseRecognitionDomainServiceServer {
	if tableName == "" {
		tableName = "expense_recognition"
	}
	return &SQLServerExpenseRecognitionRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateExpenseRecognition creates a new recognition row.
func (r *SQLServerExpenseRecognitionRepository) CreateExpenseRecognition(ctx context.Context, req *expenserecognitionpb.CreateExpenseRecognitionRequest) (*expenserecognitionpb.CreateExpenseRecognitionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("expense recognition data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")
	if v, ok := data["status"].(string); ok {
		if num, ok := expenserecognitionpb.ExpenseRecognitionStatus_value[v]; ok {
			data["status"] = int32(num)
		}
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create expense_recognition: %w", err)
	}
	sqlserverCore.ConvertMillisToRFC3339(result, "recognition_date", "period_start", "period_end")
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &expenserecognitionpb.ExpenseRecognition{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &expenserecognitionpb.CreateExpenseRecognitionResponse{Success: true, Data: []*expenserecognitionpb.ExpenseRecognition{row}}, nil
}

// ReadExpenseRecognition retrieves a recognition by ID.
func (r *SQLServerExpenseRecognitionRepository) ReadExpenseRecognition(ctx context.Context, req *expenserecognitionpb.ReadExpenseRecognitionRequest) (*expenserecognitionpb.ReadExpenseRecognitionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expense recognition ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read expense_recognition: %w", err)
	}
	sqlserverCore.ConvertMillisToRFC3339(result, "recognition_date", "period_start", "period_end")
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &expenserecognitionpb.ExpenseRecognition{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &expenserecognitionpb.ReadExpenseRecognitionResponse{Success: true, Data: []*expenserecognitionpb.ExpenseRecognition{row}}, nil
}

// UpdateExpenseRecognition updates a recognition row.
func (r *SQLServerExpenseRecognitionRepository) UpdateExpenseRecognition(ctx context.Context, req *expenserecognitionpb.UpdateExpenseRecognitionRequest) (*expenserecognitionpb.UpdateExpenseRecognitionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expense recognition ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")
	if v, ok := data["status"].(string); ok {
		if num, ok := expenserecognitionpb.ExpenseRecognitionStatus_value[v]; ok {
			data["status"] = int32(num)
		}
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update expense_recognition: %w", err)
	}
	sqlserverCore.ConvertMillisToRFC3339(result, "recognition_date", "period_start", "period_end")
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &expenserecognitionpb.ExpenseRecognition{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &expenserecognitionpb.UpdateExpenseRecognitionResponse{Success: true, Data: []*expenserecognitionpb.ExpenseRecognition{row}}, nil
}

// DeleteExpenseRecognition soft-deletes a recognition row.
func (r *SQLServerExpenseRecognitionRepository) DeleteExpenseRecognition(ctx context.Context, req *expenserecognitionpb.DeleteExpenseRecognitionRequest) (*expenserecognitionpb.DeleteExpenseRecognitionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expense recognition ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete expense_recognition: %w", err)
	}
	return &expenserecognitionpb.DeleteExpenseRecognitionResponse{Success: true}, nil
}

// ListExpenseRecognitions lists recognitions with optional filters.
func (r *SQLServerExpenseRecognitionRepository) ListExpenseRecognitions(ctx context.Context, req *expenserecognitionpb.ListExpenseRecognitionsRequest) (*expenserecognitionpb.ListExpenseRecognitionsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list expense_recognitions: %w", err)
	}
	var rows []*expenserecognitionpb.ExpenseRecognition
	for _, result := range listResult.Data {
		sqlserverCore.ConvertMillisToRFC3339(result, "recognition_date", "period_start", "period_end")
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal expense_recognition row: %v", err)
			continue
		}
		row := &expenserecognitionpb.ExpenseRecognition{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
			log.Printf("WARN: protojson unmarshal expense_recognition: %v", err)
			continue
		}
		rows = append(rows, row)
	}
	return &expenserecognitionpb.ListExpenseRecognitionsResponse{Success: true, Data: rows}, nil
}

// GetExpenseRecognitionListPageData returns a paginated list page (basic CRUD form).
func (r *SQLServerExpenseRecognitionRepository) GetExpenseRecognitionListPageData(ctx context.Context, req *expenserecognitionpb.GetExpenseRecognitionListPageDataRequest) (*expenserecognitionpb.GetExpenseRecognitionListPageDataResponse, error) {
	listResp, err := r.ListExpenseRecognitions(ctx, &expenserecognitionpb.ListExpenseRecognitionsRequest{
		Filters:    req.GetFilters(),
		Pagination: req.GetPagination(),
		Sort:       req.GetSort(),
		Search:     req.GetSearch(),
	})
	if err != nil {
		return nil, err
	}
	return &expenserecognitionpb.GetExpenseRecognitionListPageDataResponse{
		ExpenseRecognitionList: listResp.Data,
		Success:                true,
	}, nil
}

// GetExpenseRecognitionItemPageData returns a single recognition for the detail page.
func (r *SQLServerExpenseRecognitionRepository) GetExpenseRecognitionItemPageData(ctx context.Context, req *expenserecognitionpb.GetExpenseRecognitionItemPageDataRequest) (*expenserecognitionpb.GetExpenseRecognitionItemPageDataResponse, error) {
	if req == nil || req.GetExpenseRecognitionId() == "" {
		return nil, fmt.Errorf("expense recognition ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetExpenseRecognitionId())
	if err != nil {
		return nil, fmt.Errorf("failed to read expense_recognition item: %w", err)
	}
	sqlserverCore.ConvertMillisToRFC3339(result, "recognition_date", "period_start", "period_end")
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &expenserecognitionpb.ExpenseRecognition{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &expenserecognitionpb.GetExpenseRecognitionItemPageDataResponse{
		ExpenseRecognition: row,
		Success:            true,
	}, nil
}

// GetUnrecognizedExpenditures lists expenditure IDs in the workspace that lack
// a linked POSTED ExpenseRecognition row. Used by the AP team review queue.
//
// SQL Server translation:
//   - $N → @pN
//   - active = true → active = 1
//   - $1::text IS NULL OR $1::text = ” → @p1 IS NULL OR @p1 = ”
//   - LIMIT 500 → TOP 500 (before ORDER BY)
func (r *SQLServerExpenseRecognitionRepository) GetUnrecognizedExpenditures(ctx context.Context, req *expenserecognitionpb.GetUnrecognizedExpendituresRequest) (*expenserecognitionpb.GetUnrecognizedExpendituresResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get unrecognized expenditures request is required")
	}

	workspaceID := ""
	if req.WorkspaceId != nil {
		workspaceID = *req.WorkspaceId
	}

	const query = `
		SELECT TOP 500 e.id
		FROM [expenditure] e
		LEFT JOIN [expense_recognition] er
		  ON er.expenditure_id = e.id
		 AND er.active = 1
		 AND er.status = 2
		WHERE e.active = 1
		  AND (@p1 IS NULL OR @p1 = '' OR e.workspace_id = @p1)
		  AND er.id IS NULL
		ORDER BY e.date_created DESC
	`
	dbRows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query unrecognized expenditures: %w", err)
	}
	defer dbRows.Close()

	var ids []string
	for dbRows.Next() {
		var id string
		if err := dbRows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan unrecognized expenditure row: %w", err)
		}
		ids = append(ids, id)
	}
	if err = dbRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating unrecognized expenditure rows: %w", err)
	}

	return &expenserecognitionpb.GetUnrecognizedExpendituresResponse{
		ExpenditureIds: ids,
		Success:        true,
	}, nil
}

// GetUnrecognizedExpendituresInPeriod returns expenditure IDs whose
// date_created falls within [periodStart, periodEnd) and which lack a POSTED
// ExpenseRecognition. Not on the proto interface; called via concrete type.
func (r *SQLServerExpenseRecognitionRepository) GetUnrecognizedExpendituresInPeriod(ctx context.Context, workspaceID string, periodStart, periodEnd time.Time) ([]string, error) {
	const query = `
		SELECT e.id
		FROM [expenditure] e
		LEFT JOIN [expense_recognition] er
		  ON er.expenditure_id = e.id
		 AND er.active = 1
		 AND er.status = 2
		WHERE e.active = 1
		  AND (@p1 IS NULL OR @p1 = '' OR e.workspace_id = @p1)
		  AND e.date_created >= @p2
		  AND e.date_created <  @p3
		  AND er.id IS NULL
		ORDER BY e.date_created DESC
	`
	dbRows, err := r.db.QueryContext(ctx, query, workspaceID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to query unrecognized expenditures in period: %w", err)
	}
	defer dbRows.Close()

	var ids []string
	for dbRows.Next() {
		var id string
		if err := dbRows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan unrecognized expenditure row: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, dbRows.Err()
}
