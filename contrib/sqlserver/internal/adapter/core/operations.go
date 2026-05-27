//go:build sqlserver

package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/erniealice/espyna-golang/consumer"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/database/model"
	"github.com/erniealice/espyna-golang/database/operations"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	"github.com/google/uuid"
	_ "github.com/microsoft/go-mssqldb"
)

// dbExecutor abstracts *sql.DB and *sql.Tx for uniform query execution.
type dbExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func init() {
	// Register database operations factory for sqlserver.
	//
	// Mirrors the postgres/mysql factories: the returned DatabaseOperation is
	// WorkspaceAware — it injects workspace_id into Create/List/Read/Update/Delete
	// whenever (a) the request context carries a workspace_id, AND (b) the target
	// table has a workspace_id column. For global/non-tenanted tables or
	// service-to-service calls without a workspace context the decorator is a
	// pass-through.
	registry.RegisterDatabaseOperationsFactory("sqlserver", func(conn any) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver: expected *sql.DB, got %T", conn)
		}
		return NewWorkspaceAwareOperations(db), nil
	})
}

// SQLServerOperations implements DatabaseOperation for Microsoft SQL Server.
//
// It mirrors the postgres gold standard (PostgresOperations) one-for-one,
// translating the dialect differences via the shared core.Dialect
// (SQLServerDialect): @pN placeholders instead of $N, square-bracket-quoted
// identifiers instead of double quotes, OFFSET/FETCH pagination (which REQUIRES
// an ORDER BY), and LIKE instead of ILIKE. Unlike MySQL, SQL Server CAN return
// the affected row inline via OUTPUT inserted.*, so Create/Update emit the row
// in a single round-trip — the T-SQL equivalent of postgres RETURNING *.
// Timestamps and the active flag are set in Go (no triggers; the trigger-based
// reflectionless path is the out-of-scope Q-REFLECT-CRUD work).
type SQLServerOperations struct {
	db           *sql.DB
	dialect      Dialect                 // shared dialect helper (SQLServerDialect)
	auditService infraports.AuditService // optional — nil = audit disabled
}

// NewSQLServerOperations creates a new SQL Server operations instance.
func NewSQLServerOperations(db *sql.DB) interfaces.DatabaseOperation {
	return &SQLServerOperations{
		db:      db,
		dialect: DefaultDialect,
	}
}

// NewSQLServerOperationsWithAudit creates a SQL Server operations instance with
// audit logging enabled. When auditSvc is non-nil, Create/Update/Delete will
// call DiffAndLog after each successful mutation.
func NewSQLServerOperationsWithAudit(db *sql.DB, auditSvc infraports.AuditService) *SQLServerOperations {
	return &SQLServerOperations{
		db:           db,
		dialect:      DefaultDialect,
		auditService: auditSvc,
	}
}

// Create creates a new record in the specified table.
//
// SQL Server returns the inserted row inline via OUTPUT inserted.*, so Create is
// a single round-trip (the RETURNING equivalent). The UUID is supplied app-side
// for parity with the other dialects.
func (s *SQLServerOperations) Create(ctx context.Context, tableName string, data map[string]any) (map[string]any, error) {
	if tableName == "" {
		return nil, model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}

	// Normalize camelCase keys to snake_case (protojson compatibility)
	data = normalizeKeys(data)

	// Get actual table columns so we can discard fields that don't exist in the
	// DB (e.g. protobuf-only fields like date_created_string).
	resultColumns, err := s.getTableColumns(ctx, tableName)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get table columns: %v", err),
			"SQLSERVER_SCHEMA_ERROR",
			500,
		)
	}
	validColumns := make(map[string]bool, len(resultColumns))
	for _, col := range resultColumns {
		validColumns[col] = true
	}

	columnTypes, err := s.getTableColumnTypes(ctx, tableName)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get table column types: %v", err),
			"SQLSERVER_SCHEMA_ERROR",
			500,
		)
	}

	// Set creation properties.
	now := time.Now().UTC()
	if existing, ok := data["id"]; !ok || existing == nil || existing == "" {
		data["id"] = generateUUID()
	}
	data["active"] = true
	data["date_created"] = autoTimestampValue(columnTypes["date_created"], now)
	data["date_modified"] = autoTimestampValue(columnTypes["date_modified"], now)

	// Build INSERT query (only columns that exist in the table).
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]any, 0, len(data))
	var skipped []string

	i := 1
	for column, value := range data {
		if !validColumns[column] {
			skipped = append(skipped, column)
			continue
		}
		columns = append(columns, s.dialect.QuoteIdent(column))
		placeholders = append(placeholders, s.dialect.Placeholder(i))
		values = append(values, serializeValue(value))
		i++
	}
	if len(skipped) > 0 {
		log.Printf("SQLServerOperations.Create: dropped %d unknown column(s) for table=%q skipped=%v", len(skipped), tableName, skipped)
	}

	// OUTPUT inserted.* returns the freshly inserted row in one round-trip (the
	// SQL Server analogue of postgres RETURNING *). It is placed between the
	// column list and the VALUES clause per T-SQL syntax.
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) OUTPUT inserted.* VALUES (%s)",
		s.dialect.QuoteIdent(tableName),
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	result, err := s.queryOneRow(ctx, query, values)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to create record: %v", err),
			"SQLSERVER_CREATE_FAILED",
			500,
		)
	}

	if s.auditService != nil {
		if err := infraports.DiffAndLog(ctx, s.auditService, infraports.DiffAndLogRequest{
			EntityType: tableName,
			EntityID:   fmt.Sprintf("%v", result["id"]),
			Domain:     tableName,
			Action:     1, // INSERT
			MethodName: "SQLServerOperations.Create",
			NewData:    result,
		}); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// Read retrieves a record by ID from the specified table.
func (s *SQLServerOperations) Read(ctx context.Context, tableName string, id string) (map[string]any, error) {
	if tableName == "" {
		return nil, model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}
	if id == "" {
		return nil, model.NewDatabaseError("record ID is required", "MISSING_RECORD_ID", 400)
	}

	resultColumns, err := s.getTableColumns(ctx, tableName)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get table columns: %v", err),
			"SQLSERVER_SCHEMA_ERROR",
			500,
		)
	}

	query := fmt.Sprintf(
		"SELECT * FROM %s WHERE %s = %s",
		s.dialect.QuoteIdent(tableName),
		s.dialect.QuoteIdent("id"),
		s.dialect.Placeholder(1),
	)
	row := s.getExecutor(ctx).QueryRowContext(ctx, query, id)

	result, err := s.scanRowToMap(row, resultColumns)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
		}
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to read record: %v", err),
			"SQLSERVER_READ_FAILED",
			500,
		)
	}

	return result, nil
}

