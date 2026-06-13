//go:build postgresql

package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/database/model"
	sqlexec "github.com/erniealice/espyna-golang/database/sqlexec"
	"github.com/erniealice/espyna-golang/database/operations"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/registry"
	"github.com/erniealice/espyna-golang/schema"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	_ "github.com/lib/pq"
)

// dbExecutor abstracts *sql.DB and *sql.Tx for uniform query execution.
type dbExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func init() {
	// Register database operations factory for postgres.
	//
	// The returned DatabaseOperation is WorkspaceAware — it injects
	// workspace_id into Create/List/Read/Update/Delete whenever:
	//   (a) the request context carries a workspace_id, AND
	//   (b) the target table has a workspace_id column.
	// For global/non-tenanted tables (workspace, role, permission, etc.) or
	// service-to-service calls without a workspace context, the decorator is
	// a pass-through. This makes CRUDSource consumers (e.g. entydad's
	// location_area list, which went through ListSimple) automatically
	// workspace-scoped, matching the behavior of entity-specific adapters
	// that wrap themselves explicitly.
	registry.RegisterDatabaseOperationsFactory("postgresql", func(conn any) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres: expected *sql.DB, got %T", conn)
		}
		return NewWorkspaceAwareOperations(db), nil
	})
}

// PostgresOperations implements DatabaseOperation for PostgreSQL
type PostgresOperations struct {
	db           *sql.DB
	auditService infraports.AuditService // optional — nil = audit disabled
}

// NewPostgresOperations creates a new PostgreSQL operations instance
func NewPostgresOperations(db *sql.DB) interfaces.DatabaseOperation {
	return &PostgresOperations{
		db: db,
	}
}

// NewPostgresOperationsWithAudit creates a PostgreSQL operations instance with audit logging enabled.
// When auditSvc is non-nil, Create/Update/Delete will call DiffAndLog after each successful mutation.
func NewPostgresOperationsWithAudit(db *sql.DB, auditSvc infraports.AuditService) *PostgresOperations {
	return &PostgresOperations{db: db, auditService: auditSvc}
}

