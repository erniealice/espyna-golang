//go:build postgresql

package expenditure

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
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ExpenseRecognition, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres expense_recognition repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresExpenseRecognitionRepository(dbOps, tableName), nil
	})
}

// PostgresExpenseRecognitionRepository implements expense recognition CRUD using PostgreSQL.
//
// Idempotency: ExpenseRecognition rows carry a stable `idempotency_key` column with
// a status-independent unique index (migration 20260430140200). Recurrence-engine
// and manual-recognition callers race-resolve via INSERT ... ON CONFLICT
// (idempotency_key) DO NOTHING RETURNING id at the use-case layer.
type PostgresExpenseRecognitionRepository struct {
	expenserecognitionpb.UnimplementedExpenseRecognitionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresExpenseRecognitionRepository creates a new PostgreSQL expense recognition repository.
func NewPostgresExpenseRecognitionRepository(dbOps interfaces.DatabaseOperation, tableName string) expenserecognitionpb.ExpenseRecognitionDomainServiceServer {
	if tableName == "" {
		tableName = "expense_recognition"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresExpenseRecognitionRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateExpenseRecognition creates a new recognition row.
func (r *PostgresExpenseRecognitionRepository) CreateExpenseRecognition(ctx context.Context, req *expenserecognitionpb.CreateExpenseRecognitionRequest) (*expenserecognitionpb.CreateExpenseRecognitionResponse, error) {
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
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")
	if v, ok := data["status"].(string); ok {
		if num, ok := expenserecognitionpb.ExpenseRecognitionStatus_value[v]; ok {
			data["status"] = int32(num)
		}
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create expense_recognition: %w", err)
	}
	postgresCore.ConvertMillisToRFC3339(result, "recognition_date", "period_start", "period_end")
	resultJSON, err := json.Marshal(result)
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
func (r *PostgresExpenseRecognitionRepository) ReadExpenseRecognition(ctx context.Context, req *expenserecognitionpb.ReadExpenseRecognitionRequest) (*expenserecognitionpb.ReadExpenseRecognitionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expense recognition ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read expense_recognition: %w", err)
	}
	postgresCore.ConvertMillisToRFC3339(result, "recognition_date", "period_start", "period_end")
	resultJSON, err := json.Marshal(result)
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
func (r *PostgresExpenseRecognitionRepository) UpdateExpenseRecognition(ctx context.Context, req *expenserecognitionpb.UpdateExpenseRecognitionRequest) (*expenserecognitionpb.UpdateExpenseRecognitionResponse, error) {
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
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")
	if v, ok := data["status"].(string); ok {
		if num, ok := expenserecognitionpb.ExpenseRecognitionStatus_value[v]; ok {
			data["status"] = int32(num)
		}
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update expense_recognition: %w", err)
	}
	postgresCore.ConvertMillisToRFC3339(result, "recognition_date", "period_start", "period_end")
	resultJSON, err := json.Marshal(result)
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
func (r *PostgresExpenseRecognitionRepository) DeleteExpenseRecognition(ctx context.Context, req *expenserecognitionpb.DeleteExpenseRecognitionRequest) (*expenserecognitionpb.DeleteExpenseRecognitionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expense recognition ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete expense_recognition: %w", err)
	}
	return &expenserecognitionpb.DeleteExpenseRecognitionResponse{Success: true}, nil
}

// ListExpenseRecognitions lists recognitions with optional filters.
func (r *PostgresExpenseRecognitionRepository) ListExpenseRecognitions(ctx context.Context, req *expenserecognitionpb.ListExpenseRecognitionsRequest) (*expenserecognitionpb.ListExpenseRecognitionsResponse, error) {
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
		postgresCore.ConvertMillisToRFC3339(result, "recognition_date", "period_start", "period_end")
		resultJSON, err := json.Marshal(result)
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
func (r *PostgresExpenseRecognitionRepository) GetExpenseRecognitionListPageData(ctx context.Context, req *expenserecognitionpb.GetExpenseRecognitionListPageDataRequest) (*expenserecognitionpb.GetExpenseRecognitionListPageDataResponse, error) {
	listResp, err := r.ListExpenseRecognitions(ctx, &expenserecognitionpb.ListExpenseRecognitionsRequest{Filters: req.GetFilters(), Pagination: req.GetPagination(), Sort: req.GetSort(), Search: req.GetSearch()})
	if err != nil {
		return nil, err
	}
	return &expenserecognitionpb.GetExpenseRecognitionListPageDataResponse{
		ExpenseRecognitionList: listResp.Data,
		Success:                true,
	}, nil
}

// GetExpenseRecognitionItemPageData returns a single recognition for the detail page.
func (r *PostgresExpenseRecognitionRepository) GetExpenseRecognitionItemPageData(ctx context.Context, req *expenserecognitionpb.GetExpenseRecognitionItemPageDataRequest) (*expenserecognitionpb.GetExpenseRecognitionItemPageDataResponse, error) {
	if req == nil || req.GetExpenseRecognitionId() == "" {
		return nil, fmt.Errorf("expense recognition ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetExpenseRecognitionId())
	if err != nil {
		return nil, fmt.Errorf("failed to read expense_recognition item: %w", err)
	}
	postgresCore.ConvertMillisToRFC3339(result, "recognition_date", "period_start", "period_end")
	resultJSON, err := json.Marshal(result)
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

// RecognizeFromExpenditure / RecognizeFromContract / Reverse are routed through
// the use case layer (recognize_from_expenditure.go, recognize_from_contract.go,
// reverse.go). The adapter implements the interface contract via the
// embedded UnimplementedExpenseRecognitionDomainServiceServer (returning
// "not implemented") and use cases call the lower-level CRUD adapter methods
// directly. That keeps the recognition orchestration in a single place.
//
// GetUnrecognizedExpenditures is implemented at the adapter layer because it
// is a raw query.

// GetUnrecognizedExpenditures lists expenditure IDs in the period that lack a
// linked POSTED ExpenseRecognition row. Used by the AP team review queue.
func (r *PostgresExpenseRecognitionRepository) GetUnrecognizedExpenditures(ctx context.Context, req *expenserecognitionpb.GetUnrecognizedExpendituresRequest) (*expenserecognitionpb.GetUnrecognizedExpendituresResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get unrecognized expenditures request is required")
	}

	workspaceID := ""
	if req.WorkspaceId != nil {
		workspaceID = *req.WorkspaceId
	}

	// PeriodStart / PeriodEnd are surfaced via filters today; until the period
	// fields land on the request, treat all unposted-recognition expenditures as
	// candidates. Workspace scoping is honored.
	query := `
		SELECT e.id
		FROM expenditure e
		LEFT JOIN expense_recognition er
		  ON er.expenditure_id = e.id
		 AND er.active = true
		 AND er.status = 2  -- POSTED
		WHERE e.active = true
		  AND ($1::text IS NULL OR $1::text = '' OR e.workspace_id = $1)
		  AND er.id IS NULL
		ORDER BY e.date_created DESC
		LIMIT 500
	`
	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query unrecognized expenditures: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan unrecognized expenditure row: %w", err)
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating unrecognized expenditure rows: %w", err)
	}

	return &expenserecognitionpb.GetUnrecognizedExpendituresResponse{
		ExpenditureIds: ids,
		Success:        true,
	}, nil
}

// GetUnrecognizedExpendituresInPeriod is a use-case-level helper that bypasses
// proto type drift. Returns expenditure IDs whose date_created falls within
// [periodStart, periodEnd) and which lack a POSTED ExpenseRecognition.
//
// Exposed for the use case but does not appear on the proto interface.
func (r *PostgresExpenseRecognitionRepository) GetUnrecognizedExpendituresInPeriod(ctx context.Context, workspaceID string, periodStart, periodEnd time.Time) ([]string, error) {
	query := `
		SELECT e.id
		FROM expenditure e
		LEFT JOIN expense_recognition er
		  ON er.expenditure_id = e.id
		 AND er.active = true
		 AND er.status = 2  -- POSTED
		WHERE e.active = true
		  AND ($1::text IS NULL OR $1::text = '' OR e.workspace_id = $1)
		  AND e.date_created >= $2
		  AND e.date_created <  $3
		  AND er.id IS NULL
		ORDER BY e.date_created DESC
	`
	rows, err := r.db.QueryContext(ctx, query, workspaceID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to query unrecognized expenditures in period: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan unrecognized expenditure row: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