// Update updates an existing record in the specified table.
//
// SQL Server returns the updated row inline via OUTPUT inserted.* (the post-image
// of the row), so Update is the existence check (SELECT) plus a single UPDATE
// round-trip — the RETURNING equivalent.
func (s *SQLServerOperations) Update(ctx context.Context, tableName string, id string, data map[string]any) (map[string]any, error) {
	if tableName == "" {
		return nil, model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}
	if id == "" {
		return nil, model.NewDatabaseError("record ID is required", "MISSING_RECORD_ID", 400)
	}

	// Normalize camelCase keys to snake_case (protojson compatibility)
	data = normalizeKeys(data)

	resultColumns, err := s.getTableColumns(ctx, tableName)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get table columns: %v", err),
			"SQLSERVER_SCHEMA_ERROR",
			500,
		)
	}
	validColumns := make(map[string]bool, len(resultColumns))
	for _, col := range resultColumns {
		validColumns[col] = true
	}

	columnTypes, err := s.getTableColumnTypes(ctx, tableName)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get table column types: %v", err),
			"SQLSERVER_SCHEMA_ERROR",
			500,
		)
	}

	// Check if record exists (query without active filter so we can update
	// inactive records too, e.g. re-activating a soft-deleted record).
	existQuery := fmt.Sprintf(
		"SELECT * FROM %s WHERE %s = %s",
		s.dialect.QuoteIdent(tableName),
		s.dialect.QuoteIdent("id"),
		s.dialect.Placeholder(1),
	)
	existRow := s.getExecutor(ctx).QueryRowContext(ctx, existQuery, id)
	existing, err := s.scanRowToMap(existRow, resultColumns)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
		}
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to read record for update: %v", err),
			"SQLSERVER_READ_FAILED",
			500,
		)
	}

	// Set update properties (column-type-aware: BIGINT timestamp columns receive
	// unix ms, DATETIME2/DATETIME columns receive time.Time).
	now := time.Now().UTC()
	data["date_modified"] = autoTimestampValue(columnTypes["date_modified"], now)

	// Preserve original creation data. scanRowToMap normalises DATETIME columns to
	// int64 unix ms for the caller, so for those columns we convert back to
	// time.Time before passing to the driver. For BIGINT columns the stored int64
	// is already the wire format the driver expects.
	if dc := existing["date_created"]; dc != nil {
		if columnTypes["date_created"] == "bigint" {
			data["date_created"] = dc
		} else if millis, ok := dc.(int64); ok {
			data["date_created"] = time.UnixMilli(millis).UTC()
		} else {
			data["date_created"] = dc
		}
	}

	// Build UPDATE query (only columns that exist in the table).
	setParts := make([]string, 0, len(data))
	values := make([]any, 0, len(data)+1)
	var skipped []string

	i := 1
	for column, value := range data {
		if column == "id" {
			continue
		}
		if !validColumns[column] {
			skipped = append(skipped, column)
			continue
		}
		setParts = append(setParts, fmt.Sprintf("%s = %s", s.dialect.QuoteIdent(column), s.dialect.Placeholder(i)))
		values = append(values, serializeValue(value))
		i++
	}
	if len(skipped) > 0 {
		log.Printf("SQLServerOperations.Update: dropped %d unknown column(s) for table=%q id=%q skipped=%v", len(skipped), tableName, id, skipped)
	}
	values = append(values, id) // Add ID as last parameter

	// No active filter — allows re-activating soft-deleted records. OUTPUT
	// inserted.* returns the post-update row image in the same round-trip (the
	// SQL Server analogue of postgres RETURNING *). The OUTPUT clause sits between
	// SET and WHERE per T-SQL syntax.
	query := fmt.Sprintf(
		"UPDATE %s SET %s OUTPUT inserted.* WHERE %s = %s",
		s.dialect.QuoteIdent(tableName),
		strings.Join(setParts, ", "),
		s.dialect.QuoteIdent("id"),
		s.dialect.Placeholder(i),
	)

	result, err := s.queryOneRow(ctx, query, values)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to update record: %v", err),
			"SQLSERVER_UPDATE_FAILED",
			500,
		)
	}

	if s.auditService != nil {
		if err := infraports.DiffAndLog(ctx, s.auditService, infraports.DiffAndLogRequest{
			EntityType: tableName,
			EntityID:   id,
			Domain:     tableName,
			Action:     2, // UPDATE
			MethodName: "SQLServerOperations.Update",
			OldData:    existing,
			NewData:    result,
		}); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// Delete deletes a record from the specified table (soft delete by default).
func (s *SQLServerOperations) Delete(ctx context.Context, tableName string, id string) error {
	if tableName == "" {
		return model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}
	if id == "" {
		return model.NewDatabaseError("record ID is required", "MISSING_RECORD_ID", 400)
	}

	// Soft delete by setting active to false. date_modified may be BIGINT unix ms
	// or DATETIME2/DATETIME depending on the entity schema; introspect.
	columnTypes, err := s.getTableColumnTypes(ctx, tableName)
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to get table column types: %v", err),
			"SQLSERVER_SCHEMA_ERROR",
			500,
		)
	}
	now := time.Now().UTC()
	// Soft-delete is idempotent: deleting an already-inactive row is not an error
	// (no active = true predicate in WHERE).
	query := fmt.Sprintf(
		"UPDATE %s SET %s = %s, %s = %s WHERE %s = %s",
		s.dialect.QuoteIdent(tableName),
		s.dialect.QuoteIdent("active"), s.dialect.BoolLiteral(false),
		s.dialect.QuoteIdent("date_modified"), s.dialect.Placeholder(1),
		s.dialect.QuoteIdent("id"), s.dialect.Placeholder(2),
	)

	result, err := s.getExecutor(ctx).ExecContext(ctx, query, autoTimestampValue(columnTypes["date_modified"], now), id)
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to delete record: %v", err),
			"SQLSERVER_DELETE_FAILED",
			500,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to get affected rows: %v", err),
			"SQLSERVER_DELETE_FAILED",
			500,
		)
	}

	if rowsAffected == 0 {
		return model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
	}

	if s.auditService != nil {
		if err := infraports.DiffAndLog(ctx, s.auditService, infraports.DiffAndLogRequest{
			EntityType: tableName,
			EntityID:   id,
			Domain:     tableName,
			Action:     3, // DELETE
			MethodName: "SQLServerOperations.Delete",
		}); err != nil {
			return err
		}
	}

	return nil
}

// HardDelete permanently deletes a record from the specified table.
//
// TODO(recycle-bin): see the postgres gold standard for the planned two-stage
// delete (move row to a shared recycle_bin table, then scheduled purge). The
// current behavior relies on FK RESTRICT as the safety net.
func (s *SQLServerOperations) HardDelete(ctx context.Context, tableName string, id string) error {
	if tableName == "" {
		return model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}
	if id == "" {
		return model.NewDatabaseError("record ID is required", "MISSING_RECORD_ID", 400)
	}

	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s = %s",
		s.dialect.QuoteIdent(tableName),
		s.dialect.QuoteIdent("id"),
		s.dialect.Placeholder(1),
	)

	result, err := s.getExecutor(ctx).ExecContext(ctx, query, id)
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to hard delete record: %v", err),
			"SQLSERVER_HARD_DELETE_FAILED",
			500,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to get affected rows: %v", err),
			"SQLSERVER_HARD_DELETE_FAILED",
			500,
		)
	}

	if rowsAffected == 0 {
		return model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
	}

	return nil
}

