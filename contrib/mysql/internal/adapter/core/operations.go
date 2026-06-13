//go:build mysql

package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/database/model"
	sqlexec "github.com/erniealice/espyna-golang/database/sqlexec"
	"github.com/erniealice/espyna-golang/database/operations"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

// dbExecutor abstracts *sql.DB and *sql.Tx for uniform query execution.
type dbExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func init() {
	// Register database operations factory for mysql.
	//
	// Mirrors the postgres factory: the returned DatabaseOperation is
	// WorkspaceAware (MY-1 provides NewWorkspaceAwareOperations in this
	// package) — it injects workspace_id into Create/List/Read/Update/Delete
	// whenever (a) the request context carries a workspace_id, AND (b) the
	// target table has a workspace_id column. For global/non-tenanted tables
	// or service-to-service calls without a workspace context the decorator is
	// a pass-through.
	registry.RegisterDatabaseOperationsFactory("mysql", func(conn any) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql: expected *sql.DB, got %T", conn)
		}
		return NewWorkspaceAwareOperations(db), nil
	})
}

// MySQLOperations implements DatabaseOperation for MySQL.
//
// It mirrors the postgres gold standard (PostgresOperations) one-for-one,
// translating the dialect differences via the shared core.Dialect (owned by
// MY-1): positional `?` placeholders instead of `$N`, backtick-quoted
// identifiers instead of double quotes, and — most importantly — MySQL has no
// RETURNING clause, so Create/Update perform the mutation then SELECT the row
// back by id. Timestamps and the active flag are set in Go (no triggers; the
// trigger-based reflectionless path is the out-of-scope Q-REFLECT-CRUD work).
type MySQLOperations struct {
	db           *sql.DB
	dialect      Dialect                 // shared dialect helper (MY-1)
	auditService infraports.AuditService // optional — nil = audit disabled
}

// NewMySQLOperations creates a new MySQL operations instance.
func NewMySQLOperations(db *sql.DB) interfaces.DatabaseOperation {
	return &MySQLOperations{
		db:      db,
		dialect: NewMySQLDialect(),
	}
}

// NewMySQLOperationsWithAudit creates a MySQL operations instance with audit
// logging enabled. When auditSvc is non-nil, Create/Update/Delete will call
// DiffAndLog after each successful mutation.
func NewMySQLOperationsWithAudit(db *sql.DB, auditSvc infraports.AuditService) *MySQLOperations {
	return &MySQLOperations{
		db:           db,
		dialect:      NewMySQLDialect(),
		auditService: auditSvc,
	}
}

// Create creates a new record in the specified table.
//
// MySQL has no RETURNING, so the flow is: generate (or honour) a UUID app-side,
// INSERT, then SELECT the row back by id to return the canonical persisted form.
func (m *MySQLOperations) Create(ctx context.Context, tableName string, data map[string]any) (map[string]any, error) {
	if tableName == "" {
		return nil, model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}

	// Normalize camelCase keys to snake_case (protojson compatibility)
	data = normalizeKeys(data)

	// Get actual table columns so we can discard fields that don't exist in the
	// DB (e.g. protobuf-only fields like date_created_string).
	resultColumns, err := m.getTableColumns(ctx, tableName)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get table columns: %v", err),
			"MYSQL_SCHEMA_ERROR",
			500,
		)
	}
	validColumns := make(map[string]bool, len(resultColumns))
	for _, col := range resultColumns {
		validColumns[col] = true
	}

	columnTypes, err := m.getTableColumnTypes(ctx, tableName)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get table column types: %v", err),
			"MYSQL_SCHEMA_ERROR",
			500,
		)
	}

	// Set creation properties. The id is required up front because there is no
	// RETURNING to surface a DB-generated key — we SELECT back by this id.
	now := time.Now().UTC()
	if existing, ok := data["id"]; !ok || existing == nil || existing == "" {
		data["id"] = generateUUID()
	}
	id := fmt.Sprintf("%v", data["id"])
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
		columns = append(columns, m.dialect.QuoteIdent(column))
		placeholders = append(placeholders, m.dialect.Placeholder(i))
		values = append(values, serializeValue(value))
		i++
	}
	if len(skipped) > 0 {
		log.Printf("MySQLOperations.Create: dropped %d unknown column(s) for table=%q skipped=%v", len(skipped), tableName, skipped)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		m.dialect.QuoteIdent(tableName),
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	if _, err := m.getExecutor(ctx).ExecContext(ctx, query, values...); err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to create record: %v", err),
			"MYSQL_CREATE_FAILED",
			500,
		)
	}

	// No RETURNING — SELECT the row back by id to produce the canonical result.
	result, err := m.readByID(ctx, tableName, id, resultColumns)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to read created record: %v", err),
			"MYSQL_CREATE_FAILED",
			500,
		)
	}

	if m.auditService != nil {
		if err := infraports.DiffAndLog(ctx, m.auditService, infraports.DiffAndLogRequest{
			EntityType: tableName,
			EntityID:   fmt.Sprintf("%v", result["id"]),
			Domain:     tableName,
			Action:     1, // INSERT
			MethodName: "MySQLOperations.Create",
			NewData:    result,
		}); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// Read retrieves a record by ID from the specified table.
func (m *MySQLOperations) Read(ctx context.Context, tableName string, id string) (map[string]any, error) {
	if tableName == "" {
		return nil, model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}
	if id == "" {
		return nil, model.NewDatabaseError("record ID is required", "MISSING_RECORD_ID", 400)
	}

	resultColumns, err := m.getTableColumns(ctx, tableName)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get table columns: %v", err),
			"MYSQL_SCHEMA_ERROR",
			500,
		)
	}

	result, err := m.readByID(ctx, tableName, id, resultColumns)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
		}
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to read record: %v", err),
			"MYSQL_READ_FAILED",
			500,
		)
	}

	return result, nil
}

