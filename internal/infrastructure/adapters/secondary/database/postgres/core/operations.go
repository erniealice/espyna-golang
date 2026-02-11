//go:build postgresql

package core

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/model"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

func init() {
	// Register database operations factory for postgres
	registry.RegisterDatabaseOperationsFactory("postgresql", func(conn any) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres: expected *sql.DB, got %T", conn)
		}
		return NewPostgresOperations(db), nil
	})
}

// PostgresOperations implements DatabaseOperation for PostgreSQL
type PostgresOperations struct {
	db *sql.DB
}

// NewPostgresOperations creates a new PostgreSQL operations instance
func NewPostgresOperations(db *sql.DB) interfaces.DatabaseOperation {
	return &PostgresOperations{
		db: db,
	}
}

// Create creates a new record in the specified table
func (p *PostgresOperations) Create(ctx context.Context, tableName string, data map[string]any) (map[string]any, error) {
	if tableName == "" {
		return nil, model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}

	// Normalize camelCase keys to snake_case (protojson compatibility)
	data = normalizeKeys(data)

	// Get actual table columns so we can discard fields that don't exist in the DB
	// (e.g. protobuf-only fields like date_created_string, date_modified_string)
	resultColumns, err := p.getTableColumns(ctx, tableName)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get table columns: %v", err),
			"POSTGRES_SCHEMA_ERROR",
			500,
		)
	}
	validColumns := make(map[string]bool, len(resultColumns))
	for _, col := range resultColumns {
		validColumns[col] = true
	}

	// Set creation properties
	now := time.Now().UTC()
	if _, exists := data["id"]; !exists {
		data["id"] = generateUUID()
	}
	data["active"] = true
	data["date_created"] = now
	data["date_modified"] = now

	// Build INSERT query (only columns that exist in the table)
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]any, 0, len(data))

	i := 1
	for column, value := range data {
		if !validColumns[column] {
			continue
		}
		columns = append(columns, column)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, value)
		i++
	}

	query := fmt.Sprintf(
		"INSERT INTO \"%s\" (%s) VALUES (%s) RETURNING *",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	// Execute query
	row := p.db.QueryRowContext(ctx, query, values...)

	// Scan result
	result, err := p.scanRowToMap(row, resultColumns)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to create record: %v", err),
			"POSTGRES_CREATE_FAILED",
			500,
		)
	}

	return result, nil
}

// Read retrieves a record by ID from the specified table
func (p *PostgresOperations) Read(ctx context.Context, tableName string, id string) (map[string]any, error) {
	if tableName == "" {
		return nil, model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}
	if id == "" {
		return nil, model.NewDatabaseError("record ID is required", "MISSING_RECORD_ID", 400)
	}

	query := fmt.Sprintf("SELECT * FROM \"%s\" WHERE id = $1 AND active = true", tableName)

	row := p.db.QueryRowContext(ctx, query, id)

	// Get column names
	resultColumns, err := p.getTableColumns(ctx, tableName)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get table columns: %v", err),
			"POSTGRES_SCHEMA_ERROR",
			500,
		)
	}

	// Scan result
	result, err := p.scanRowToMap(row, resultColumns)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
		}
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to read record: %v", err),
			"POSTGRES_READ_FAILED",
			500,
		)
	}

	return result, nil
}