// Create creates a new record in the specified table
func (p *PostgresOperations) Create(ctx context.Context, tableName string, data map[string]any) (map[string]any, error) {
	if tableName == "" {
		return nil, model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}

	// Normalize camelCase keys to snake_case (protojson compatibility).
	// NOTE: the descriptor-registry shadow lookups below (schema.ColsFor /
	// schema.ColByName) depend on this running FIRST — descriptor column names are
	// the proto snake_case TextNames, so they only match the map keys / reflected
	// column names after camelCase keys are snake_cased (phase0 §b key-match invariant).
	data = normalizeKeys(data)

	// Get actual table columns so we can discard fields that don't exist in the DB
	// (e.g. protobuf-only fields like date_created_string, date_modified_string).
	// This reflected set remains the authoritative source-of-truth in SHADOW mode
	// (Phase 2); its deletion is deferred to Phase 5.
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

	columnTypes, err := p.getTableColumnTypes(ctx, tableName)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get table column types: %v", err),
			"POSTGRES_SCHEMA_ERROR",
			500,
		)
	}

	// SHADOW: assert the descriptor-derived column set agrees with the reflected set.
	// On disagreement this WARNs and falls back to the reflected validColumns below.
	shadowAssertColumnSet(tableName, validColumns)

	// Set creation properties. The timestamp column type feeding autoTimestampValue
	// is sourced from the descriptor (bigint-millis vs Timestamp) and cross-checked
	// against the reflected information_schema data_type — on mismatch a WARN is
	// logged and the reflected type wins (SHADOW: reflection authoritative).
	now := time.Now().UTC()
	if _, exists := data["id"]; !exists {
		data["id"] = generateUUID()
	}
	data["active"] = true
	data["date_created"] = autoTimestampValue(shadowTimestampType(tableName, "date_created", columnTypes), now)
	data["date_modified"] = autoTimestampValue(shadowTimestampType(tableName, "date_modified", columnTypes), now)

	// SHADOW: compare the descriptor-derived autoTimestamp VALUE against the
	// reflection-derived value for the SAME `now` (not just the type axis). The
	// reflection-derived value (already stamped above) drives the write; this only
	// records the agreement metric + WARNs on a value divergence.
	shadowAssertAutoTimestamp(tableName, "date_created", columnTypes, now)
	shadowAssertAutoTimestamp(tableName, "date_modified", columnTypes, now)

	// Build INSERT query (only columns that exist in the table)
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
		columns = append(columns, column)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, serializeValue(value))
		i++
	}
	if len(skipped) > 0 {
		log.Printf("PostgresOperations.Create: dropped %d unknown column(s) for table=%q skipped=%v", len(skipped), tableName, skipped)
	}
	// SHADOW: surface where a descriptor-authoritative drop would differ from the
	// reflected drop (the reflected `skipped` set still drives the write).
	shadowAssertDropSet(tableName, data, skipped, false)

	query := fmt.Sprintf(
		"INSERT INTO \"%s\" (%s) VALUES (%s) RETURNING *",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	// Execute query
	row := p.getExecutor(ctx).QueryRowContext(ctx, query, values...)

	// Scan result
	result, err := p.scanRowToMap(row, resultColumns)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to create record: %v", err),
			"POSTGRES_CREATE_FAILED",
			500,
		)
	}

	if p.auditService != nil {
		if err := infraports.DiffAndLog(ctx, p.auditService, infraports.DiffAndLogRequest{
			EntityType: tableName,
			EntityID:   fmt.Sprintf("%v", result["id"]),
			Domain:     tableName,
			Action:     1, // INSERT
			MethodName: "PostgresOperations.Create",
			NewData:    result,
		}); err != nil {
			return nil, err
		}
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

	query := fmt.Sprintf("SELECT * FROM \"%s\" WHERE id = $1", tableName)

	row := p.getExecutor(ctx).QueryRowContext(ctx, query, id)

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

	// Normalize camelCase keys to snake_case (protojson compatibility).
	// The descriptor-registry shadow lookups below depend on this running FIRST
	// (phase0 §b key-match invariant — see Create).
	data = normalizeKeys(data)

	// Get actual table columns to discard protobuf-only fields. Reflected set
	// remains authoritative in SHADOW mode; deletion deferred to Phase 5.
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

	columnTypes, err := p.getTableColumnTypes(ctx, tableName)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get table column types: %v", err),
			"POSTGRES_SCHEMA_ERROR",
			500,
		)
	}

	// SHADOW: assert descriptor-derived column set agrees with the reflected set.
	shadowAssertColumnSet(tableName, validColumns)

	// Check if record exists (query without active filter so we can update
	// inactive records too, e.g. re-activating a soft-deleted record).
	existQuery := fmt.Sprintf("SELECT * FROM \"%s\" WHERE id = $1", tableName)
	existRow := p.getExecutor(ctx).QueryRowContext(ctx, existQuery, id)
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

	// Set update properties (column-type-aware: BIGINT timestamp columns
	// receive unix ms, TIMESTAMP columns receive time.Time). The timestamp type
	// is sourced from the descriptor (bigint-millis vs Timestamp), cross-checked
	// against the reflected information_schema data_type — on mismatch WARN + the
	// reflected type wins (SHADOW: reflection authoritative).
	now := time.Now().UTC()
	dateModifiedType := shadowTimestampType(tableName, "date_modified", columnTypes)
	dateCreatedType := shadowTimestampType(tableName, "date_created", columnTypes)
	data["date_modified"] = autoTimestampValue(dateModifiedType, now)

	// SHADOW: value-axis agreement check for the only auto-stamped column on Update
	// (date_modified). date_created is preserved from the existing row, not stamped,
	// so it is not compared here. Observe-only; reflection still drives the write.
	shadowAssertAutoTimestamp(tableName, "date_modified", columnTypes, now)

	// Preserve original creation data.
	// scanRowToMap normalises TIMESTAMP columns to int64 unix ms for the
	// caller, so for TIMESTAMP columns we must convert back to time.Time
	// before passing to pq. For BIGINT columns the stored int64 is already
	// the wire format pq expects.
	if dc := existing["date_created"]; dc != nil {
		if dateCreatedType == "bigint" {
			data["date_created"] = dc
		} else if millis, ok := dc.(int64); ok {
			data["date_created"] = time.UnixMilli(millis).UTC()
		} else {
			data["date_created"] = dc
		}
	}

	// Build UPDATE query (only columns that exist in the table)
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
		setParts = append(setParts, fmt.Sprintf("%s = $%d", column, i))
		values = append(values, serializeValue(value))
		i++
	}
	if len(skipped) > 0 {
		log.Printf("PostgresOperations.Update: dropped %d unknown column(s) for table=%q id=%q skipped=%v", len(skipped), tableName, id, skipped)
	}
	// SHADOW: surface where a descriptor-authoritative drop would differ from the
	// reflected drop (the reflected `skipped` set still drives the write). The
	// update loop skips the id key, so the descriptor drop computation does too.
	shadowAssertDropSet(tableName, data, skipped, true)
	values = append(values, id) // Add ID as last parameter

	// No active filter — allows re-activating soft-deleted records.
	query := fmt.Sprintf(
		"UPDATE \"%s\" SET %s WHERE id = $%d RETURNING *",
		tableName,
		strings.Join(setParts, ", "),
		i,
	)

	// Execute query
	row := p.getExecutor(ctx).QueryRowContext(ctx, query, values...)

	// Scan result
	result, err := p.scanRowToMap(row, resultColumns)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to update record: %v", err),
			"POSTGRES_UPDATE_FAILED",
			500,
		)
	}

	if p.auditService != nil {
		if err := infraports.DiffAndLog(ctx, p.auditService, infraports.DiffAndLogRequest{
			EntityType: tableName,
			EntityID:   id,
			Domain:     tableName,
			Action:     2, // UPDATE
			MethodName: "PostgresOperations.Update",
			OldData:    existing,
			NewData:    result,
		}); err != nil {
			return nil, err
		}
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

	// Soft delete by setting active to false. date_modified may be BIGINT
	// unix ms or TIMESTAMP depending on the entity schema. The type is sourced
	// from the descriptor (schema.ColByName via shadowTimestampType) and
	// cross-checked against the reflected information_schema data_type — on
	// disagreement a WARN is logged and the reflected type wins (SHADOW:
	// reflection authoritative; getTableColumnTypes stays as the cross-check).
	columnTypes, err := p.getTableColumnTypes(ctx, tableName)
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to get table column types: %v", err),
			"POSTGRES_SCHEMA_ERROR",
			500,
		)
	}
	now := time.Now().UTC()
	dateModifiedType := shadowTimestampType(tableName, "date_modified", columnTypes)
	// SHADOW: value-axis agreement check for the date_modified stamp written by the
	// soft-delete UPDATE below. Observe-only; reflection still drives the write.
	shadowAssertAutoTimestamp(tableName, "date_modified", columnTypes, now)
	// Soft-delete is idempotent: deleting an already-inactive row is not an
	// error (prior behavior required active = true in WHERE, which caused
	// RECORD_NOT_FOUND when users ran Delete from the inactive list).
	// Soft-delete stamps active=false + date_modified in Go (Q-RC6 reverted, unchanged).
	query := fmt.Sprintf(
		"UPDATE \"%s\" SET active = false, date_modified = $1 WHERE id = $2",
		tableName,
	)

	result, err := p.getExecutor(ctx).ExecContext(ctx, query, autoTimestampValue(dateModifiedType, now), id)
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

	if p.auditService != nil {
		if err := infraports.DiffAndLog(ctx, p.auditService, infraports.DiffAndLogRequest{
			EntityType: tableName,
			EntityID:   id,
			Domain:     tableName,
			Action:     3, // DELETE
			MethodName: "PostgresOperations.Delete",
		}); err != nil {
			return err
		}
	}

	return nil
}