// Update updates an existing record in the specified table.
//
// MySQL has no RETURNING, so the flow is: existence check (SELECT), UPDATE,
// then SELECT the row back by id to return the canonical persisted form.
func (m *MySQLOperations) Update(ctx context.Context, tableName string, id string, data map[string]any) (map[string]any, error) {
	if tableName == "" {
		return nil, model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}
	if id == "" {
		return nil, model.NewDatabaseError("record ID is required", "MISSING_RECORD_ID", 400)
	}

	// Normalize camelCase keys to snake_case (protojson compatibility)
	data = normalizeKeys(data)

	resultColumns, err := m.getTableColumns(ctx, tableName)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get table columns: %v", err),
			"MYSQL_SCHEMA_ERROR",
			500,
		)
	}
	validColumns := make(map[string]bool, len(resultColumns))
	for _, col := range resultColumns {
		validColumns[col] = true
	}

	columnTypes, err := m.getTableColumnTypes(ctx, tableName)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get table column types: %v", err),
			"MYSQL_SCHEMA_ERROR",
			500,
		)
	}

	// Check if record exists (query without active filter so we can update
	// inactive records too, e.g. re-activating a soft-deleted record).
	existing, err := m.readByID(ctx, tableName, id, resultColumns)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
		}
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to read record for update: %v", err),
			"MYSQL_READ_FAILED",
			500,
		)
	}

	// Set update properties (column-type-aware: BIGINT timestamp columns
	// receive unix ms, DATETIME/TIMESTAMP columns receive time.Time).
	now := time.Now().UTC()
	data["date_modified"] = autoTimestampValue(columnTypes["date_modified"], now)

	// Preserve original creation data. readByID normalises DATETIME/TIMESTAMP
	// columns to int64 unix ms for the caller, so for those columns we convert
	// back to time.Time before passing to the driver. For BIGINT columns the
	// stored int64 is already the wire format the driver expects.
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
		setParts = append(setParts, fmt.Sprintf("%s = %s", m.dialect.QuoteIdent(column), m.dialect.Placeholder(i)))
		values = append(values, serializeValue(value))
		i++
	}
	if len(skipped) > 0 {
		log.Printf("MySQLOperations.Update: dropped %d unknown column(s) for table=%q id=%q skipped=%v", len(skipped), tableName, id, skipped)
	}
	values = append(values, id) // Add ID as last parameter

	// No active filter — allows re-activating soft-deleted records.
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = %s",
		m.dialect.QuoteIdent(tableName),
		strings.Join(setParts, ", "),
		m.dialect.QuoteIdent("id"),
		m.dialect.Placeholder(i),
	)

	if _, err := m.getExecutor(ctx).ExecContext(ctx, query, values...); err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to update record: %v", err),
			"MYSQL_UPDATE_FAILED",
			500,
		)
	}

	// No RETURNING — SELECT the row back by id to produce the canonical result.
	result, err := m.readByID(ctx, tableName, id, resultColumns)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to read updated record: %v", err),
			"MYSQL_UPDATE_FAILED",
			500,
		)
	}

	if m.auditService != nil {
		if err := infraports.DiffAndLog(ctx, m.auditService, infraports.DiffAndLogRequest{
			EntityType: tableName,
			EntityID:   id,
			Domain:     tableName,
			Action:     2, // UPDATE
			MethodName: "MySQLOperations.Update",
			OldData:    existing,
			NewData:    result,
		}); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// Delete deletes a record from the specified table (soft delete by default).
func (m *MySQLOperations) Delete(ctx context.Context, tableName string, id string) error {
	if tableName == "" {
		return model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}
	if id == "" {
		return model.NewDatabaseError("record ID is required", "MISSING_RECORD_ID", 400)
	}

	// Soft delete by setting active to false. date_modified may be BIGINT
	// unix ms or DATETIME/TIMESTAMP depending on the entity schema; introspect.
	columnTypes, err := m.getTableColumnTypes(ctx, tableName)
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to get table column types: %v", err),
			"MYSQL_SCHEMA_ERROR",
			500,
		)
	}
	now := time.Now().UTC()
	// Soft-delete is idempotent: deleting an already-inactive row is not an
	// error (no active = true predicate in WHERE).
	query := fmt.Sprintf(
		"UPDATE %s SET %s = %s, %s = %s WHERE %s = %s",
		m.dialect.QuoteIdent(tableName),
		m.dialect.QuoteIdent("active"), m.dialect.BoolLiteral(false),
		m.dialect.QuoteIdent("date_modified"), m.dialect.Placeholder(1),
		m.dialect.QuoteIdent("id"), m.dialect.Placeholder(2),
	)

	result, err := m.getExecutor(ctx).ExecContext(ctx, query, autoTimestampValue(columnTypes["date_modified"], now), id)
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to delete record: %v", err),
			"MYSQL_DELETE_FAILED",
			500,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to get affected rows: %v", err),
			"MYSQL_DELETE_FAILED",
			500,
		)
	}

	if rowsAffected == 0 {
		return model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
	}

	if m.auditService != nil {
		if err := infraports.DiffAndLog(ctx, m.auditService, infraports.DiffAndLogRequest{
			EntityType: tableName,
			EntityID:   id,
			Domain:     tableName,
			Action:     3, // DELETE
			MethodName: "MySQLOperations.Delete",
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
func (m *MySQLOperations) HardDelete(ctx context.Context, tableName string, id string) error {
	if tableName == "" {
		return model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}
	if id == "" {
		return model.NewDatabaseError("record ID is required", "MISSING_RECORD_ID", 400)
	}

	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s = %s",
		m.dialect.QuoteIdent(tableName),
		m.dialect.QuoteIdent("id"),
		m.dialect.Placeholder(1),
	)

	result, err := m.getExecutor(ctx).ExecContext(ctx, query, id)
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to hard delete record: %v", err),
			"MYSQL_HARD_DELETE_FAILED",
			500,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to get affected rows: %v", err),
			"MYSQL_HARD_DELETE_FAILED",
			500,
		)
	}

	if rowsAffected == 0 {
		return model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
	}

	return nil
}