// Update updates an existing record in the specified table
func (p *PostgresOperations) Update(ctx context.Context, tableName string, id string, data map[string]any) (map[string]any, error) {
	if tableName == "" {
		return nil, model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}
	if id == "" {
		return nil, model.NewDatabaseError("record ID is required", "MISSING_RECORD_ID", 400)
	}

	// Normalize camelCase keys to snake_case (protojson compatibility)
	data = normalizeKeys(data)

	// Get actual table columns to discard protobuf-only fields
	resultColumns, err := p.getTableColumns(ctx, tableName)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get table columns: %v", err),
			"POSTGRES_SCHEMA_ERROR",
			500,
		)
	}
	validColumns := make(map[string]bool, len(resultColumns))
	for _, col := range resultColumns {
		validColumns[col] = true
	}

	// Check if record exists (query without active filter so we can update
	// inactive records too, e.g. re-activating a soft-deleted record).
	existQuery := fmt.Sprintf("SELECT * FROM \"%s\" WHERE id = $1", tableName)
	existRow := p.db.QueryRowContext(ctx, existQuery, id)
	existing, err := p.scanRowToMap(existRow, resultColumns)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
		}
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to read record for update: %v", err),
			"POSTGRES_READ_FAILED",
			500,
		)
	}

	// Set update properties
	now := time.Now().UTC()
	data["date_modified"] = now

	// Preserve original creation data, converting millis back to time.Time
	// (normalizeValue converts time.Time → int64 millis during Read, but
	// Postgres needs time.Time for timestamp columns on write)
	if dc := existing["date_created"]; dc != nil {
		if millis, ok := dc.(int64); ok {
			data["date_created"] = time.UnixMilli(millis).UTC()
		} else {
			data["date_created"] = dc
		}
	}

	// Build UPDATE query (only columns that exist in the table)
	setParts := make([]string, 0, len(data))
	values := make([]any, 0, len(data)+1)

	i := 1
	for column, value := range data {
		if column == "id" || !validColumns[column] {
			continue
		}
		setParts = append(setParts, fmt.Sprintf("%s = $%d", column, i))
		values = append(values, value)
		i++
	}
	values = append(values, id) // Add ID as last parameter

	// No active filter — allows re-activating soft-deleted records.
	query := fmt.Sprintf(
		"UPDATE \"%s\" SET %s WHERE id = $%d RETURNING *",
		tableName,
		strings.Join(setParts, ", "),
		i,
	)

	// Execute query
	row := p.db.QueryRowContext(ctx, query, values...)

	// Scan result
	result, err := p.scanRowToMap(row, resultColumns)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to update record: %v", err),
			"POSTGRES_UPDATE_FAILED",
			500,
		)
	}

	return result, nil
}

// Delete deletes a record from the specified table (soft delete by default)
func (p *PostgresOperations) Delete(ctx context.Context, tableName string, id string) error {
	if tableName == "" {
		return model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}
	if id == "" {
		return model.NewDatabaseError("record ID is required", "MISSING_RECORD_ID", 400)
	}

	// Soft delete by setting active to false
	now := time.Now().UTC()
	query := fmt.Sprintf(
		"UPDATE \"%s\" SET active = false, date_modified = $1 WHERE id = $2 AND active = true",
		tableName,
	)

	result, err := p.db.ExecContext(ctx, query, now, id)
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to delete record: %v", err),
			"POSTGRES_DELETE_FAILED",
			500,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to get affected rows: %v", err),
			"POSTGRES_DELETE_FAILED",
			500,
		)
	}

	if rowsAffected == 0 {
		return model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
	}

	return nil
}

// HardDelete permanently deletes a record from the specified table
func (p *PostgresOperations) HardDelete(ctx context.Context, tableName string, id string) error {
	if tableName == "" {
		return model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}
	if id == "" {
		return model.NewDatabaseError("record ID is required", "MISSING_RECORD_ID", 400)
	}

	query := fmt.Sprintf("DELETE FROM \"%s\" WHERE id = $1", tableName)

	result, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to hard delete record: %v", err),
			"POSTGRES_HARD_DELETE_FAILED",
			500,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to get affected rows: %v", err),
			"POSTGRES_HARD_DELETE_FAILED",
			500,
		)
	}

	if rowsAffected == 0 {
		return model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
	}

	return nil
}