// List retrieves records from the specified table with standardized params.
func (s *SQLServerOperations) List(ctx context.Context, tableName string, params *interfaces.ListParams) (*interfaces.ListResult, error) {
	if tableName == "" {
		return nil, model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}

	// Build WHERE clause.
	// Default to active = true unless the caller supplies an explicit "active"
	// BooleanFilter — in that case we honour the caller's value so inactive
	// records can be retrieved.
	hasActiveFilter := false
	if params != nil && params.Filters != nil {
		for _, f := range params.Filters.Filters {
			if f.GetField() == "active" {
				if _, ok := f.FilterType.(*commonpb.TypedFilter_BooleanFilter); ok {
					hasActiveFilter = true
					break
				}
			}
		}
	}
	var whereConditions []string
	if !hasActiveFilter {
		whereConditions = []string{fmt.Sprintf("%s = %s", s.dialect.QuoteIdent("active"), s.dialect.BoolLiteral(true))}
	}
	values := []any{}
	paramIndex := 1

	// Apply filters from FilterRequest
	if params != nil && params.Filters != nil {
		filterConditions, filterValues, nextIndex := s.buildFilterConditions(params.Filters, paramIndex)
		whereConditions = append(whereConditions, filterConditions...)
		values = append(values, filterValues...)
		paramIndex = nextIndex
	}

	// Search — LIKE OR block across declared search fields. SQL Server's default
	// CI collation makes plain LIKE case-insensitive, the dialect equivalent of
	// postgres ILIKE.
	if params != nil && params.Search != nil && params.Search.Query != "" {
		q := "%" + params.Search.Query + "%"
		fields := params.Search.GetOptions().GetSearchFields()
		if len(fields) == 0 {
			return nil, model.NewDatabaseError(
				"search requires SearchOptions.search_fields",
				"MISSING_SEARCH_FIELDS",
				400,
			)
		}
		var likeClauses []string
		for _, col := range fields {
			values = append(values, q)
			likeClauses = append(likeClauses, fmt.Sprintf("%s LIKE %s", s.dialect.QuoteIdent(col), s.dialect.Placeholder(paramIndex)))
			paramIndex++
		}
		whereConditions = append(whereConditions, "("+strings.Join(likeClauses, " OR ")+")")
	}

	// Build ORDER BY clause. SQL Server OFFSET/FETCH pagination REQUIRES an
	// ORDER BY, so a deterministic default is always present.
	orderByClause := fmt.Sprintf("ORDER BY %s DESC", s.dialect.QuoteIdent("date_created"))
	if params != nil && params.Sort != nil && len(params.Sort.Fields) > 0 {
		orderByParts := make([]string, 0, len(params.Sort.Fields))
		for _, sortField := range params.Sort.Fields {
			direction := "ASC"
			if sortField.Direction == commonpb.SortDirection_DESC {
				direction = "DESC"
			}
			// SQL Server has no NULLS FIRST/LAST; NULL ordering is emulated with a
			// leading "col IS NULL" key when requested.
			switch sortField.NullOrder {
			case commonpb.NullOrder_NULLS_FIRST:
				orderByParts = append(orderByParts, fmt.Sprintf("CASE WHEN %s IS NULL THEN 0 ELSE 1 END", s.dialect.QuoteIdent(sortField.Field)))
			case commonpb.NullOrder_NULLS_LAST:
				orderByParts = append(orderByParts, fmt.Sprintf("CASE WHEN %s IS NULL THEN 1 ELSE 0 END", s.dialect.QuoteIdent(sortField.Field)))
			}
			orderByParts = append(orderByParts, fmt.Sprintf("%s %s", s.dialect.QuoteIdent(sortField.Field), direction))
		}
		orderByClause = "ORDER BY " + strings.Join(orderByParts, ", ")
	}

	// Get total count before pagination
	countQuery := fmt.Sprintf(
		"SELECT COUNT(*) FROM %s WHERE %s",
		s.dialect.QuoteIdent(tableName),
		strings.Join(whereConditions, " AND "),
	)

	var totalItems int32
	if err := s.getExecutor(ctx).QueryRowContext(ctx, countQuery, values...).Scan(&totalItems); err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to count records: %v", err),
			"SQLSERVER_COUNT_FAILED",
			500,
		)
	}

	// Apply pagination
	limit := int32(100) // Default limit
	offset := int32(0)
	if params != nil && params.Pagination != nil {
		if params.Pagination.Limit > 0 && params.Pagination.Limit <= 100 {
			limit = params.Pagination.Limit
		}
		if offsetPagination := params.Pagination.GetOffset(); offsetPagination != nil {
			if offsetPagination.Page > 0 {
				offset = (offsetPagination.Page - 1) * limit
			}
		}
	}

	// Build final query with pagination. The dialect owns the OFFSET/FETCH
	// fragment and folds the (mandatory) ORDER BY into it. limit/offset are
	// integers interpolated by the dialect, not bound parameters.
	baseQuery := fmt.Sprintf(
		"SELECT * FROM %s WHERE %s",
		s.dialect.QuoteIdent(tableName),
		strings.Join(whereConditions, " AND "),
	)
	query := s.dialect.Paginate(baseQuery, orderByClause, int(limit), int(offset))

	rows, err := s.getExecutor(ctx).QueryContext(ctx, query, values...)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to list records: %v", err),
			"SQLSERVER_LIST_FAILED",
			500,
		)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get columns: %v", err),
			"SQLSERVER_LIST_FAILED",
			500,
		)
	}

	var results []map[string]any
	for rows.Next() {
		result, err := s.scanRowsToMap(rows, columns)
		if err != nil {
			return nil, model.NewDatabaseError(
				fmt.Sprintf("failed to scan row: %v", err),
				"SQLSERVER_LIST_FAILED",
				500,
			)
		}
		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("rows iteration error: %v", err),
			"SQLSERVER_LIST_FAILED",
			500,
		)
	}

	// Build pagination response
	currentPage := int32(1)
	if offset > 0 && limit > 0 {
		currentPage = (offset / limit) + 1
	}
	totalPages := (totalItems + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}
	hasNext := currentPage < totalPages
	hasPrev := currentPage > 1

	return &interfaces.ListResult{
		Data:  results,
		Total: totalItems,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  totalItems,
			CurrentPage: &currentPage,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
	}, nil
}