// List retrieves records from the specified table with standardized params.
func (m *MySQLOperations) List(ctx context.Context, tableName string, params *interfaces.ListParams) (*interfaces.ListResult, error) {
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
		whereConditions = []string{fmt.Sprintf("%s = %s", m.dialect.QuoteIdent("active"), m.dialect.BoolLiteral(true))}
	}
	values := []any{}
	paramIndex := 1

	// Apply filters from FilterRequest
	if params != nil && params.Filters != nil {
		filterConditions, filterValues, nextIndex := m.buildFilterConditions(params.Filters, paramIndex)
		whereConditions = append(whereConditions, filterConditions...)
		values = append(values, filterValues...)
		paramIndex = nextIndex
	}

	// Search — LIKE OR block across declared search fields. MySQL string
	// columns use a case-insensitive collation by default, so plain LIKE is the
	// dialect equivalent of postgres ILIKE.
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
			likeClauses = append(likeClauses, fmt.Sprintf("%s LIKE %s", m.dialect.QuoteIdent(col), m.dialect.Placeholder(paramIndex)))
			paramIndex++
		}
		whereConditions = append(whereConditions, "("+strings.Join(likeClauses, " OR ")+")")
	}

	// Build ORDER BY clause
	orderByClause := fmt.Sprintf("ORDER BY %s DESC", m.dialect.QuoteIdent("date_created")) // Default ordering
	if params != nil && params.Sort != nil && len(params.Sort.Fields) > 0 {
		orderByParts := make([]string, 0, len(params.Sort.Fields))
		for _, sortField := range params.Sort.Fields {
			direction := "ASC"
			if sortField.Direction == commonpb.SortDirection_DESC {
				direction = "DESC"
			}
			// MySQL has no NULLS FIRST/LAST; NULL ordering is emulated with a
			// leading "col IS NULL" key when requested.
			switch sortField.NullOrder {
			case commonpb.NullOrder_NULLS_FIRST:
				orderByParts = append(orderByParts, fmt.Sprintf("%s IS NOT NULL", m.dialect.QuoteIdent(sortField.Field)))
			case commonpb.NullOrder_NULLS_LAST:
				orderByParts = append(orderByParts, fmt.Sprintf("%s IS NULL", m.dialect.QuoteIdent(sortField.Field)))
			}
			orderByParts = append(orderByParts, fmt.Sprintf("%s %s", m.dialect.QuoteIdent(sortField.Field), direction))
		}
		orderByClause = "ORDER BY " + strings.Join(orderByParts, ", ")
	}

	// Get total count before pagination
	countQuery := fmt.Sprintf(
		"SELECT COUNT(*) FROM %s WHERE %s",
		m.dialect.QuoteIdent(tableName),
		strings.Join(whereConditions, " AND "),
	)

	var totalItems int32
	if err := m.getExecutor(ctx).QueryRowContext(ctx, countQuery, values...).Scan(&totalItems); err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to count records: %v", err),
			"MYSQL_COUNT_FAILED",
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

	// Build final query with pagination. The dialect owns the LIMIT/OFFSET
	// fragment (MySQL: `LIMIT n OFFSET m`); limit/offset are bound as values to
	// match the postgres gold standard's parameterized pagination.
	baseQuery := fmt.Sprintf(
		"SELECT * FROM %s WHERE %s",
		m.dialect.QuoteIdent(tableName),
		strings.Join(whereConditions, " AND "),
	)
	query := m.dialect.Paginate(baseQuery, orderByClause, int(limit), int(offset))

	rows, err := m.getExecutor(ctx).QueryContext(ctx, query, values...)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to list records: %v", err),
			"MYSQL_LIST_FAILED",
			500,
		)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get columns: %v", err),
			"MYSQL_LIST_FAILED",
			500,
		)
	}

	var results []map[string]any
	for rows.Next() {
		result, err := m.scanRowsToMap(rows, columns)
		if err != nil {
			return nil, model.NewDatabaseError(
				fmt.Sprintf("failed to scan row: %v", err),
				"MYSQL_LIST_FAILED",
				500,
			)
		}
		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("rows iteration error: %v", err),
			"MYSQL_LIST_FAILED",
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

// Query executes a structured query against the MySQL table.
func (m *MySQLOperations) Query(ctx context.Context, tableName string, queryBuilder interfaces.QueryBuilder) ([]map[string]any, error) {
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
		col := m.dialect.QuoteIdent(condition.Field)
		switch condition.Operator {
		case "==":
			whereConditions = append(whereConditions, fmt.Sprintf("%s = %s", col, m.dialect.Placeholder(paramIndex)))
			values = append(values, condition.Value)
			paramIndex++
		case "!=":
			whereConditions = append(whereConditions, fmt.Sprintf("%s != %s", col, m.dialect.Placeholder(paramIndex)))
			values = append(values, condition.Value)
			paramIndex++
		case "in":
			if valueSlice, ok := condition.Value.([]any); ok && len(valueSlice) > 0 {
				placeholders := make([]string, len(valueSlice))
				for i, val := range valueSlice {
					placeholders[i] = m.dialect.Placeholder(paramIndex)
					values = append(values, val)
					paramIndex++
				}
				whereConditions = append(whereConditions, fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", ")))
			}
		case ">":
			whereConditions = append(whereConditions, fmt.Sprintf("%s > %s", col, m.dialect.Placeholder(paramIndex)))
			values = append(values, condition.Value)
			paramIndex++
		case "<":
			whereConditions = append(whereConditions, fmt.Sprintf("%s < %s", col, m.dialect.Placeholder(paramIndex)))
			values = append(values, condition.Value)
			paramIndex++
		case ">=":
			whereConditions = append(whereConditions, fmt.Sprintf("%s >= %s", col, m.dialect.Placeholder(paramIndex)))
			values = append(values, condition.Value)
			paramIndex++
		case "<=":
			whereConditions = append(whereConditions, fmt.Sprintf("%s <= %s", col, m.dialect.Placeholder(paramIndex)))
			values = append(values, condition.Value)
			paramIndex++
		case "LIKE":
			whereConditions = append(whereConditions, fmt.Sprintf("%s LIKE %s", col, m.dialect.Placeholder(paramIndex)))
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

	query := fmt.Sprintf("SELECT * FROM %s", m.dialect.QuoteIdent(tableName))

	if len(whereConditions) > 0 {
		query += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Add ORDER BY clause
	if len(filter.OrderBy) > 0 {
		orderParts := make([]string, len(filter.OrderBy))
		for i, orderBy := range filter.OrderBy {
			direction := "ASC"
			if !orderBy.Ascending {
				direction = "DESC"
			}
			orderParts[i] = fmt.Sprintf("%s %s", m.dialect.QuoteIdent(orderBy.Field), direction)
		}
		query += " ORDER BY " + strings.Join(orderParts, ", ")
	} else {
		query += fmt.Sprintf(" ORDER BY %s DESC", m.dialect.QuoteIdent("date_created"))
	}

	// Add LIMIT clause. Limit is an author/builder-controlled integer, so it is
	// interpolated directly (mirrors the postgres gold standard).
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	rows, err := m.getExecutor(ctx).QueryContext(ctx, query, values...)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to execute query: %v", err),
			"MYSQL_QUERY_FAILED",
			500,
		)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get columns: %v", err),
			"MYSQL_QUERY_FAILED",
			500,
		)
	}

	var results []map[string]any
	for rows.Next() {
		result, err := m.scanRowsToMap(rows, columns)
		if err != nil {
			return nil, model.NewDatabaseError(
				fmt.Sprintf("failed to scan row: %v", err),
				"MYSQL_QUERY_FAILED",
				500,
			)
		}
		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("rows iteration error: %v", err),
			"MYSQL_QUERY_FAILED",
			500,
		)
	}

	return results, nil
}

