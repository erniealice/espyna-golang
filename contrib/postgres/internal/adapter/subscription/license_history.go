//go:build postgresql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	espynactx "github.com/erniealice/espyna-golang/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	licensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license"
	licensehistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license_history"
	"google.golang.org/protobuf/encoding/protojson"
)

// licenseHistorySortableSQLCols lists the SQL column names handled by CASE WHEN
// branches in the sorted CTE. Any unrecognised column triggers a loud error
// instead of silently producing no ORDER BY.
var licenseHistorySortableSQLCols = []string{
	"date_created",
	"action",
}

// licenseHistoryViewToSQLColMap translates view-facing sort column keys to the
// SQL column names used in the CTE. Columns absent from the map pass through unchanged.
var licenseHistoryViewToSQLColMap = map[string]string{}

// PostgresLicenseHistoryRepository implements license_history CRUD operations using PostgreSQL
type PostgresLicenseHistoryRepository struct {
	licensehistorypb.UnimplementedLicenseHistoryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.LicenseHistory, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres license_history repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresLicenseHistoryRepository(dbOps, tableName), nil
	})
}

// NewPostgresLicenseHistoryRepository creates a new PostgreSQL license_history repository
func NewPostgresLicenseHistoryRepository(dbOps interfaces.DatabaseOperation, tableName string) licensehistorypb.LicenseHistoryDomainServiceServer {
	if tableName == "" {
		tableName = "license_history" // default fallback
	}
	return &PostgresLicenseHistoryRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateLicenseHistory creates a new license history record using common PostgreSQL operations
func (r *PostgresLicenseHistoryRepository) CreateLicenseHistory(ctx context.Context, req *licensehistorypb.CreateLicenseHistoryRequest) (*licensehistorypb.CreateLicenseHistoryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("license history data is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create license history: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	licenseHistory := &licensehistorypb.LicenseHistory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, licenseHistory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensehistorypb.CreateLicenseHistoryResponse{
		Data:    []*licensehistorypb.LicenseHistory{licenseHistory},
		Success: true,
	}, nil
}

// ReadLicenseHistory retrieves a license history record using common PostgreSQL operations
func (r *PostgresLicenseHistoryRepository) ReadLicenseHistory(ctx context.Context, req *licensehistorypb.ReadLicenseHistoryRequest) (*licensehistorypb.ReadLicenseHistoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("license history ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read license history: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	licenseHistory := &licensehistorypb.LicenseHistory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, licenseHistory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensehistorypb.ReadLicenseHistoryResponse{
		Data:    []*licensehistorypb.LicenseHistory{licenseHistory},
		Success: true,
	}, nil
}

// ListLicenseHistory lists license history records using common PostgreSQL operations
func (r *PostgresLicenseHistoryRepository) ListLicenseHistory(ctx context.Context, req *licensehistorypb.ListLicenseHistoryRequest) (*licensehistorypb.ListLicenseHistoryResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list license history: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var histories []*licensehistorypb.LicenseHistory
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		licenseHistory := &licensehistorypb.LicenseHistory{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, licenseHistory); err != nil {
			// Log error and continue with next item
			continue
		}

		// Filter by license_id if provided
		if req.LicenseId != nil && *req.LicenseId != "" {
			if licenseHistory.LicenseId != *req.LicenseId {
				continue
			}
		}

		histories = append(histories, licenseHistory)
	}

	return &licensehistorypb.ListLicenseHistoryResponse{
		Data:    histories,
		Success: true,
	}, nil
}