// Query executes a structured query against the SQL Server table.
func (s *SQLServerOperations) Query(ctx context.Context, tableName string, queryBuilder interfaces.QueryBuilder) ([]map[string]any, error) {
	if tableName == "" {
		return nil, model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}
	if queryBuilder == nil {
		return nil, model.NewDatabaseError("query builder is required", "MISSING_QUERY_BUILDER", 400)
	}

	filter, err := queryBuilder.Build()
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to build query: %v", err),
			"QUERY_BUILD_FAILED",
			400,
		)
	}

	whereConditions := []string{}
	values := []any{}
	paramIndex := 1

	for _, condition := range filter.Conditions {
		col := s.dialect.QuoteIdent(condition.Field)
		switch condition.Operator {
		case "==":
			whereConditions = append(whereConditions, fmt.Sprintf("%s = %s", col, s.dialect.Placeholder(paramIndex)))
			values = append(values, condition.Value)
			paramIndex++
		case "!=":
			whereConditions = append(whereConditions, fmt.Sprintf("%s != %s", col, s.dialect.Placeholder(paramIndex)))
			values = append(values, condition.Value)
			paramIndex++
		case "in":
			if valueSlice, ok := condition.Value.([]any); ok && len(valueSlice) > 0 {
				placeholders := make([]string, len(valueSlice))
				for i, val := range valueSlice {
					placeholders[i] = s.dialect.Placeholder(paramIndex)
					values = append(values, val)
					paramIndex++
				}
				whereConditions = append(whereConditions, fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", ")))
			}
		case ">":
			whereConditions = append(whereConditions, fmt.Sprintf("%s > %s", col, s.dialect.Placeholder(paramIndex)))
			values = append(values, condition.Value)
			paramIndex++
		case "<":
			whereConditions = append(whereConditions, fmt.Sprintf("%s < %s", col, s.dialect.Placeholder(paramIndex)))
			values = append(values, condition.Value)
			paramIndex++
		case ">=":
			whereConditions = append(whereConditions, fmt.Sprintf("%s >= %s", col, s.dialect.Placeholder(paramIndex)))
			values = append(values, condition.Value)
			paramIndex++
		case "<=":
			whereConditions = append(whereConditions, fmt.Sprintf("%s <= %s", col, s.dialect.Placeholder(paramIndex)))
			values = append(values, condition.Value)
			paramIndex++
		case "LIKE":
			whereConditions = append(whereConditions, fmt.Sprintf("%s LIKE %s", col, s.dialect.Placeholder(paramIndex)))
			values = append(values, condition.Value)
			paramIndex++
		default:
			return nil, model.NewDatabaseError(
				fmt.Sprintf("unsupported operator: %s", condition.Operator),
				"UNSUPPORTED_OPERATOR",
				400,
			)
		}
	}

	query := fmt.Sprintf("SELECT * FROM %s", s.dialect.QuoteIdent(tableName))

	if len(whereConditions) > 0 {
		query += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Build ORDER BY clause. A deterministic order is required when a row limit
	// is applied (TOP), so a default sort is always emitted.
	orderByClause := fmt.Sprintf("ORDER BY %s DESC", s.dialect.QuoteIdent("date_created"))
	if len(filter.OrderBy) > 0 {
		orderParts := make([]string, len(filter.OrderBy))
		for i, orderBy := range filter.OrderBy {
			direction := "ASC"
			if !orderBy.Ascending {
				direction = "DESC"
			}
			orderParts[i] = fmt.Sprintf("%s %s", s.dialect.QuoteIdent(orderBy.Field), direction)
		}
		orderByClause = "ORDER BY " + strings.Join(orderParts, ", ")
	}

	// SQL Server has no LIMIT clause. A bare row cap (no offset) uses OFFSET 0
	// ROWS FETCH NEXT n ROWS ONLY via the dialect, which also folds in the
	// mandatory ORDER BY. Limit is an author/builder-controlled integer (mirrors
	// the postgres gold standard's direct interpolation).
	if filter.Limit > 0 {
		query = s.dialect.Paginate(query, orderByClause, filter.Limit, 0)
	} else {
		query += " " + orderByClause
	}

	rows, err := s.getExecutor(ctx).QueryContext(ctx, query, values...)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to execute query: %v", err),
			"SQLSERVER_QUERY_FAILED",
			500,
		)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get columns: %v", err),
			"SQLSERVER_QUERY_FAILED",
			500,
		)
	}

	var results []map[string]any
	for rows.Next() {
		result, err := s.scanRowsToMap(rows, columns)
		if err != nil {
			return nil, model.NewDatabaseError(
				fmt.Sprintf("failed to scan row: %v", err),
				"SQLSERVER_QUERY_FAILED",
				500,
			)
		}
		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("rows iteration error: %v", err),
			"SQLSERVER_QUERY_FAILED",
			500,
		)
	}

	return results, nil
}

// QueryOne executes a structured query and returns the first result.
func (s *SQLServerOperations) QueryOne(ctx context.Context, tableName string, queryBuilder interfaces.QueryBuilder) (map[string]any, error) {
	limitedBuilder := queryBuilder.Limit(1)
	results, err := s.Query(ctx, tableName, limitedBuilder)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, model.NewDatabaseError("no results found", "NO_RESULTS_FOUND", 404)
	}

	return results[0], nil
}

// Helper methods

// queryOneRow runs a row-returning statement (an INSERT/UPDATE with OUTPUT
// inserted.*, or any single-row SELECT) and scans the first row into a
// snake_case map. Column names are taken from the live result set so the scan
// order always matches OUTPUT inserted.* regardless of table column order.
func (s *SQLServerOperations) queryOneRow(ctx context.Context, query string, args []any) (map[string]any, error) {
	rows, err := s.getExecutor(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, sql.ErrNoRows
	}

	result, err := s.scanRowsToMap(rows, columns)
	if err != nil {
		return nil, err
	}
	return result, rows.Err()
}