// QueryOne executes a structured query and returns the first result.
func (m *MySQLOperations) QueryOne(ctx context.Context, tableName string, queryBuilder interfaces.QueryBuilder) (map[string]any, error) {
	limitedBuilder := queryBuilder.Limit(1)
	results, err := m.Query(ctx, tableName, limitedBuilder)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, model.NewDatabaseError("no results found", "NO_RESULTS_FOUND", 404)
	}

	return results[0], nil
}

// Helper methods

// readByID fetches a single row by id and scans it into a snake_case map.
// This is the no-RETURNING substitute used by Create/Update (and Read): after a
// mutation, MySQL cannot return the affected row inline, so we SELECT it back.
func (m *MySQLOperations) readByID(ctx context.Context, tableName, id string, columns []string) (map[string]any, error) {
	query := fmt.Sprintf(
		"SELECT * FROM %s WHERE %s = %s",
		m.dialect.QuoteIdent(tableName),
		m.dialect.QuoteIdent("id"),
		m.dialect.Placeholder(1),
	)
	row := m.getExecutor(ctx).QueryRowContext(ctx, query, id)
	return m.scanRowToMap(row, columns)
}

// buildFilterConditions builds WHERE conditions from FilterRequest.
func (m *MySQLOperations) buildFilterConditions(filterReq *commonpb.FilterRequest, startIndex int) ([]string, []any, int) {
	conditions := []string{}
	values := []any{}
	paramIndex := startIndex

	for _, filter := range filterReq.Filters {
		field := filter.Field

		switch ft := filter.FilterType.(type) {
		case *commonpb.TypedFilter_StringFilter:
			condition, vals, nextIndex := m.buildStringFilter(field, ft.StringFilter, paramIndex)
			conditions = append(conditions, condition)
			values = append(values, vals...)
			paramIndex = nextIndex

		case *commonpb.TypedFilter_NumberFilter:
			condition, val, nextIndex := m.buildNumberFilter(field, ft.NumberFilter, paramIndex)
			conditions = append(conditions, condition)
			values = append(values, val)
			paramIndex = nextIndex

		case *commonpb.TypedFilter_BooleanFilter:
			conditions = append(conditions, fmt.Sprintf("%s = %s", m.dialect.QuoteIdent(field), m.dialect.Placeholder(paramIndex)))
			values = append(values, ft.BooleanFilter.Value)
			paramIndex++

		case *commonpb.TypedFilter_ListFilter:
			condition, vals, nextIndex := m.buildListFilter(field, ft.ListFilter, paramIndex)
			if condition != "" {
				conditions = append(conditions, condition)
				values = append(values, vals...)
				paramIndex = nextIndex
			}

		case *commonpb.TypedFilter_RangeFilter:
			rangeConditions, vals, nextIndex := m.buildRangeFilter(field, ft.RangeFilter, paramIndex)
			conditions = append(conditions, rangeConditions...)
			values = append(values, vals...)
			paramIndex = nextIndex

		case *commonpb.TypedFilter_DateFilter:
			condition, vals, nextIndex := m.buildDateFilter(field, ft.DateFilter, paramIndex)
			if condition != "" {
				conditions = append(conditions, condition)
				values = append(values, vals...)
				paramIndex = nextIndex
			}

		case *commonpb.TypedFilter_MoneyFilter:
			mf := ft.MoneyFilter
			col := m.dialect.QuoteIdent(filter.Field)
			switch mf.Operator {
			case commonpb.MoneyOperator_MONEY_EQUALS:
				conditions = append(conditions, fmt.Sprintf("%s = %s", col, m.dialect.Placeholder(paramIndex)))
				values = append(values, mf.Amount)
				paramIndex++
			case commonpb.MoneyOperator_MONEY_LESS_THAN:
				conditions = append(conditions, fmt.Sprintf("%s < %s", col, m.dialect.Placeholder(paramIndex)))
				values = append(values, mf.Amount)
				paramIndex++
			case commonpb.MoneyOperator_MONEY_GREATER_THAN:
				conditions = append(conditions, fmt.Sprintf("%s > %s", col, m.dialect.Placeholder(paramIndex)))
				values = append(values, mf.Amount)
				paramIndex++
			case commonpb.MoneyOperator_MONEY_LESS_THAN_OR_EQUAL:
				conditions = append(conditions, fmt.Sprintf("%s <= %s", col, m.dialect.Placeholder(paramIndex)))
				values = append(values, mf.Amount)
				paramIndex++
			case commonpb.MoneyOperator_MONEY_GREATER_THAN_OR_EQUAL:
				conditions = append(conditions, fmt.Sprintf("%s >= %s", col, m.dialect.Placeholder(paramIndex)))
				values = append(values, mf.Amount)
				paramIndex++
			case commonpb.MoneyOperator_MONEY_BETWEEN:
				conditions = append(conditions, fmt.Sprintf("%s BETWEEN %s AND %s", col, m.dialect.Placeholder(paramIndex), m.dialect.Placeholder(paramIndex+1)))
				values = append(values, mf.Amount, mf.AmountTo)
				paramIndex += 2
			}

		case *commonpb.TypedFilter_StatusFilter:
			sf := ft.StatusFilter
			if len(sf.Values) > 0 {
				placeholders := make([]string, len(sf.Values))
				for i, v := range sf.Values {
					placeholders[i] = m.dialect.Placeholder(paramIndex)
					values = append(values, v)
					paramIndex++
				}
				conditions = append(conditions, fmt.Sprintf(
					"%s IN (%s)", m.dialect.QuoteIdent(filter.Field), strings.Join(placeholders, ", "),
				))
			}
		}
	}

	return conditions, values, paramIndex
}