// List retrieves records from the specified table with standardized params
func (p *PostgresOperations) List(ctx context.Context, tableName string, params *interfaces.ListParams) (*interfaces.ListResult, error) {
	if tableName == "" {
		return nil, model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}

	// Build WHERE clause
	whereConditions := []string{"active = true"}
	values := []any{}
	paramIndex := 1

	// Apply filters from FilterRequest
	if params != nil && params.Filters != nil {
		filterConditions, filterValues, nextIndex := p.buildFilterConditions(params.Filters, paramIndex)
		whereConditions = append(whereConditions, filterConditions...)
		values = append(values, filterValues...)
		paramIndex = nextIndex
	}

	// Build ORDER BY clause
	orderByClause := "ORDER BY date_created DESC" // Default ordering
	if params != nil && params.Sort != nil && len(params.Sort.Fields) > 0 {
		orderByParts := make([]string, 0, len(params.Sort.Fields))
		for _, sortField := range params.Sort.Fields {
			direction := "ASC"
			if sortField.Direction == commonpb.SortDirection_DESC {
				direction = "DESC"
			}

			// Handle NULL ordering
			nullOrder := ""
			if sortField.NullOrder == commonpb.NullOrder_NULLS_FIRST {
				nullOrder = " NULLS FIRST"
			} else if sortField.NullOrder == commonpb.NullOrder_NULLS_LAST {
				nullOrder = " NULLS LAST"
			}

			orderByParts = append(orderByParts, fmt.Sprintf("%s %s%s", sortField.Field, direction, nullOrder))
		}
		orderByClause = "ORDER BY " + strings.Join(orderByParts, ", ")
	}

	// Get total count before pagination
	countQuery := fmt.Sprintf(
		"SELECT COUNT(*) FROM \"%s\" WHERE %s",
		tableName,
		strings.Join(whereConditions, " AND "),
	)

	var totalItems int32
	err := p.db.QueryRowContext(ctx, countQuery, values...).Scan(&totalItems)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to count records: %v", err),
			"POSTGRES_COUNT_FAILED",
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
		// Handle offset pagination
		if offsetPagination := params.Pagination.GetOffset(); offsetPagination != nil {
			if offsetPagination.Page > 0 {
				offset = (offsetPagination.Page - 1) * limit
			}
		}
	}

	// Build final query with pagination
	query := fmt.Sprintf(
		"SELECT * FROM \"%s\" WHERE %s %s LIMIT $%d OFFSET $%d",
		tableName,
		strings.Join(whereConditions, " AND "),
		orderByClause,
		paramIndex,
		paramIndex+1,
	)
	values = append(values, limit, offset)

	// Execute query
	rows, err := p.db.QueryContext(ctx, query, values...)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to list records: %v", err),
			"POSTGRES_LIST_FAILED",
			500,
		)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get columns: %v", err),
			"POSTGRES_LIST_FAILED",
			500,
		)
	}

	// Scan results
	var results []map[string]any
	for rows.Next() {
		result, err := p.scanRowsToMap(rows, columns)
		if err != nil {
			return nil, model.NewDatabaseError(
				fmt.Sprintf("failed to scan row: %v", err),
				"POSTGRES_LIST_FAILED",
				500,
			)
		}
		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("rows iteration error: %v", err),
			"POSTGRES_LIST_FAILED",
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

// Query executes a structured query against the PostgreSQL table
func (p *PostgresOperations) Query(ctx context.Context, tableName string, queryBuilder interfaces.QueryBuilder) ([]map[string]any, error) {
	if tableName == "" {
		return nil, model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}

	if queryBuilder == nil {
		return nil, model.NewDatabaseError("query builder is required", "MISSING_QUERY_BUILDER", 400)
	}

	// Build the query filter
	filter, err := queryBuilder.Build()
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to build query: %v", err),
			"QUERY_BUILD_FAILED",
			400,
		)
	}

	// Build WHERE clause
	whereConditions := []string{}
	values := []any{}
	paramIndex := 1

	// Apply query conditions
	for _, condition := range filter.Conditions {
		switch condition.Operator {
		case "==":
			whereConditions = append(whereConditions, fmt.Sprintf("%s = $%d", condition.Field, paramIndex))
			values = append(values, condition.Value)
			paramIndex++
		case "!=":
			whereConditions = append(whereConditions, fmt.Sprintf("%s != $%d", condition.Field, paramIndex))
			values = append(values, condition.Value)
			paramIndex++
		case "in":
			if valueSlice, ok := condition.Value.([]any); ok && len(valueSlice) > 0 {
				placeholders := make([]string, len(valueSlice))
				for i, val := range valueSlice {
					placeholders[i] = fmt.Sprintf("$%d", paramIndex)
					values = append(values, val)
					paramIndex++
				}
				whereConditions = append(whereConditions, fmt.Sprintf("%s IN (%s)", condition.Field, strings.Join(placeholders, ", ")))
			}
		case ">":
			whereConditions = append(whereConditions, fmt.Sprintf("%s > $%d", condition.Field, paramIndex))
			values = append(values, condition.Value)
			paramIndex++
		case "<":
			whereConditions = append(whereConditions, fmt.Sprintf("%s < $%d", condition.Field, paramIndex))
			values = append(values, condition.Value)
			paramIndex++
		case ">=":
			whereConditions = append(whereConditions, fmt.Sprintf("%s >= $%d", condition.Field, paramIndex))
			values = append(values, condition.Value)
			paramIndex++
		case "<=":
			whereConditions = append(whereConditions, fmt.Sprintf("%s <= $%d", condition.Field, paramIndex))
			values = append(values, condition.Value)
			paramIndex++
		case "LIKE":
			whereConditions = append(whereConditions, fmt.Sprintf("%s LIKE $%d", condition.Field, paramIndex))
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

	// Build the base query
	query := fmt.Sprintf("SELECT * FROM \"%s\"", tableName)

	// Add WHERE clause if we have conditions
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
			orderParts[i] = fmt.Sprintf("%s %s", orderBy.Field, direction)
		}
		query += " ORDER BY " + strings.Join(orderParts, ", ")
	} else {
		// Default ordering by date_created if no explicit ordering
		query += " ORDER BY date_created DESC"
	}

	// Add LIMIT clause
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	// Execute query
	rows, err := p.db.QueryContext(ctx, query, values...)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to execute query: %v", err),
			"POSTGRES_QUERY_FAILED",
			500,
		)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get columns: %v", err),
			"POSTGRES_QUERY_FAILED",
			500,
		)
	}

	// Scan results
	var results []map[string]any
	for rows.Next() {
		result, err := p.scanRowsToMap(rows, columns)
		if err != nil {
			return nil, model.NewDatabaseError(
				fmt.Sprintf("failed to scan row: %v", err),
				"POSTGRES_QUERY_FAILED",
				500,
			)
		}
		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("rows iteration error: %v", err),
			"POSTGRES_QUERY_FAILED",
			500,
		)
	}

	return results, nil
}