// buildFilterConditions builds WHERE conditions from FilterRequest.
func (s *SQLServerOperations) buildFilterConditions(filterReq *commonpb.FilterRequest, startIndex int) ([]string, []any, int) {
	conditions := []string{}
	values := []any{}
	paramIndex := startIndex

	for _, filter := range filterReq.Filters {
		field := filter.Field

		switch ft := filter.FilterType.(type) {
		case *commonpb.TypedFilter_StringFilter:
			condition, vals, nextIndex := s.buildStringFilter(field, ft.StringFilter, paramIndex)
			conditions = append(conditions, condition)
			values = append(values, vals...)
			paramIndex = nextIndex

		case *commonpb.TypedFilter_NumberFilter:
			condition, val, nextIndex := s.buildNumberFilter(field, ft.NumberFilter, paramIndex)
			conditions = append(conditions, condition)
			values = append(values, val)
			paramIndex = nextIndex

		case *commonpb.TypedFilter_BooleanFilter:
			conditions = append(conditions, fmt.Sprintf("%s = %s", s.dialect.QuoteIdent(field), s.dialect.Placeholder(paramIndex)))
			values = append(values, ft.BooleanFilter.Value)
			paramIndex++

		case *commonpb.TypedFilter_ListFilter:
			condition, vals, nextIndex := s.buildListFilter(field, ft.ListFilter, paramIndex)
			if condition != "" {
				conditions = append(conditions, condition)
				values = append(values, vals...)
				paramIndex = nextIndex
			}

		case *commonpb.TypedFilter_RangeFilter:
			rangeConditions, vals, nextIndex := s.buildRangeFilter(field, ft.RangeFilter, paramIndex)
			conditions = append(conditions, rangeConditions...)
			values = append(values, vals...)
			paramIndex = nextIndex

		case *commonpb.TypedFilter_DateFilter:
			condition, vals, nextIndex := s.buildDateFilter(field, ft.DateFilter, paramIndex)
			if condition != "" {
				conditions = append(conditions, condition)
				values = append(values, vals...)
				paramIndex = nextIndex
			}

		case *commonpb.TypedFilter_MoneyFilter:
			mf := ft.MoneyFilter
			col := s.dialect.QuoteIdent(filter.Field)
			switch mf.Operator {
			case commonpb.MoneyOperator_MONEY_EQUALS:
				conditions = append(conditions, fmt.Sprintf("%s = %s", col, s.dialect.Placeholder(paramIndex)))
				values = append(values, mf.Amount)
				paramIndex++
			case commonpb.MoneyOperator_MONEY_LESS_THAN:
				conditions = append(conditions, fmt.Sprintf("%s < %s", col, s.dialect.Placeholder(paramIndex)))
				values = append(values, mf.Amount)
				paramIndex++
			case commonpb.MoneyOperator_MONEY_GREATER_THAN:
				conditions = append(conditions, fmt.Sprintf("%s > %s", col, s.dialect.Placeholder(paramIndex)))
				values = append(values, mf.Amount)
				paramIndex++
			case commonpb.MoneyOperator_MONEY_LESS_THAN_OR_EQUAL:
				conditions = append(conditions, fmt.Sprintf("%s <= %s", col, s.dialect.Placeholder(paramIndex)))
				values = append(values, mf.Amount)
				paramIndex++
			case commonpb.MoneyOperator_MONEY_GREATER_THAN_OR_EQUAL:
				conditions = append(conditions, fmt.Sprintf("%s >= %s", col, s.dialect.Placeholder(paramIndex)))
				values = append(values, mf.Amount)
				paramIndex++
			case commonpb.MoneyOperator_MONEY_BETWEEN:
				conditions = append(conditions, fmt.Sprintf("%s BETWEEN %s AND %s", col, s.dialect.Placeholder(paramIndex), s.dialect.Placeholder(paramIndex+1)))
				values = append(values, mf.Amount, mf.AmountTo)
				paramIndex += 2
			}

		case *commonpb.TypedFilter_StatusFilter:
			sf := ft.StatusFilter
			if len(sf.Values) > 0 {
				placeholders := make([]string, len(sf.Values))
				for i, v := range sf.Values {
					placeholders[i] = s.dialect.Placeholder(paramIndex)
					values = append(values, v)
					paramIndex++
				}
				conditions = append(conditions, fmt.Sprintf(
					"%s IN (%s)", s.dialect.QuoteIdent(filter.Field), strings.Join(placeholders, ", "),
				))
			}
		}
	}

	return conditions, values, paramIndex
}

// buildStringFilter builds a SQL condition for StringFilter.
func (s *SQLServerOperations) buildStringFilter(field string, filter *commonpb.StringFilter, paramIndex int) (string, []any, int) {
	col := s.dialect.QuoteIdent(field)
	value := filter.Value
	if !filter.CaseSensitive {
		// SQL Server's default CI collation already case-folds, but LOWER() on
		// both sides keeps parity with the postgres gold standard regardless of
		// the column's collation.
		col = fmt.Sprintf("LOWER(%s)", col)
		value = strings.ToLower(value)
	}

	var condition string
	var values []any

	switch filter.Operator {
	case commonpb.StringOperator_STRING_EQUALS:
		condition = fmt.Sprintf("%s = %s", col, s.dialect.Placeholder(paramIndex))
		values = append(values, value)
		paramIndex++
	case commonpb.StringOperator_STRING_NOT_EQUALS:
		condition = fmt.Sprintf("%s != %s", col, s.dialect.Placeholder(paramIndex))
		values = append(values, value)
		paramIndex++
	case commonpb.StringOperator_STRING_CONTAINS:
		condition = fmt.Sprintf("%s LIKE %s", col, s.dialect.Placeholder(paramIndex))
		values = append(values, "%"+value+"%")
		paramIndex++
	case commonpb.StringOperator_STRING_STARTS_WITH:
		condition = fmt.Sprintf("%s LIKE %s", col, s.dialect.Placeholder(paramIndex))
		values = append(values, value+"%")
		paramIndex++
	case commonpb.StringOperator_STRING_ENDS_WITH:
		condition = fmt.Sprintf("%s LIKE %s", col, s.dialect.Placeholder(paramIndex))
		values = append(values, "%"+value)
		paramIndex++
	case commonpb.StringOperator_STRING_REGEX:
		// SQL Server has no native regex operator; LIKE is the closest portable
		// fallback for the (rarely used) regex string operator.
		condition = fmt.Sprintf("%s LIKE %s", col, s.dialect.Placeholder(paramIndex))
		values = append(values, value)
		paramIndex++
	}

	return condition, values, paramIndex
}

// buildNumberFilter builds a SQL condition for NumberFilter.
func (s *SQLServerOperations) buildNumberFilter(field string, filter *commonpb.NumberFilter, paramIndex int) (string, any, int) {
	var operator string
	switch filter.Operator {
	case commonpb.NumberOperator_NUMBER_EQUALS:
		operator = "="
	case commonpb.NumberOperator_NUMBER_NOT_EQUALS:
		operator = "!="
	case commonpb.NumberOperator_NUMBER_GREATER_THAN:
		operator = ">"
	case commonpb.NumberOperator_NUMBER_GREATER_THAN_OR_EQUAL:
		operator = ">="
	case commonpb.NumberOperator_NUMBER_LESS_THAN:
		operator = "<"
	case commonpb.NumberOperator_NUMBER_LESS_THAN_OR_EQUAL:
		operator = "<="
	}

	condition := fmt.Sprintf("%s %s %s", s.dialect.QuoteIdent(field), operator, s.dialect.Placeholder(paramIndex))
	return condition, filter.Value, paramIndex + 1
}