// buildStringFilter builds a SQL condition for StringFilter.
func (m *MySQLOperations) buildStringFilter(field string, filter *commonpb.StringFilter, paramIndex int) (string, []any, int) {
	col := m.dialect.QuoteIdent(field)
	value := filter.Value
	if !filter.CaseSensitive {
		// MySQL string columns use a case-insensitive collation by default, so
		// LOWER() on both sides keeps parity with the postgres gold standard
		// without depending on the column's collation.
		col = fmt.Sprintf("LOWER(%s)", col)
		value = strings.ToLower(value)
	}

	var condition string
	var values []any

	switch filter.Operator {
	case commonpb.StringOperator_STRING_EQUALS:
		condition = fmt.Sprintf("%s = %s", col, m.dialect.Placeholder(paramIndex))
		values = append(values, value)
		paramIndex++
	case commonpb.StringOperator_STRING_NOT_EQUALS:
		condition = fmt.Sprintf("%s != %s", col, m.dialect.Placeholder(paramIndex))
		values = append(values, value)
		paramIndex++
	case commonpb.StringOperator_STRING_CONTAINS:
		condition = fmt.Sprintf("%s LIKE %s", col, m.dialect.Placeholder(paramIndex))
		values = append(values, "%"+value+"%")
		paramIndex++
	case commonpb.StringOperator_STRING_STARTS_WITH:
		condition = fmt.Sprintf("%s LIKE %s", col, m.dialect.Placeholder(paramIndex))
		values = append(values, value+"%")
		paramIndex++
	case commonpb.StringOperator_STRING_ENDS_WITH:
		condition = fmt.Sprintf("%s LIKE %s", col, m.dialect.Placeholder(paramIndex))
		values = append(values, "%"+value)
		paramIndex++
	case commonpb.StringOperator_STRING_REGEX:
		// MySQL uses REGEXP rather than the postgres ~ operator.
		condition = fmt.Sprintf("%s REGEXP %s", col, m.dialect.Placeholder(paramIndex))
		values = append(values, value)
		paramIndex++
	}

	return condition, values, paramIndex
}