// HardDelete permanently deletes a record from the specified table.
//
// TODO(recycle-bin): long-term, catalog entities (product, plan, price_plan,
// price_schedule, price_list, etc.) that use HardDelete today should migrate
// to a two-stage delete: move the row to a shared `recycle_bin` table with
// `entity_type`, `entity_id`, `payload JSONB`, and `deleted_at` columns, then
// run a scheduled purge (e.g. 30-day retention). This gives users an undelete
// affordance without reintroducing the active=false graveyard pattern the
// previous soft-delete implementation suffered from. The current hard-delete
// behavior relies on FK RESTRICT as the safety net; the recycle-bin layer
// should preserve that guarantee by checking references before bin insert.
func (p *PostgresOperations) HardDelete(ctx context.Context, tableName string, id string) error {
	if tableName == "" {
		return model.NewDatabaseError("table name is required", "MISSING_TABLE_NAME", 400)
	}
	if id == "" {
		return model.NewDatabaseError("record ID is required", "MISSING_RECORD_ID", 400)
	}

	query := fmt.Sprintf("DELETE FROM \"%s\" WHERE id = $1", tableName)

	result, err := p.getExecutor(ctx).ExecContext(ctx, query, id)
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

	// Build WHERE clause.
	// Default to active = true unless the caller supplies an explicit "active"
	// BooleanFilter — in that case we honour the caller's value so that inactive
	// records can be retrieved (e.g. inactive product/service list page).
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
		whereConditions = []string{"active = true"}
	}
	values := []any{}
	paramIndex := 1

	// Apply filters from FilterRequest
	if params != nil && params.Filters != nil {
		filterConditions, filterValues, nextIndex := p.buildFilterConditions(params.Filters, paramIndex)
		whereConditions = append(whereConditions, filterConditions...)
		values = append(values, filterValues...)
		paramIndex = nextIndex
	}

	// Search — ILIKE OR block across declared search fields
	if params != nil && params.Search != nil && params.Search.Query != "" {
		query := "%" + params.Search.Query + "%"
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
			values = append(values, query)
			likeClauses = append(likeClauses, fmt.Sprintf("%s ILIKE $%d", col, paramIndex))
			paramIndex++
		}
		whereConditions = append(whereConditions, "("+strings.Join(likeClauses, " OR ")+")")
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
	err := p.getExecutor(ctx).QueryRowContext(ctx, countQuery, values...).Scan(&totalItems)
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
	rows, err := p.getExecutor(ctx).QueryContext(ctx, query, values...)
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
	rows, err := p.getExecutor(ctx).QueryContext(ctx, query, values...)
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

		case *commonpb.TypedFilter_MoneyFilter:
			mf := ft.MoneyFilter
			col := filter.Field
			switch mf.Operator {
			case commonpb.MoneyOperator_MONEY_EQUALS:
				conditions = append(conditions, fmt.Sprintf("%s = $%d", col, paramIndex))
				values = append(values, mf.Amount)
				paramIndex++
			case commonpb.MoneyOperator_MONEY_LESS_THAN:
				conditions = append(conditions, fmt.Sprintf("%s < $%d", col, paramIndex))
				values = append(values, mf.Amount)
				paramIndex++
			case commonpb.MoneyOperator_MONEY_GREATER_THAN:
				conditions = append(conditions, fmt.Sprintf("%s > $%d", col, paramIndex))
				values = append(values, mf.Amount)
				paramIndex++
			case commonpb.MoneyOperator_MONEY_LESS_THAN_OR_EQUAL:
				conditions = append(conditions, fmt.Sprintf("%s <= $%d", col, paramIndex))
				values = append(values, mf.Amount)
				paramIndex++
			case commonpb.MoneyOperator_MONEY_GREATER_THAN_OR_EQUAL:
				conditions = append(conditions, fmt.Sprintf("%s >= $%d", col, paramIndex))
				values = append(values, mf.Amount)
				paramIndex++
			case commonpb.MoneyOperator_MONEY_BETWEEN:
				conditions = append(conditions, fmt.Sprintf("%s BETWEEN $%d AND $%d", col, paramIndex, paramIndex+1))
				values = append(values, mf.Amount, mf.AmountTo)
				paramIndex += 2
			}

		case *commonpb.TypedFilter_StatusFilter:
			sf := ft.StatusFilter
			if len(sf.Values) > 0 {
				placeholders := make([]string, len(sf.Values))
				for i, v := range sf.Values {
					placeholders[i] = fmt.Sprintf("$%d", paramIndex)
					values = append(values, v)
					paramIndex++
				}
				conditions = append(conditions, fmt.Sprintf(
					"%s IN (%s)", filter.Field, strings.Join(placeholders, ", "),
				))
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

	rows, err := p.getExecutor(ctx).QueryContext(ctx, query, tableName)
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

// getTableColumnTypes returns column-name → information_schema.data_type
// for a table. Used by Create/Update to pick the right serialization for
// auto-injected timestamp fields (BIGINT unix-ms vs TIMESTAMP WITH TIME ZONE).
func (p *PostgresOperations) getTableColumnTypes(ctx context.Context, tableName string) (map[string]string, error) {
	query := `
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_name = $1
	`
	rows, err := p.getExecutor(ctx).QueryContext(ctx, query, tableName)
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
// column at creation/update time. BIGINT columns (the new proto-aligned
// convention, e.g. session.date_created) receive unix ms; TIMESTAMP /
// TIMESTAMP WITH TIME ZONE columns receive a time.Time for the pq driver.
func autoTimestampValue(columnType string, now time.Time) any {
	if columnType == "bigint" {
		return now.UnixMilli()
	}
	return now
}

// ---------------------------------------------------------------------------
// SHADOW MODE (Plan 2, docs/plan/20260530-reflectionless-crud/, Phase 2)
//
// The descriptor registry (schema.Global, populated by schema.Build() at the
// container wirePoint) is the future source of column truth. This wave runs it
// in SHADOW: the descriptor-derived column set, the descriptor column types, AND
// the descriptor-derived auto-timestamp VALUE are computed ALONGSIDE the existing
// reflection path (getTableColumns / getTableColumnTypes), the two are asserted to
// agree, and on ANY disagreement a structured WARN is logged and the code FALLS
// BACK to the reflection-derived data — reflection remains authoritative this wave.
// Net behavior change is ZERO; the agreement metric is now observable (the Q-RC4
// 0-disagreement gate prerequisite for Phase 4). The unknown-column silent-drop LOG
// line is NOT flipped to ERROR — that is Phase 4 (a runtime soak gate), out of
// scope here.
//
// Phase-2 wiring (this wave, observe-only) over the Phase-1 shadow-rewire:
//   - Every shadow comparison (column-set, drop-set, timestamp-type, and the NEW
//     timestamp-VALUE check) records an AGREE / DISAGREE tick into the package-level
//     shadowMetric so the per-write agreement rate is observable — the Phase-4 flip
//     is gated on this metric reading 0 disagreements, which previously could only be
//     inferred from the ABSENCE of WARN lines. ShadowAgreementSnapshot() exposes the
//     running counts.
//   - shadowAssertAutoTimestamp closes the gap the Phase-1 type-axis check left open:
//     it computes the autoTimestampValue the DESCRIPTOR type would produce for the
//     same `now` and compares it (kind + concrete value) against the value the
//     REFLECTED type produces. The reflected value is still what is stamped into the
//     row; this only surfaces a divergence in the actually-written value.
//
// NO new env flag / fail-fast / boot-blocker is introduced: the per-call shadow is
// unconditionally WARN-only (reflection authoritative), matching the Phase-1
// convention. The only enforce gate in this plan is the boot-shot's
// SCHEMA_BOOTSHOT_ENFORCE (schema_validator.go), which is unrelated to this per-call
// path and untouched here.
// ---------------------------------------------------------------------------

// shadowCrudMetric is the structured agreement/disagreement counter for the
// per-write descriptor-vs-reflection shadow comparisons. Each shadow assertion
// records exactly one tick (AGREE or DISAGREE) per dimension it checks. The
// Phase-4 enforce flip (descriptor becomes authoritative; reflection fallback
// removed) is gated on Disagree reading 0 across a soak window — this counter is
// what makes that gate observable rather than inferred from the absence of WARN
// lines.
type shadowCrudMetric struct {
	agree    atomic.Int64
	disagree atomic.Int64
}

// shadowMetric is the package-level singleton. It is process-wide (the shadow path
// has no per-instance state) and lock-free.
var shadowMetric shadowCrudMetric

// ShadowAgreementSnapshot returns the running (agree, disagree) shadow-comparison
// counts. Exported so a soak harness / health endpoint can read the agreement rate
// without reaching into the unexported counter. Observe-only.
func ShadowAgreementSnapshot() (agree, disagree int64) {
	return shadowMetric.agree.Load(), shadowMetric.disagree.Load()
}

// shadowRecord ticks the agreement metric and, on disagreement, emits a structured
// pipe-delimited WARN (mirroring the AUTHZ_WS_SHADOW_PASS shape in
// workspace_operations.go: a stable prefix + key=val fields). `detail` carries the
// already-formatted per-dimension diff. It NEVER returns an error and NEVER affects
// the write — reflection stays authoritative.
func shadowRecord(dimension, tableName string, agreed bool, detail string) {
	if agreed {
		shadowMetric.agree.Add(1)
		return
	}
	shadowMetric.disagree.Add(1)
	log.Printf("WARN shadow-crud | mode=SHADOW(reflection-authoritative) | dim=%s | table=%q | %s",
		dimension, tableName, detail)
}

// descriptorColumnSet returns (validColumns, ok) for tableName from the descriptor
// registry. ok=false when the table is not registry-covered (e.g. an out-of-scope
// allowlist table such as payment_method / session, or a not-yet-annotated GAP-B
// table) — callers then have nothing to shadow-assert and silently fall back to
// reflection. The key set is the descriptor column names, which are the proto
// snake_case TextNames and therefore directly comparable to post-normalizeKeys map
// keys / reflected information_schema column names (phase0 §b key-match invariant).
func descriptorColumnSet(tableName string) (map[string]bool, bool) {
	cols, ok := schema.ColsFor(tableName)
	if !ok {
		return nil, false
	}
	set := make(map[string]bool, len(cols))
	for _, c := range cols {
		set[c.Name] = true
	}
	return set, true
}

// shadowAssertColumnSet compares the descriptor-derived column set against the
// reflection-derived set for tableName, ticks the agreement metric, and on
// disagreement logs a structured WARN listing the per-side diffs. It NEVER returns
// an error and NEVER mutates the reflected set — reflection stays authoritative. A
// table absent from the registry is a no-op (nothing to assert; not counted —
// only registry-covered tables contribute to the agreement rate).
//
// The descriptor set is already proto-(db).ignore-stripped by schema.Classify
// (rule 3: an annotated *_string mirror never becomes a ColumnInfo), so it is
// apples-to-apples with the reflected information_schema set, which likewise never
// contains those proto-only mirror fields. No extra stripping is needed here.
func shadowAssertColumnSet(tableName string, reflected map[string]bool) {
	derived, ok := descriptorColumnSet(tableName)
	if !ok {
		return
	}
	var derivedOnly, reflectedOnly []string
	for col := range derived {
		if !reflected[col] {
			derivedOnly = append(derivedOnly, col)
		}
	}
	for col := range reflected {
		if !derived[col] {
			reflectedOnly = append(reflectedOnly, col)
		}
	}
	if len(derivedOnly) == 0 && len(reflectedOnly) == 0 {
		shadowRecord("column-set", tableName, true, "")
		return
	}
	sort.Strings(derivedOnly)
	sort.Strings(reflectedOnly)
	shadowRecord("column-set", tableName, false,
		fmt.Sprintf("derived_only=%v | reflected_only=%v | note=falling-back-to-reflection", derivedOnly, reflectedOnly))
}

// descriptorTimestampType returns the autoTimestampValue columnType string for a
// timestamp column, sourced from the descriptor registry, plus whether the
// descriptor knew the column. The descriptor's IsBigintMillis flag maps to the
// "bigint" sentinel autoTimestampValue's bigint→UnixMilli branch keys on; any
// other descriptor column maps to "" (the non-bigint / time.Time branch).
func descriptorTimestampType(tableName, column string) (string, bool) {
	ci, ok := schema.ColByName(tableName, column)
	if !ok {
		return "", false
	}
	if ci.IsBigintMillis {
		return "bigint", true
	}
	return "", true
}

// shadowTimestampType resolves the autoTimestampValue columnType for a timestamp
// column under SHADOW: it prefers the descriptor type, but cross-checks against the
// reflected information_schema data_type and, on mismatch, logs a WARN and FALLS
// BACK to the reflected type (reflection authoritative this wave). When the
// descriptor does not know the column it returns the reflected type unchanged.
func shadowTimestampType(tableName, column string, reflectedTypes map[string]string) string {
	reflected := reflectedTypes[column]
	derived, ok := descriptorTimestampType(tableName, column)
	if !ok {
		return reflected
	}
	// autoTimestampValue only distinguishes "bigint" from everything else, so we
	// compare on that axis: descriptor-bigint must match reflected information_schema
	// "bigint", and a non-bigint descriptor must match a non-bigint reflected type.
	derivedIsBigint := derived == "bigint"
	reflectedIsBigint := reflected == "bigint"
	if derivedIsBigint != reflectedIsBigint {
		shadowRecord("timestamp-type", tableName, false,
			fmt.Sprintf("column=%q | derived=%q | reflected=%q | note=falling-back-to-reflection", column, derived, reflected))
		return reflected
	}
	shadowRecord("timestamp-type", tableName, true, "")
	return reflected
}

// shadowAssertAutoTimestamp closes the gap the Phase-1 type-axis check left open: it
// computes the autoTimestampValue the DESCRIPTOR-derived type would produce for the
// SAME `now`, and compares it (Go kind + concrete value) against the value the
// REFLECTED type produces. The reflected value is what the caller has already
// stamped into the row; this only surfaces a divergence in the actually-written
// VALUE (e.g. descriptor says bigint-millis -> int64 ms while the live column is
// TIMESTAMPTZ -> time.Time). It ticks the agreement metric and WARNs on divergence.
//
// A table/column the descriptor does not know is a no-op (nothing to assert; not
// counted) — reflection silently drives the write, exactly as before. Observe-only;
// reflection stays authoritative.
func shadowAssertAutoTimestamp(tableName, column string, reflectedTypes map[string]string, now time.Time) {
	derivedType, ok := descriptorTimestampType(tableName, column)
	if !ok {
		return
	}
	reflectedType := reflectedTypes[column]
	derivedVal := autoTimestampValue(derivedType, now)
	reflectedVal := autoTimestampValue(reflectedType, now)
	if autoTimestampValuesEqual(derivedVal, reflectedVal) {
		shadowRecord("timestamp-value", tableName, true, "")
		return
	}
	shadowRecord("timestamp-value", tableName, false,
		fmt.Sprintf("column=%q | derived=%T(%v) | reflected=%T(%v) | note=reflection-value-still-written",
			column, derivedVal, derivedVal, reflectedVal, reflectedVal))
}

// autoTimestampValuesEqual compares two autoTimestampValue results on both Go type
// and concrete value. autoTimestampValue returns either int64 unix-ms (bigint) or a
// time.Time; a mismatch in EITHER the type or the value is a divergence. Pinned to
// the two shapes autoTimestampValue can return so the comparison stays exact rather
// than relying on reflect.DeepEqual across unrelated kinds.
func autoTimestampValuesEqual(a, b any) bool {
	switch av := a.(type) {
	case int64:
		bv, ok := b.(int64)
		return ok && av == bv
	case time.Time:
		bv, ok := b.(time.Time)
		return ok && av.Equal(bv)
	default:
		// autoTimestampValue never returns any other type; treat as divergent.
		return false
	}
}

// shadowAssertDropSet compares the would-be unknown-column DROP set computed
// against the DESCRIPTOR-derived valid set with the actual (reflected) drop set,
// for the same data keys. On disagreement it logs a parallel WARN. Behavior is
// unchanged: the reflected drop set (reflectedSkipped) still drives the write; this
// only surfaces where descriptor-authoritative dropping would differ. A table
// absent from the registry is a no-op. The unknown-column LOG is NOT flipped to
// ERROR this wave (Phase 4).
func shadowAssertDropSet(tableName string, data map[string]any, reflectedSkipped []string, skipID bool) {
	derived, ok := descriptorColumnSet(tableName)
	if !ok {
		return
	}
	var derivedDrop []string
	for column := range data {
		if skipID && column == "id" {
			continue
		}
		if !derived[column] {
			derivedDrop = append(derivedDrop, column)
		}
	}
	reflectedSet := make(map[string]bool, len(reflectedSkipped))
	for _, c := range reflectedSkipped {
		reflectedSet[c] = true
	}
	derivedSet := make(map[string]bool, len(derivedDrop))
	for _, c := range derivedDrop {
		derivedSet[c] = true
	}
	var derivedOnly, reflectedOnly []string
	for c := range derivedSet {
		if !reflectedSet[c] {
			derivedOnly = append(derivedOnly, c)
		}
	}
	for c := range reflectedSet {
		if !derivedSet[c] {
			reflectedOnly = append(reflectedOnly, c)
		}
	}
	if len(derivedOnly) == 0 && len(reflectedOnly) == 0 {
		shadowRecord("drop-set", tableName, true, "")
		return
	}
	sort.Strings(derivedOnly)
	sort.Strings(reflectedOnly)
	shadowRecord("drop-set", tableName, false,
		fmt.Sprintf("descriptor_would_drop_only=%v | reflection_dropped_only=%v | note=reflection-still-drives-the-write", derivedOnly, reflectedOnly))
}

// scanRowToMap scans a single row into a map with snake_case keys (matching DB columns).
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

// scanRowsToMap scans a single row from *sql.Rows into a map with snake_case keys.
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

// ConvertMillisToDateStr converts business date fields in a result map from
// int64 Unix millis (produced by normalizeValue) to ISO 8601 date strings
// ("YYYY-MM-DD"). This is needed because proto business date fields were
// migrated from int64 to string, but normalizeValue still converts all
// time.Time values to int64 millis (correct for audit timestamps).
//
// Call this on the map returned by Read/List BEFORE json.Marshal + protojson.Unmarshal.
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

// ConvertMillisToRFC3339 converts timestamp fields in a result map from
// int64 Unix millis (produced by normalizeValue) to RFC3339 strings, which
// is the format protojson expects for google.protobuf.Timestamp fields.
//
// Call this on the map returned by Read/List BEFORE json.Marshal + protojson.Unmarshal
// for any TIMESTAMPTZ column whose proto field is google.protobuf.Timestamp.
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
// Specifically, time.Time (from TIMESTAMPTZ) → int64 Unix millis,
// so protojson can unmarshal into int64 protobuf fields.
// Business date fields need an additional ConvertMillisToDateStr call
// to convert millis to ISO 8601 strings for string proto fields.
func normalizeValue(v any) any {
	switch t := v.(type) {
	case time.Time:
		if t.IsZero() {
			return nil
		}
		return t.UnixMilli()
	case []byte:
		// jsonb columns: unmarshal to native Go types so json.Marshal
		// produces proper JSON instead of base64-encoded strings
		var parsed any
		if err := json.Unmarshal(t, &parsed); err == nil {
			return parsed
		}
		return string(t)
	default:
		return v
	}
}

// generateUUID generates a simple UUID (simplified implementation)
func generateUUID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// RunWithTransaction executes a function within a database transaction
func (p *PostgresOperations) RunWithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
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

// WithTransaction returns a DatabaseOperation that routes all queries through
// the transaction stored in ctx. Implements interfaces.TransactionAware.
func (p *PostgresOperations) WithTransaction(ctx context.Context) interfaces.DatabaseOperation {
	return p
}

// SupportsTransactions implements interfaces.TransactionAware.
func (p *PostgresOperations) SupportsTransactions() bool {
	return true
}

// GetDB returns the underlying database connection
// This is used for executing raw SQL queries in repository implementations
func (p *PostgresOperations) GetDB() *sql.DB {
	return p.db
}

// getExecutor returns *sql.Tx if one is active in ctx, otherwise *sql.DB.
func (p *PostgresOperations) getExecutor(ctx context.Context) dbExecutor {
	tx, ok := operations.GetTransactionFromContext(ctx)
	if ok {
		if pgTx, ok := tx.(*PostgreSQLTransaction); ok && pgTx.State() == interfaces.TransactionStatePending {
			return pgTx.GetTx()
		}
	}
	return p.db
}

// GetExecutor returns *sql.Tx if one is active in ctx, otherwise *sql.DB.
// Entity adapters that build raw SQL (CTEs, JOINs) must call this instead
// of holding their own *sql.DB reference.
// The return type uses the shared interfaces.DBExecutor so that adapter
// packages (e.g. the entity package) can type-assert dbOps to a common
// executorProvider interface without each package defining its own copy.
func (p *PostgresOperations) GetExecutor(ctx context.Context) sqlexec.DBExecutor {
	return p.getExecutor(ctx)
}

// serializeValue converts map and slice values to JSON bytes so the SQL
// driver can store them in JSONB columns. Primitive types pass through.
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

// normalizeKeys converts all map keys from camelCase to snake_case.
// This ensures protojson-marshaled data (camelCase) maps correctly to
// PostgreSQL column names (snake_case).
//
// LOAD-BEARING for the descriptor-registry shadow path: normalizeKeys MUST run
// before any schema.ColsFor / schema.ColByName lookup in Create/Update/Delete.
// Descriptor column names are the proto snake_case TextNames (== camelToSnake of
// the protojson field name), so they only match the map keys once those keys are
// snake_cased here. This ordering is the phase0 §b key-match invariant.
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

// snakeToCamel converts snake_case to camelCase.
// This ensures DB column names (snake_case) map correctly to
// protojson field names (camelCase) for protobuf unmarshalling.
func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// DenormalizeKeys converts all map keys from snake_case to camelCase.
// This ensures PostgreSQL column names (snake_case) map correctly to
// protojson field names (camelCase) for protobuf unmarshalling.
// Exported for use by entity adapters that convert DB results to protobuf.
func DenormalizeKeys(data map[string]any) map[string]any {
	result := make(map[string]any, len(data))
	for key, value := range data {
		result[snakeToCamel(key)] = value
	}
	return result
}