// buildListFilter builds a SQL condition for ListFilter.
func (s *SQLServerOperations) buildListFilter(field string, filter *commonpb.ListFilter, paramIndex int) (string, []any, int) {
	if len(filter.Values) == 0 {
		return "", nil, paramIndex
	}

	placeholders := make([]string, len(filter.Values))
	values := make([]any, len(filter.Values))
	for i, val := range filter.Values {
		placeholders[i] = s.dialect.Placeholder(paramIndex)
		values[i] = val
		paramIndex++
	}

	col := s.dialect.QuoteIdent(field)
	var condition string
	switch filter.Operator {
	case commonpb.ListOperator_LIST_IN:
		condition = fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", "))
	case commonpb.ListOperator_LIST_NOT_IN:
		condition = fmt.Sprintf("%s NOT IN (%s)", col, strings.Join(placeholders, ", "))
	}

	return condition, values, paramIndex
}

// buildRangeFilter builds SQL conditions for RangeFilter.
func (s *SQLServerOperations) buildRangeFilter(field string, filter *commonpb.RangeFilter, paramIndex int) ([]string, []any, int) {
	conditions := []string{}
	values := []any{}
	col := s.dialect.QuoteIdent(field)

	minOp := ">"
	if filter.IncludeMin {
		minOp = ">="
	}
	conditions = append(conditions, fmt.Sprintf("%s %s %s", col, minOp, s.dialect.Placeholder(paramIndex)))
	values = append(values, filter.Min)
	paramIndex++

	maxOp := "<"
	if filter.IncludeMax {
		maxOp = "<="
	}
	conditions = append(conditions, fmt.Sprintf("%s %s %s", col, maxOp, s.dialect.Placeholder(paramIndex)))
	values = append(values, filter.Max)
	paramIndex++

	return conditions, values, paramIndex
}

// buildDateFilter builds a SQL condition for DateFilter.
//
// SQL Server has no `::date`/`::timestamp` cast syntax; CAST(... AS date) and
// CAST(... AS datetime2) are the dialect equivalents of the postgres casts.
func (s *SQLServerOperations) buildDateFilter(field string, filter *commonpb.DateFilter, paramIndex int) (string, []any, int) {
	var condition string
	values := []any{}
	col := s.dialect.QuoteIdent(field)

	switch filter.Operator {
	case commonpb.DateOperator_DATE_EQUALS:
		condition = fmt.Sprintf("CAST(%s AS date) = CAST(%s AS date)", col, s.dialect.Placeholder(paramIndex))
		values = append(values, filter.Value)
		paramIndex++
	case commonpb.DateOperator_DATE_BEFORE:
		condition = fmt.Sprintf("%s < CAST(%s AS datetime2)", col, s.dialect.Placeholder(paramIndex))
		values = append(values, filter.Value)
		paramIndex++
	case commonpb.DateOperator_DATE_AFTER:
		condition = fmt.Sprintf("%s > CAST(%s AS datetime2)", col, s.dialect.Placeholder(paramIndex))
		values = append(values, filter.Value)
		paramIndex++
	case commonpb.DateOperator_DATE_BETWEEN:
		if filter.RangeEnd != nil && *filter.RangeEnd != "" {
			condition = fmt.Sprintf("%s BETWEEN CAST(%s AS datetime2) AND CAST(%s AS datetime2)", col, s.dialect.Placeholder(paramIndex), s.dialect.Placeholder(paramIndex+1))
			values = append(values, filter.Value, *filter.RangeEnd)
			paramIndex += 2
		}
	}

	return condition, values, paramIndex
}

// getTableColumns retrieves column names for a table.
//
// SQL Server's information_schema is database-scoped to the connection, so no
// extra schema predicate is required for the common single-schema deployment.
func (s *SQLServerOperations) getTableColumns(ctx context.Context, tableName string) ([]string, error) {
	query := `
		SELECT COLUMN_NAME
		FROM information_schema.COLUMNS
		WHERE TABLE_NAME = @p1
		ORDER BY ORDINAL_POSITION
	`

	rows, err := s.getExecutor(ctx).QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			return nil, err
		}
		columns = append(columns, columnName)
	}

	return columns, rows.Err()
}

// getTableColumnTypes returns column-name → information_schema DATA_TYPE for a
// table. Used by Create/Update to pick the right serialization for
// auto-injected timestamp fields (BIGINT unix-ms vs DATETIME2/DATETIME).
func (s *SQLServerOperations) getTableColumnTypes(ctx context.Context, tableName string) (map[string]string, error) {
	query := `
		SELECT COLUMN_NAME, DATA_TYPE
		FROM information_schema.COLUMNS
		WHERE TABLE_NAME = @p1
	`
	rows, err := s.getExecutor(ctx).QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	types := make(map[string]string)
	for rows.Next() {
		var name, dataType string
		if err := rows.Scan(&name, &dataType); err != nil {
			return nil, err
		}
		types[name] = dataType
	}
	return types, rows.Err()
}

// autoTimestampValue returns the appropriate value to write for a timestamp
// column at creation/update time. BIGINT columns (the proto-aligned convention)
// receive unix ms; DATETIME2/DATETIME columns receive a time.Time for the driver.
func autoTimestampValue(columnType string, now time.Time) any {
	if columnType == "bigint" {
		return now.UnixMilli()
	}
	return now
}

// scanRowToMap scans a single row into a map with snake_case keys (matching DB columns).
func (s *SQLServerOperations) scanRowToMap(row *sql.Row, columns []string) (map[string]any, error) {
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))

	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := row.Scan(valuePtrs...); err != nil {
		return nil, err
	}

	result := make(map[string]any)
	for i, column := range columns {
		result[column] = normalizeValue(values[i])
	}

	return result, nil
}

// scanRowsToMap scans a single row from *sql.Rows into a map with snake_case keys.
func (s *SQLServerOperations) scanRowsToMap(rows *sql.Rows, columns []string) (map[string]any, error) {
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))

	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, err
	}

	result := make(map[string]any)
	for i, column := range columns {
		result[column] = normalizeValue(values[i])
	}

	return result, nil
}

// ConvertMillisToDateStr converts business date fields in a result map from
// int64 Unix millis (produced by normalizeValue) to ISO 8601 date strings
// ("YYYY-MM-DD"). Call this on the map returned by Read/List BEFORE
// json.Marshal + protojson.Unmarshal.
func ConvertMillisToDateStr(data map[string]any, keys ...string) {
	for _, key := range keys {
		v, ok := data[key]
		if !ok || v == nil {
			continue
		}
		switch val := v.(type) {
		case int64:
			if val > 0 {
				data[key] = time.UnixMilli(val).UTC().Format("2006-01-02")
			}
		case float64:
			if val > 0 {
				data[key] = time.UnixMilli(int64(val)).UTC().Format("2006-01-02")
			}
		case string:
			// Already a date string — leave as-is
		}
	}
}