// QueryOne executes a structured query and returns the first result
func (p *PostgresOperations) QueryOne(ctx context.Context, tableName string, queryBuilder interfaces.QueryBuilder) (map[string]any, error) {
	// Use Query with limit 1
	limitedBuilder := queryBuilder.Limit(1)
	results, err := p.Query(ctx, tableName, limitedBuilder)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, model.NewDatabaseError("no results found", "NO_RESULTS_FOUND", 404)
	}

	return results[0], nil
}

// Helper methods

// buildFilterConditions builds WHERE conditions from FilterRequest
func (p *PostgresOperations) buildFilterConditions(filterReq *commonpb.FilterRequest, startIndex int) ([]string, []any, int) {
	conditions := []string{}
	values := []any{}
	paramIndex := startIndex

	for _, filter := range filterReq.Filters {
		field := filter.Field

		switch ft := filter.FilterType.(type) {
		case *commonpb.TypedFilter_StringFilter:
			condition, vals, nextIndex := p.buildStringFilter(field, ft.StringFilter, paramIndex)
			conditions = append(conditions, condition)
			values = append(values, vals...)
			paramIndex = nextIndex

		case *commonpb.TypedFilter_NumberFilter:
			condition, val, nextIndex := p.buildNumberFilter(field, ft.NumberFilter, paramIndex)
			conditions = append(conditions, condition)
			values = append(values, val)
			paramIndex = nextIndex

		case *commonpb.TypedFilter_BooleanFilter:
			conditions = append(conditions, fmt.Sprintf("%s = $%d", field, paramIndex))
			values = append(values, ft.BooleanFilter.Value)
			paramIndex++

		case *commonpb.TypedFilter_ListFilter:
			condition, vals, nextIndex := p.buildListFilter(field, ft.ListFilter, paramIndex)
			if condition != "" {
				conditions = append(conditions, condition)
				values = append(values, vals...)
				paramIndex = nextIndex
			}

		case *commonpb.TypedFilter_RangeFilter:
			rangeConditions, vals, nextIndex := p.buildRangeFilter(field, ft.RangeFilter, paramIndex)
			conditions = append(conditions, rangeConditions...)
			values = append(values, vals...)
			paramIndex = nextIndex

		case *commonpb.TypedFilter_DateFilter:
			condition, vals, nextIndex := p.buildDateFilter(field, ft.DateFilter, paramIndex)
			if condition != "" {
				conditions = append(conditions, condition)
				values = append(values, vals...)
				paramIndex = nextIndex
			}
		}
	}

	return conditions, values, paramIndex
}

