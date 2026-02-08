//go:build postgres

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	licensepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/license"
	licensehistorypb "leapfor.xyz/esqyma/golang/v1/domain/subscription/license_history"
)

// PostgresLicenseHistoryRepository implements license_history CRUD operations using PostgreSQL
type PostgresLicenseHistoryRepository struct {
	licensehistorypb.UnimplementedLicenseHistoryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", "license_history", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres license_history repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
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
	if err := protojson.Unmarshal(resultJSON, licenseHistory); err != nil {
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
	if err := protojson.Unmarshal(resultJSON, licenseHistory); err != nil {
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
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
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
		if err := protojson.Unmarshal(resultJSON, licenseHistory); err != nil {
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

	// Build the CTE query
	query := `
		WITH
		-- CTE 1: Apply license_id filter
		filtered AS (
			SELECT lh.*
			FROM license_history lh
			WHERE lh.active = true
				AND ($1::text = '' OR lh.license_id = $1)
		),

		-- CTE 2: Apply sorting
		sorted AS (
			SELECT * FROM filtered
			ORDER BY
				CASE WHEN ($4 = 'date_created' OR $4 = '') AND $5 = 'DESC' THEN date_created END DESC,
				CASE WHEN $4 = 'date_created' AND $5 = 'ASC' THEN date_created END ASC,
				CASE WHEN $4 = 'action' AND $5 = 'ASC' THEN action END ASC,
				CASE WHEN $4 = 'action' AND $5 = 'DESC' THEN action END DESC
		),

		-- CTE 3: Calculate total count for pagination
		total_count AS (
			SELECT count(*) as total FROM sorted
		)

		-- Final SELECT with pagination
		SELECT
			s.id,
			s.license_id,
			s.action,
			s.assignee_id,
			s.assignee_type,
			s.assignee_name,
			s.previous_assignee_id,
			s.previous_assignee_type,
			s.previous_assignee_name,
			s.performed_by,
			s.reason,
			s.notes,
			s.license_status_before,
			s.license_status_after,
			s.date_created,
			s.active,
			tc.total as _total_count
		FROM sorted s
		CROSS JOIN total_count tc
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
			dateCreated          int64
			dateCreatedString    string
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
			DateCreated:         dateCreated,
			DateCreatedString:   dateCreatedString,
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
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresLicenseHistoryRepository(dbOps, tableName)
}