// buildNumberFilter builds a SQL condition for NumberFilter.
func (m *MySQLOperations) buildNumberFilter(field string, filter *commonpb.NumberFilter, paramIndex int) (string, any, int) {
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

	condition := fmt.Sprintf("%s %s %s", m.dialect.QuoteIdent(field), operator, m.dialect.Placeholder(paramIndex))
	return condition, filter.Value, paramIndex + 1
}

// buildListFilter builds a SQL condition for ListFilter.
func (m *MySQLOperations) buildListFilter(field string, filter *commonpb.ListFilter, paramIndex int) (string, []any, int) {
	if len(filter.Values) == 0 {
		return "", nil, paramIndex
	}

	placeholders := make([]string, len(filter.Values))
	values := make([]any, len(filter.Values))
	for i, val := range filter.Values {
		placeholders[i] = m.dialect.Placeholder(paramIndex)
		values[i] = val
		paramIndex++
	}

	col := m.dialect.QuoteIdent(field)
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
func (m *MySQLOperations) buildRangeFilter(field string, filter *commonpb.RangeFilter, paramIndex int) ([]string, []any, int) {
	conditions := []string{}
	values := []any{}
	col := m.dialect.QuoteIdent(field)

	minOp := ">"
	if filter.IncludeMin {
		minOp = ">="
	}
	conditions = append(conditions, fmt.Sprintf("%s %s %s", col, minOp, m.dialect.Placeholder(paramIndex)))
	values = append(values, filter.Min)
	paramIndex++

	maxOp := "<"
	if filter.IncludeMax {
		maxOp = "<="
	}
	conditions = append(conditions, fmt.Sprintf("%s %s %s", col, maxOp, m.dialect.Placeholder(paramIndex)))
	values = append(values, filter.Max)
	paramIndex++

	return conditions, values, paramIndex
}

// buildDateFilter builds a SQL condition for DateFilter.
//
// MySQL has no `::date`/`::timestamp` cast syntax; DATE() and CAST(... AS
// DATETIME) are the dialect equivalents of the postgres casts.
func (m *MySQLOperations) buildDateFilter(field string, filter *commonpb.DateFilter, paramIndex int) (string, []any, int) {
	var condition string
	values := []any{}
	col := m.dialect.QuoteIdent(field)

	switch filter.Operator {
	case commonpb.DateOperator_DATE_EQUALS:
		condition = fmt.Sprintf("DATE(%s) = DATE(%s)", col, m.dialect.Placeholder(paramIndex))
		values = append(values, filter.Value)
		paramIndex++
	case commonpb.DateOperator_DATE_BEFORE:
		condition = fmt.Sprintf("%s < CAST(%s AS DATETIME)", col, m.dialect.Placeholder(paramIndex))
		values = append(values, filter.Value)
		paramIndex++
	case commonpb.DateOperator_DATE_AFTER:
		condition = fmt.Sprintf("%s > CAST(%s AS DATETIME)", col, m.dialect.Placeholder(paramIndex))
		values = append(values, filter.Value)
		paramIndex++
	case commonpb.DateOperator_DATE_BETWEEN:
		if filter.RangeEnd != nil && *filter.RangeEnd != "" {
			condition = fmt.Sprintf("%s BETWEEN CAST(%s AS DATETIME) AND CAST(%s AS DATETIME)", col, m.dialect.Placeholder(paramIndex), m.dialect.Placeholder(paramIndex+1))
			values = append(values, filter.Value, *filter.RangeEnd)
			paramIndex += 2
		}
	}

	return condition, values, paramIndex
}

// getTableColumns retrieves column names for a table.
//
// MySQL's information_schema is server-wide, so the lookup is scoped to the
// current schema with table_schema = DATABASE().
func (m *MySQLOperations) getTableColumns(ctx context.Context, tableName string) ([]string, error) {
	query := `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = ?
		ORDER BY ordinal_position
	`

	rows, err := m.getExecutor(ctx).QueryContext(ctx, query, tableName)
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

// getTableColumnTypes returns column-name → information_schema data_type for a
// table. Used by Create/Update to pick the right serialization for
// auto-injected timestamp fields (BIGINT unix-ms vs DATETIME/TIMESTAMP).
func (m *MySQLOperations) getTableColumnTypes(ctx context.Context, tableName string) (map[string]string, error) {
	query := `
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = ?
	`
	rows, err := m.getExecutor(ctx).QueryContext(ctx, query, tableName)
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
// receive unix ms; DATETIME/TIMESTAMP columns receive a time.Time for the
// driver.
func autoTimestampValue(columnType string, now time.Time) any {
	if columnType == "bigint" {
		return now.UnixMilli()
	}
	return now
}

// scanRowToMap scans a single row into a map with snake_case keys (matching DB columns).
func (m *MySQLOperations) scanRowToMap(row *sql.Row, columns []string) (map[string]any, error) {
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
func (m *MySQLOperations) scanRowsToMap(rows *sql.Rows, columns []string) (map[string]any, error) {
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
// time.Time (from DATETIME/TIMESTAMP) → int64 Unix millis, so protojson can
// unmarshal into int64 protobuf fields. JSON columns arrive as []byte and are
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
		// proper JSON instead of base64-encoded strings. The go-sql-driver/mysql
		// driver also returns most text/decimal columns as []byte — if the bytes
		// are not valid JSON we fall back to the string form.
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

// generateUUID generates an application-side UUID. MySQL has no RETURNING, so
// Create assigns the id up front and SELECTs the row back by it.
func generateUUID() string {
	return uuid.NewString()
}

// RunWithTransaction executes a function within a database transaction.
func (m *MySQLOperations) RunWithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to begin transaction: %v", err),
			"MYSQL_TRANSACTION_FAILED",
			500,
		)
	}

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return model.NewDatabaseError(
				fmt.Sprintf("transaction failed and rollback failed: %v, %v", err, rollbackErr),
				"MYSQL_TRANSACTION_ROLLBACK_FAILED",
				500,
			)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to commit transaction: %v", err),
			"MYSQL_TRANSACTION_COMMIT_FAILED",
			500,
		)
	}

	return nil
}