// buildStringFilter builds SQL condition for StringFilter
func (p *PostgresOperations) buildStringFilter(field string, filter *commonpb.StringFilter, paramIndex int) (string, []any, int) {
	value := filter.Value
	if !filter.CaseSensitive {
		field = fmt.Sprintf("LOWER(%s)", field)
		value = strings.ToLower(value)
	}

	var condition string
	var values []any

	switch filter.Operator {
	case commonpb.StringOperator_STRING_EQUALS:
		condition = fmt.Sprintf("%s = $%d", field, paramIndex)
		values = append(values, value)
		paramIndex++
	case commonpb.StringOperator_STRING_NOT_EQUALS:
		condition = fmt.Sprintf("%s != $%d", field, paramIndex)
		values = append(values, value)
		paramIndex++
	case commonpb.StringOperator_STRING_CONTAINS:
		condition = fmt.Sprintf("%s LIKE $%d", field, paramIndex)
		values = append(values, "%"+value+"%")
		paramIndex++
	case commonpb.StringOperator_STRING_STARTS_WITH:
		condition = fmt.Sprintf("%s LIKE $%d", field, paramIndex)
		values = append(values, value+"%")
		paramIndex++
	case commonpb.StringOperator_STRING_ENDS_WITH:
		condition = fmt.Sprintf("%s LIKE $%d", field, paramIndex)
		values = append(values, "%"+value)
		paramIndex++
	case commonpb.StringOperator_STRING_REGEX:
		condition = fmt.Sprintf("%s ~ $%d", field, paramIndex)
		values = append(values, value)
		paramIndex++
	}

	return condition, values, paramIndex
}

// buildNumberFilter builds SQL condition for NumberFilter
func (p *PostgresOperations) buildNumberFilter(field string, filter *commonpb.NumberFilter, paramIndex int) (string, any, int) {
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

	condition := fmt.Sprintf("%s %s $%d", field, operator, paramIndex)
	return condition, filter.Value, paramIndex + 1
}

// buildListFilter builds SQL condition for ListFilter
func (p *PostgresOperations) buildListFilter(field string, filter *commonpb.ListFilter, paramIndex int) (string, []any, int) {
	if len(filter.Values) == 0 {
		return "", nil, paramIndex
	}

	placeholders := make([]string, len(filter.Values))
	values := make([]any, len(filter.Values))
	for i, val := range filter.Values {
		placeholders[i] = fmt.Sprintf("$%d", paramIndex)
		values[i] = val
		paramIndex++
	}

	var condition string
	switch filter.Operator {
	case commonpb.ListOperator_LIST_IN:
		condition = fmt.Sprintf("%s IN (%s)", field, strings.Join(placeholders, ", "))
	case commonpb.ListOperator_LIST_NOT_IN:
		condition = fmt.Sprintf("%s NOT IN (%s)", field, strings.Join(placeholders, ", "))
	}

	return condition, values, paramIndex
}