// ConvertMillisToRFC3339 converts timestamp fields in a result map from int64
// Unix millis (produced by normalizeValue) to RFC3339 strings, the format
// protojson expects for google.protobuf.Timestamp fields. Call this on the map
// returned by Read/List BEFORE json.Marshal + protojson.Unmarshal.
func ConvertMillisToRFC3339(data map[string]any, keys ...string) {
	for _, key := range keys {
		v, ok := data[key]
		if !ok || v == nil {
			continue
		}
		switch val := v.(type) {
		case int64:
			if val > 0 {
				data[key] = time.UnixMilli(val).UTC().Format(time.RFC3339Nano)
			}
		case float64:
			if val > 0 {
				data[key] = time.UnixMilli(int64(val)).UTC().Format(time.RFC3339Nano)
			}
		case string:
			// Already a string — leave as-is (assume RFC3339)
		}
	}
}

// normalizeValue converts DB-native types to protobuf-compatible types.
// time.Time (from DATETIME2/DATETIME) → int64 Unix millis, so protojson can
// unmarshal into int64 protobuf fields. JSON-bearing string/byte columns are
// unmarshalled to native Go types.
func normalizeValue(v any) any {
	switch t := v.(type) {
	case time.Time:
		if t.IsZero() {
			return nil
		}
		return t.UnixMilli()
	case []byte:
		// JSON columns: unmarshal to native Go types so json.Marshal produces
		// proper JSON instead of base64-encoded strings. If the bytes are not a
		// JSON object/array, fall back to the string form.
		var parsed any
		if err := json.Unmarshal(t, &parsed); err == nil {
			switch parsed.(type) {
			case map[string]any, []any:
				return parsed
			}
		}
		return string(t)
	default:
		return v
	}
}

// generateUUID generates an application-side UUID. SQL Server can generate ids
// with NEWID(), but app-side generation keeps parity across dialects and lets
// callers supply their own id.
func generateUUID() string {
	return uuid.NewString()
}

// RunWithTransaction executes a function within a database transaction.
func (s *SQLServerOperations) RunWithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to begin transaction: %v", err),
			"SQLSERVER_TRANSACTION_FAILED",
			500,
		)
	}

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return model.NewDatabaseError(
				fmt.Sprintf("transaction failed and rollback failed: %v, %v", err, rollbackErr),
				"SQLSERVER_TRANSACTION_ROLLBACK_FAILED",
				500,
			)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to commit transaction: %v", err),
			"SQLSERVER_TRANSACTION_COMMIT_FAILED",
			500,
		)
	}

	return nil
}

// WithTransaction returns a DatabaseOperation that routes all queries through
// the transaction stored in ctx. Implements interfaces.TransactionAware.
func (s *SQLServerOperations) WithTransaction(ctx context.Context) interfaces.DatabaseOperation {
	return s
}

// SupportsTransactions implements interfaces.TransactionAware.
func (s *SQLServerOperations) SupportsTransactions() bool {
	return true
}

// GetDB returns the underlying database connection for raw-SQL repositories.
func (s *SQLServerOperations) GetDB() *sql.DB {
	return s.db
}

// getExecutor returns the transaction's *sql.Tx if a pending transaction is
// active in ctx, otherwise *sql.DB. The transaction type (added by a later
// transactions.go) is reached via a small structural interface so this package
// does not depend on a not-yet-extracted concrete type; when none is present the
// assertion fails and we fall back to the pooled *sql.DB.
func (s *SQLServerOperations) getExecutor(ctx context.Context) dbExecutor {
	if tx, ok := operations.GetTransactionFromContext(ctx); ok {
		type txExecutor interface {
			GetTx() *sql.Tx
			State() interfaces.TransactionState
		}
		if sqlTx, ok := tx.(txExecutor); ok && sqlTx.State() == interfaces.TransactionStatePending {
			return sqlTx.GetTx()
		}
	}
	return s.db
}

// GetExecutor returns *sql.Tx if one is active in ctx, otherwise *sql.DB.
// Entity adapters that build raw SQL (CTEs, JOINs) must call this instead of
// holding their own *sql.DB reference. The return type uses the shared
// interfaces.DBExecutor so adapter packages can type-assert without each package
// defining its own copy.
func (s *SQLServerOperations) GetExecutor(ctx context.Context) interfaces.DBExecutor {
	return s.getExecutor(ctx)
}

// serializeValue converts map and slice values to JSON bytes so the SQL driver
// can store them in JSON/NVARCHAR columns. Primitive types pass through.
func serializeValue(v any) any {
	switch v.(type) {
	case map[string]any, []any:
		b, err := json.Marshal(v)
		if err != nil {
			return v
		}
		return b
	default:
		return v
	}
}

// normalizeKeys converts all map keys from camelCase to snake_case so that
// protojson-marshaled data (camelCase) maps to SQL Server column names
// (snake_case).
func normalizeKeys(data map[string]any) map[string]any {
	result := make(map[string]any, len(data))
	for key, value := range data {
		result[camelToSnake(key)] = value
	}
	return result
}