// WithTransaction returns a DatabaseOperation that routes all queries through
// the transaction stored in ctx. Implements interfaces.TransactionAware.
func (m *MySQLOperations) WithTransaction(ctx context.Context) interfaces.DatabaseOperation {
	return m
}

// SupportsTransactions implements interfaces.TransactionAware.
func (m *MySQLOperations) SupportsTransactions() bool {
	return true
}

// GetDB returns the underlying database connection for raw-SQL repositories.
func (m *MySQLOperations) GetDB() *sql.DB {
	return m.db
}

// getExecutor returns *sql.Tx if one is active in ctx, otherwise *sql.DB.
func (m *MySQLOperations) getExecutor(ctx context.Context) dbExecutor {
	tx, ok := operations.GetTransactionFromContext(ctx)
	if ok {
		if myTx, ok := tx.(*MySQLTransaction); ok && myTx.State() == interfaces.TransactionStatePending {
			return myTx.GetTx()
		}
	}
	return m.db
}

// GetExecutor returns *sql.Tx if one is active in ctx, otherwise *sql.DB.
// Entity adapters that build raw SQL (CTEs, JOINs) must call this instead of
// holding their own *sql.DB reference. The return type uses the shared
// interfaces.DBExecutor so adapter packages can type-assert without each
// package defining its own copy.
func (m *MySQLOperations) GetExecutor(ctx context.Context) sqlexec.DBExecutor {
	return m.getExecutor(ctx)
}

// serializeValue converts map and slice values to JSON bytes so the SQL driver
// can store them in JSON columns. Primitive types pass through.
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
// protojson-marshaled data (camelCase) maps to MySQL column names (snake_case).
func normalizeKeys(data map[string]any) map[string]any {
	result := make(map[string]any, len(data))
	for key, value := range data {
		result[camelToSnake(key)] = value
	}
	return result
}

// camelToSnake converts camelCase to snake_case.
func camelToSnake(s string) string {
	var result []rune
	for i, r := range s {
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

// snakeToCamel converts snake_case to camelCase so DB column names map to
// protojson field names for protobuf unmarshalling.
func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// DenormalizeKeys converts all map keys from snake_case to camelCase so that
// MySQL column names map to protojson field names for protobuf unmarshalling.
// Exported for use by entity adapters that convert DB results to protobuf.
func DenormalizeKeys(data map[string]any) map[string]any {
	result := make(map[string]any, len(data))
	for key, value := range data {
		result[snakeToCamel(key)] = value
	}
	return result
}