// buildRangeFilter builds SQL conditions for RangeFilter
func (p *PostgresOperations) buildRangeFilter(field string, filter *commonpb.RangeFilter, paramIndex int) ([]string, []any, int) {
	conditions := []string{}
	values := []any{}

	// Min condition
	minOp := ">"
	if filter.IncludeMin {
		minOp = ">="
	}
	conditions = append(conditions, fmt.Sprintf("%s %s $%d", field, minOp, paramIndex))
	values = append(values, filter.Min)
	paramIndex++

	// Max condition
	maxOp := "<"
	if filter.IncludeMax {
		maxOp = "<="
	}
	conditions = append(conditions, fmt.Sprintf("%s %s $%d", field, maxOp, paramIndex))
	values = append(values, filter.Max)
	paramIndex++

	return conditions, values, paramIndex
}

// buildDateFilter builds SQL condition for DateFilter
func (p *PostgresOperations) buildDateFilter(field string, filter *commonpb.DateFilter, paramIndex int) (string, []any, int) {
	var condition string
	values := []any{}

	switch filter.Operator {
	case commonpb.DateOperator_DATE_EQUALS:
		condition = fmt.Sprintf("%s::date = $%d::date", field, paramIndex)
		values = append(values, filter.Value)
		paramIndex++
	case commonpb.DateOperator_DATE_BEFORE:
		condition = fmt.Sprintf("%s < $%d::timestamp", field, paramIndex)
		values = append(values, filter.Value)
		paramIndex++
	case commonpb.DateOperator_DATE_AFTER:
		condition = fmt.Sprintf("%s > $%d::timestamp", field, paramIndex)
		values = append(values, filter.Value)
		paramIndex++
	case commonpb.DateOperator_DATE_BETWEEN:
		if filter.RangeEnd != nil && *filter.RangeEnd != "" {
			condition = fmt.Sprintf("%s BETWEEN $%d::timestamp AND $%d::timestamp", field, paramIndex, paramIndex+1)
			values = append(values, filter.Value, *filter.RangeEnd)
			paramIndex += 2
		}
	}

	return condition, values, paramIndex
}

// getTableColumns retrieves column names for a table
func (p *PostgresOperations) getTableColumns(ctx context.Context, tableName string) ([]string, error) {
	query := `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_name = $1
		ORDER BY ordinal_position
	`

	rows, err := p.db.QueryContext(ctx, query, tableName)
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

// scanRowToMap scans a single row into a map
func (p *PostgresOperations) scanRowToMap(row *sql.Row, columns []string) (map[string]any, error) {
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

// scanRowsToMap scans a single row from *sql.Rows into a map
func (p *PostgresOperations) scanRowsToMap(rows *sql.Rows, columns []string) (map[string]any, error) {
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

// normalizeValue converts DB-native types to protobuf-compatible types.
// Specifically, time.Time (from TIMESTAMPTZ) → int64 Unix millis,
// so protojson can unmarshal into int64 protobuf fields.
func normalizeValue(v any) any {
	switch t := v.(type) {
	case time.Time:
		if t.IsZero() {
			return nil
		}
		return t.UnixMilli()
	default:
		return v
	}
}

// generateUUID generates a simple UUID (simplified implementation)
func generateUUID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// WithTransaction executes a function within a database transaction
func (p *PostgresOperations) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to begin transaction: %v", err),
			"POSTGRES_TRANSACTION_FAILED",
			500,
		)
	}

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return model.NewDatabaseError(
				fmt.Sprintf("transaction failed and rollback failed: %v, %v", err, rollbackErr),
				"POSTGRES_TRANSACTION_ROLLBACK_FAILED",
				500,
			)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to commit transaction: %v", err),
			"POSTGRES_TRANSACTION_COMMIT_FAILED",
			500,
		)
	}

	return nil
}

// GetDB returns the underlying database connection
// This is used for executing raw SQL queries in repository implementations
func (p *PostgresOperations) GetDB() *sql.DB {
	return p.db
}

// normalizeKeys converts all map keys from camelCase to snake_case.
// This ensures protojson-marshaled data (camelCase) maps correctly to
// PostgreSQL column names (snake_case).
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