// camelToSnake converts camelCase to snake_case.
func camelToSnake(str string) string {
	var result []rune
	for i, r := range str {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		if r >= 'A' && r <= 'Z' {
			result = append(result, r-'A'+'a')
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

// ── WorkspaceAwareOperations ─────────────────────────────────────────────────

// WorkspaceAwareOperations decorates a DatabaseOperation with automatic
// workspace_id isolation derived from the request context.
//
// It mirrors contrib/postgres/internal/adapter/core.WorkspaceAwareOperations
// with SQL Server schema introspection (information_schema, which SQL Server
// supports). The column-existence cache is populated lazily on first access.
type WorkspaceAwareOperations struct {
	inner         interfaces.DatabaseOperation
	db            *sql.DB
	columnCache   map[string]map[string]bool // table → column → exists
	columnCacheMu sync.RWMutex
}

// Ensure WorkspaceAwareOperations satisfies the full DatabaseOperation interface
// at compile time.
var _ interfaces.DatabaseOperation = (*WorkspaceAwareOperations)(nil)

// NewWorkspaceAwareOperations returns a workspace-scoped DatabaseOperation backed
// by a fresh SQLServerOperations instance.
func NewWorkspaceAwareOperations(db *sql.DB) interfaces.DatabaseOperation {
	return &WorkspaceAwareOperations{
		inner:       NewSQLServerOperations(db),
		db:          db,
		columnCache: make(map[string]map[string]bool),
	}
}

// NewWorkspaceAwareOperationsFromInner wraps an existing DatabaseOperation with
// workspace-aware filtering.
func NewWorkspaceAwareOperationsFromInner(db *sql.DB, inner interfaces.DatabaseOperation) interfaces.DatabaseOperation {
	return &WorkspaceAwareOperations{
		inner:       inner,
		db:          db,
		columnCache: make(map[string]map[string]bool),
	}
}

// ── DatabaseOperation methods ────────────────────────────────────────────────

func (w *WorkspaceAwareOperations) List(ctx context.Context, tableName string, params *interfaces.ListParams) (*interfaces.ListResult, error) {
	wsID := w.getWorkspaceID(ctx)
	if wsID != "" && w.tableHasWorkspaceColumn(ctx, tableName) {
		params = w.injectWorkspaceFilter(params, wsID)
	}
	return w.inner.List(ctx, tableName, params)
}

func (w *WorkspaceAwareOperations) Create(ctx context.Context, tableName string, data map[string]any) (map[string]any, error) {
	wsID := w.getWorkspaceID(ctx)
	if wsID != "" && w.tableHasWorkspaceColumn(ctx, tableName) {
		cloned := make(map[string]any, len(data)+1)
		for k, v := range data {
			cloned[k] = v
		}
		cloned["workspace_id"] = wsID
		data = cloned
	}
	return w.inner.Create(ctx, tableName, data)
}

func (w *WorkspaceAwareOperations) Read(ctx context.Context, tableName string, id string) (map[string]any, error) {
	result, err := w.inner.Read(ctx, tableName, id)
	if err != nil {
		return nil, err
	}

	wsID := w.getWorkspaceID(ctx)
	if wsID == "" || !w.tableHasWorkspaceColumn(ctx, tableName) {
		return result, nil
	}

	recordWsID, hasCol := result["workspace_id"]
	if !hasCol {
		return result, nil
	}

	if recordWsID == nil {
		return nil, model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
	}
	if s, ok := recordWsID.(string); ok && (s == "" || s != wsID) {
		return nil, model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
	}
	return result, nil
}

func (w *WorkspaceAwareOperations) Update(ctx context.Context, tableName string, id string, data map[string]any) (map[string]any, error) {
	wsID := w.getWorkspaceID(ctx)
	if wsID != "" && w.tableHasWorkspaceColumn(ctx, tableName) {
		if _, err := w.Read(ctx, tableName, id); err != nil {
			return nil, err
		}
		cloned := make(map[string]any, len(data))
		for k, v := range data {
			if k != "workspace_id" {
				cloned[k] = v
			}
		}
		data = cloned
	}
	return w.inner.Update(ctx, tableName, id, data)
}

func (w *WorkspaceAwareOperations) Delete(ctx context.Context, tableName string, id string) error {
	wsID := w.getWorkspaceID(ctx)
	if wsID != "" && w.tableHasWorkspaceColumn(ctx, tableName) {
		if _, err := w.Read(ctx, tableName, id); err != nil {
			return err
		}
	}
	return w.inner.Delete(ctx, tableName, id)
}

func (w *WorkspaceAwareOperations) HardDelete(ctx context.Context, tableName string, id string) error {
	wsID := w.getWorkspaceID(ctx)
	if wsID != "" && w.tableHasWorkspaceColumn(ctx, tableName) {
		if _, err := w.Read(ctx, tableName, id); err != nil {
			return err
		}
	}
	return w.inner.HardDelete(ctx, tableName, id)
}

func (w *WorkspaceAwareOperations) Query(ctx context.Context, tableName string, query interfaces.QueryBuilder) ([]map[string]any, error) {
	return w.inner.Query(ctx, tableName, query)
}

func (w *WorkspaceAwareOperations) QueryOne(ctx context.Context, tableName string, query interfaces.QueryBuilder) (map[string]any, error) {
	return w.inner.QueryOne(ctx, tableName, query)
}

// ── Optional interface methods ───────────────────────────────────────────────

// GetDB returns the underlying *sql.DB.
func (w *WorkspaceAwareOperations) GetDB() *sql.DB { return w.db }

// GetExecutor returns the transaction-aware executor from the inner operation.
// Entity adapters that type-assert to executorProvider use this to participate
// in active transactions.
func (w *WorkspaceAwareOperations) GetExecutor(ctx context.Context) interfaces.DBExecutor {
	type executorProvider interface {
		GetExecutor(ctx context.Context) interfaces.DBExecutor
	}
	if ep, ok := w.inner.(executorProvider); ok {
		return ep.GetExecutor(ctx)
	}
	return w.db
}

// ── Helper methods ───────────────────────────────────────────────────────────

func (w *WorkspaceAwareOperations) getWorkspaceID(ctx context.Context) string {
	return consumer.GetWorkspaceIDFromContext(ctx)
}

// tableHasWorkspaceColumn reports whether tableName has a workspace_id column.
// Results are cached; the first miss queries information_schema.COLUMNS (which
// SQL Server supports).
func (w *WorkspaceAwareOperations) tableHasWorkspaceColumn(ctx context.Context, tableName string) bool {
	w.columnCacheMu.RLock()
	cols, cached := w.columnCache[tableName]
	w.columnCacheMu.RUnlock()
	if cached {
		return cols["workspace_id"]
	}

	query := `
		SELECT COLUMN_NAME
		FROM information_schema.COLUMNS
		WHERE TABLE_NAME = @p1
		ORDER BY ORDINAL_POSITION
	`
	rows, err := w.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return false
	}
	defer rows.Close()

	colMap := make(map[string]bool)
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			continue
		}
		colMap[col] = true
	}
	if rows.Err() != nil {
		return false
	}

	w.columnCacheMu.Lock()
	w.columnCache[tableName] = colMap
	w.columnCacheMu.Unlock()

	return colMap["workspace_id"]
}

// injectWorkspaceFilter returns a copy of params with a workspace_id StringFilter
// prepended. The original params value is never mutated.
func (w *WorkspaceAwareOperations) injectWorkspaceFilter(params *interfaces.ListParams, wsID string) *interfaces.ListParams {
	wsFilter := &commonpb.TypedFilter{
		Field: "workspace_id",
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{
				Value:         wsID,
				Operator:      commonpb.StringOperator_STRING_EQUALS,
				CaseSensitive: true,
			},
		},
	}

	if params == nil {
		return &interfaces.ListParams{
			Filters: &commonpb.FilterRequest{
				Filters: []*commonpb.TypedFilter{wsFilter},
			},
		}
	}

	cloned := *params
	if cloned.Filters == nil {
		cloned.Filters = &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{wsFilter},
		}
	} else {
		newFilters := make([]*commonpb.TypedFilter, 0, len(cloned.Filters.Filters)+1)
		newFilters = append(newFilters, wsFilter)
		newFilters = append(newFilters, cloned.Filters.Filters...)
		cloned.Filters = &commonpb.FilterRequest{
			Filters: newFilters,
			Logic:   cloned.Filters.Logic,
		}
	}
	return &cloned
}

// ── Key helpers ──────────────────────────────────────────────────────────────

// DenormalizeKeys converts snake_case DB column names to camelCase protojson
// field names for protobuf unmarshalling. Exported for use by entity adapters
// that convert DB results to protobuf.
func DenormalizeKeys(data map[string]any) map[string]any {
	result := make(map[string]any, len(data))
	for key, value := range data {
		result[snakeToCamel(key)] = value
	}
	return result
}

// snakeToCamel converts a snake_case string to lowerCamelCase.
func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}