// GetLicenseHistoryListPageData retrieves a paginated, filtered, sorted, and searchable list of license history records
func (r *PostgresLicenseHistoryRepository) GetLicenseHistoryListPageData(ctx context.Context, req *licensehistorypb.GetLicenseHistoryListPageDataRequest) (*licensehistorypb.GetLicenseHistoryListPageDataResponse, error) {
	// Extract pagination parameters with defaults
	limit := int32(20)
	page := int32(1)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
		if limit > 100 {
			limit = 100 // Cap at 100 items per page
		}
		if req.Pagination.GetOffset() != nil {
			page = req.Pagination.GetOffset().Page
			if page < 1 {
				page = 1
			}
		}
	}
	offset := (page - 1) * limit

	// Extract license_id filter
	licenseIdFilter := ""
	if req.LicenseId != nil && *req.LicenseId != "" {
		licenseIdFilter = *req.LicenseId
	}

	// Extract sort parameters with defaults
	sortField := "date_created"
	sortDirection := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == 1 { // DESC enum value
			sortDirection = "DESC"
		} else {
			sortDirection = "ASC"
		}
	}

	// Translate view-facing column key to SQL column name via ColMap.
	if mapped, ok := licenseHistoryViewToSQLColMap[sortField]; ok {
		sortField = mapped
	}

	// Loud-failure guard: reject any sort column not handled by the CASE WHEN
	// chain in the sorted CTE. Turns silent fall-through into an obvious error.
	if sortField != "" && !slices.Contains(licenseHistorySortableSQLCols, sortField) {
		return nil, fmt.Errorf("unknown sort column %q for entity %q (allowed: %v)", sortField, "license_history", licenseHistorySortableSQLCols)
	}

	// Workspace isolation: this method bypasses the WorkspaceAwareOperations
	// decorator (raw SQL via db.GetDB()), so we extract workspace_id from context
	// and filter explicitly. Empty wsID = service-to-service call → no scoping (A1).
	wsID := espynactx.ExtractWorkspaceIDFromContext(ctx)

	// Build the CTE query.
	// A1: scope to the caller's workspace. license_history has no workspace_id
	// column of its own (verified against the baseline schema), and unlike its
	// siblings it has no direct subscription FK — tenancy chains two hops:
	// license_history.license_id → license.id → license.subscription_id →
	// subscription.workspace_id. The predicate scopes on the joined subscription's
	// workspace_id. Empty wsID = service-to-service call → no scoping.
	// A10: COUNT(*) OVER () replaces the prior total_count CTE + CROSS JOIN,
	// computed over the full filtered set before LIMIT/OFFSET. The parameterized
	// CASE WHEN sort (guarded above by licenseHistorySortableSQLCols) moves into
	// the final SELECT so the window count still spans every filtered row.
	query := `
		WITH
		-- CTE 1: Apply license_id + workspace filter
		filtered AS (
			SELECT lh.*
			FROM license_history lh
			LEFT JOIN license l ON lh.license_id = l.id
			LEFT JOIN subscription s ON l.subscription_id = s.id
			WHERE lh.active = true
				AND ($1::text = '' OR lh.license_id = $1)
				AND ($6::text = '' OR s.workspace_id = $6::text)
		)

		-- Final SELECT with sorting, window count, and pagination
		SELECT
			f.id,
			f.license_id,
			f.action,
			f.assignee_id,
			f.assignee_type,
			f.assignee_name,
			f.previous_assignee_id,
			f.previous_assignee_type,
			f.previous_assignee_name,
			f.performed_by,
			f.reason,
			f.notes,
			f.license_status_before,
			f.license_status_after,
			f.date_created,
			f.active,
			COUNT(*) OVER () as _total_count
		FROM filtered f
		ORDER BY
			CASE WHEN ($4 = 'date_created' OR $4 = '') AND $5 = 'DESC' THEN f.date_created END DESC,
			CASE WHEN $4 = 'date_created' AND $5 = 'ASC' THEN f.date_created END ASC,
			CASE WHEN $4 = 'action' AND $5 = 'ASC' THEN f.action END ASC,
			CASE WHEN $4 = 'action' AND $5 = 'DESC' THEN f.action END DESC
		LIMIT $2 OFFSET $3
	`

	// Get DB connection from dbOps interface
	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	// Execute query
	rows, err := db.GetDB().QueryContext(ctx, query,
		licenseIdFilter, // $1
		limit,           // $2
		offset,          // $3
		sortField,       // $4
		sortDirection,   // $5
		wsID,            // $6
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetLicenseHistoryListPageData query: %w", err)
	}
	defer rows.Close()

	var histories []*licensehistorypb.LicenseHistory
	var totalCount int32

	for rows.Next() {
		var (
			id                   string
			licenseID            string
			action               int32
			assigneeID           sql.NullString
			assigneeType         sql.NullString
			assigneeName         sql.NullString
			previousAssigneeID   sql.NullString
			previousAssigneeType sql.NullString
			previousAssigneeName sql.NullString
			performedBy          string
			reason               sql.NullString
			notes                sql.NullString
			licenseStatusBefore  int32
			licenseStatusAfter   int32
			dateCreated          time.Time
			active               bool
			rowTotalCount        int32
		)

		err := rows.Scan(
			&id,
			&licenseID,
			&action,
			&assigneeID,
			&assigneeType,
			&assigneeName,
			&previousAssigneeID,
			&previousAssigneeType,
			&previousAssigneeName,
			&performedBy,
			&reason,
			&notes,
			&licenseStatusBefore,
			&licenseStatusAfter,
			&dateCreated,
			&active,
			&rowTotalCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan license history row: %w", err)
		}

		totalCount = rowTotalCount

		// Build license history message
		history := &licensehistorypb.LicenseHistory{
			Id:                  id,
			LicenseId:           licenseID,
			Action:              licensehistorypb.LicenseHistoryAction(action),
			PerformedBy:         performedBy,
			LicenseStatusBefore: licensepb.LicenseStatus(licenseStatusBefore),
			LicenseStatusAfter:  licensepb.LicenseStatus(licenseStatusAfter),
			DateCreated:         dateCreated.UnixMilli(),
			DateCreatedString:   dateCreated.UTC().Format(time.RFC3339),
			Active:              active,
		}

		if assigneeID.Valid {
			history.AssigneeId = &assigneeID.String
		}
		if assigneeType.Valid {
			history.AssigneeType = &assigneeType.String
		}
		if assigneeName.Valid {
			history.AssigneeName = &assigneeName.String
		}
		if previousAssigneeID.Valid {
			history.PreviousAssigneeId = &previousAssigneeID.String
		}
		if previousAssigneeType.Valid {
			history.PreviousAssigneeType = &previousAssigneeType.String
		}
		if previousAssigneeName.Valid {
			history.PreviousAssigneeName = &previousAssigneeName.String
		}
		if reason.Valid {
			history.Reason = &reason.String
		}
		if notes.Valid {
			history.Notes = &notes.String
		}

		histories = append(histories, history)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating license history rows: %w", err)
	}

	// Build pagination response
	totalPages := (totalCount + limit - 1) / limit
	hasNext := page < totalPages
	hasPrev := page > 1

	paginationResponse := &commonpb.PaginationResponse{
		TotalItems:  totalCount,
		CurrentPage: &page,
		TotalPages:  &totalPages,
		HasNext:     hasNext,
		HasPrev:     hasPrev,
	}

	return &licensehistorypb.GetLicenseHistoryListPageDataResponse{
		Success:            true,
		LicenseHistoryList: histories,
		Pagination:         paginationResponse,
	}, nil
}

// NewLicenseHistoryRepository creates a new PostgreSQL license_history repository (old-style constructor)
func NewLicenseHistoryRepository(db *sql.DB, tableName string) licensehistorypb.LicenseHistoryDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresLicenseHistoryRepository(dbOps, tableName)
}
