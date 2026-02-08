//go:build postgres

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	licensepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/license"
)

// PostgresLicenseRepository implements license CRUD operations using PostgreSQL
type PostgresLicenseRepository struct {
	licensepb.UnimplementedLicenseDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", "license", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres license repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresLicenseRepository(dbOps, tableName), nil
	})
}

// NewPostgresLicenseRepository creates a new PostgreSQL license repository
func NewPostgresLicenseRepository(dbOps interfaces.DatabaseOperation, tableName string) licensepb.LicenseDomainServiceServer {
	if tableName == "" {
		tableName = "license" // default fallback
	}
	return &PostgresLicenseRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateLicense creates a new license using common PostgreSQL operations
func (r *PostgresLicenseRepository) CreateLicense(ctx context.Context, req *licensepb.CreateLicenseRequest) (*licensepb.CreateLicenseResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("license data is required")
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
		return nil, fmt.Errorf("failed to create license: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := protojson.Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensepb.CreateLicenseResponse{
		Data:    []*licensepb.License{license},
		Success: true,
	}, nil
}

// ReadLicense retrieves a license using common PostgreSQL operations
func (r *PostgresLicenseRepository) ReadLicense(ctx context.Context, req *licensepb.ReadLicenseRequest) (*licensepb.ReadLicenseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := protojson.Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensepb.ReadLicenseResponse{
		Data:    []*licensepb.License{license},
		Success: true,
	}, nil
}

// UpdateLicense updates a license using common PostgreSQL operations
func (r *PostgresLicenseRepository) UpdateLicense(ctx context.Context, req *licensepb.UpdateLicenseRequest) (*licensepb.UpdateLicenseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("license ID is required")
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

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update license: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := protojson.Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensepb.UpdateLicenseResponse{
		Data:    []*licensepb.License{license},
		Success: true,
	}, nil
}

// DeleteLicense deletes a license using common PostgreSQL operations
func (r *PostgresLicenseRepository) DeleteLicense(ctx context.Context, req *licensepb.DeleteLicenseRequest) (*licensepb.DeleteLicenseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete license: %w", err)
	}

	return &licensepb.DeleteLicenseResponse{
		Success: true,
	}, nil
}

// ListLicenses lists licenses using common PostgreSQL operations
func (r *PostgresLicenseRepository) ListLicenses(ctx context.Context, req *licensepb.ListLicensesRequest) (*licensepb.ListLicensesResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list licenses: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var licenses []*licensepb.License
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		license := &licensepb.License{}
		if err := protojson.Unmarshal(resultJSON, license); err != nil {
			// Log error and continue with next item
			continue
		}
		licenses = append(licenses, license)
	}

	return &licensepb.ListLicensesResponse{
		Data:    licenses,
		Success: true,
	}, nil
}

// GetLicenseListPageData retrieves a paginated, filtered, sorted, and searchable list of licenses
func (r *PostgresLicenseRepository) GetLicenseListPageData(ctx context.Context, req *licensepb.GetLicenseListPageDataRequest) (*licensepb.GetLicenseListPageDataResponse, error) {
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

	// Extract search query
	searchQuery := ""
	if req.Search != nil && req.Search.Query != "" {
		searchQuery = "%" + req.Search.Query + "%"
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
		-- CTE 1: Apply search filter on license
		search_filtered AS (
			SELECT l.*
			FROM license l
			WHERE l.active = true
				AND ($1::text = '' OR
					l.license_key ILIKE $1 OR
					l.assignee_name ILIKE $1)
		),

		-- CTE 2: Apply sorting
		sorted AS (
			SELECT * FROM search_filtered
			ORDER BY
				CASE WHEN $4 = 'license_key' AND $5 = 'ASC' THEN license_key END ASC,
				CASE WHEN $4 = 'license_key' AND $5 = 'DESC' THEN license_key END DESC,
				CASE WHEN ($4 = 'date_created' OR $4 = '') AND $5 = 'DESC' THEN date_created END DESC,
				CASE WHEN $4 = 'date_created' AND $5 = 'ASC' THEN date_created END ASC,
				CASE WHEN $4 = 'status' AND $5 = 'ASC' THEN status END ASC,
				CASE WHEN $4 = 'status' AND $5 = 'DESC' THEN status END DESC
		),

		-- CTE 3: Calculate total count for pagination
		total_count AS (
			SELECT count(*) as total FROM sorted
		)

		-- Final SELECT with pagination
		SELECT
			s.id,
			s.subscription_id,
			s.plan_id,
			s.license_key,
			s.external_key,
			s.license_type,
			s.status,
			s.date_valid_from,
			s.date_valid_from_string,
			s.date_valid_until,
			s.date_valid_until_string,
			s.assignee_id,
			s.assignee_type,
			s.assignee_name,
			s.assigned_by,
			s.date_assigned,
			s.date_assigned_string,
			s.sequence_number,
			s.date_created,
			s.date_modified,
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
		searchQuery,   // $1
		limit,         // $2
		offset,        // $3
		sortField,     // $4
		sortDirection, // $5
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetLicenseListPageData query: %w", err)
	}
	defer rows.Close()

	var licenses []*licensepb.License
	var totalCount int32

	for rows.Next() {
		var (
			id                   string
			subscriptionID       string
			planID               string
			licenseKey           string
			externalKey          sql.NullString
			licenseType          int32
			status               int32
			dateValidFrom        sql.NullInt64
			dateValidFromString  sql.NullString
			dateValidUntil       sql.NullInt64
			dateValidUntilString sql.NullString
			assigneeID           sql.NullString
			assigneeType         sql.NullString
			assigneeName         sql.NullString
			assignedBy           sql.NullString
			dateAssigned         sql.NullInt64
			dateAssignedString   sql.NullString
			sequenceNumber       sql.NullInt32
			dateCreated          sql.NullInt64
			dateCreatedString    sql.NullString
			dateModified         sql.NullInt64
			dateModifiedString   sql.NullString
			active               bool
			rowTotalCount        int32
		)

		err := rows.Scan(
			&id,
			&subscriptionID,
			&planID,
			&licenseKey,
			&externalKey,
			&licenseType,
			&status,
			&dateValidFrom,
			&dateValidFromString,
			&dateValidUntil,
			&dateValidUntilString,
			&assigneeID,
			&assigneeType,
			&assigneeName,
			&assignedBy,
			&dateAssigned,
			&dateAssignedString,
			&sequenceNumber,
			&dateCreated,
			&dateModified,
			&active,
			&rowTotalCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan license row: %w", err)
		}

		totalCount = rowTotalCount

		// Build license message
		license := &licensepb.License{
			Id:             id,
			SubscriptionId: subscriptionID,
			PlanId:         planID,
			LicenseKey:     licenseKey,
			LicenseType:    licensepb.LicenseType(licenseType),
			Status:         licensepb.LicenseStatus(status),
			Active:         active,
		}

		if externalKey.Valid {
			license.ExternalKey = &externalKey.String
		}
		if dateValidFrom.Valid {
			license.DateValidFrom = &dateValidFrom.Int64
		}
		if dateValidFromString.Valid {
			license.DateValidFromString = &dateValidFromString.String
		}
		if dateValidUntil.Valid {
			license.DateValidUntil = &dateValidUntil.Int64
		}
		if dateValidUntilString.Valid {
			license.DateValidUntilString = &dateValidUntilString.String
		}
		if assigneeID.Valid {
			license.AssigneeId = &assigneeID.String
		}
		if assigneeType.Valid {
			license.AssigneeType = &assigneeType.String
		}
		if assigneeName.Valid {
			license.AssigneeName = &assigneeName.String
		}
		if assignedBy.Valid {
			license.AssignedBy = &assignedBy.String
		}
		if dateAssigned.Valid {
			license.DateAssigned = &dateAssigned.Int64
		}
		if dateAssignedString.Valid {
			license.DateAssignedString = &dateAssignedString.String
		}
		if sequenceNumber.Valid {
			seqNum := sequenceNumber.Int32
			license.SequenceNumber = &seqNum
		}
		if dateCreated.Valid {
			license.DateCreated = &dateCreated.Int64
		}
		if dateCreatedString.Valid {
			license.DateCreatedString = &dateCreatedString.String
		}
		if dateModified.Valid {
			license.DateModified = &dateModified.Int64
		}
		if dateModifiedString.Valid {
			license.DateModifiedString = &dateModifiedString.String
		}

		licenses = append(licenses, license)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating license rows: %w", err)
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

	return &licensepb.GetLicenseListPageDataResponse{
		Success:     true,
		LicenseList: licenses,
		Pagination:  paginationResponse,
	}, nil
}

// GetLicenseItemPageData retrieves a single license with all related data
func (r *PostgresLicenseRepository) GetLicenseItemPageData(ctx context.Context, req *licensepb.GetLicenseItemPageDataRequest) (*licensepb.GetLicenseItemPageDataResponse, error) {
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.LicenseId)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := protojson.Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensepb.GetLicenseItemPageDataResponse{
		Success: true,
		License: license,
	}, nil
}

// AssignLicense assigns a license to an assignee
func (r *PostgresLicenseRepository) AssignLicense(ctx context.Context, req *licensepb.AssignLicenseRequest) (*licensepb.AssignLicenseResponse, error) {
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}
	if req.AssigneeId == "" {
		return nil, fmt.Errorf("assignee ID is required")
	}

	// Read existing license
	result, err := r.dbOps.Read(ctx, r.tableName, req.LicenseId)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

	// Update assignment fields
	now := time.Now()
	result["assignee_id"] = req.AssigneeId
	result["assignee_type"] = req.AssigneeType
	if req.AssigneeName != nil {
		result["assignee_name"] = *req.AssigneeName
	}
	result["assigned_by"] = req.AssignedBy
	result["date_assigned"] = now.UnixMilli()
	result["date_assigned_string"] = now.Format(time.RFC3339)
	result["status"] = int32(licensepb.LicenseStatus_LICENSE_STATUS_ACTIVE)
	result["date_modified"] = now.UnixMilli()

	// Update document
	updatedResult, err := r.dbOps.Update(ctx, r.tableName, req.LicenseId, result)
	if err != nil {
		return nil, fmt.Errorf("failed to update license assignment: %w", err)
	}

	// Convert result to protobuf
	resultJSON, err := json.Marshal(updatedResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := protojson.Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensepb.AssignLicenseResponse{
		License: license,
		Success: true,
	}, nil
}

// RevokeLicenseAssignment revokes the assignment of a license
func (r *PostgresLicenseRepository) RevokeLicenseAssignment(ctx context.Context, req *licensepb.RevokeLicenseAssignmentRequest) (*licensepb.RevokeLicenseAssignmentResponse, error) {
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	// Read existing license
	result, err := r.dbOps.Read(ctx, r.tableName, req.LicenseId)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

	// Clear assignment fields
	now := time.Now()
	delete(result, "assignee_id")
	delete(result, "assignee_type")
	delete(result, "assignee_name")
	delete(result, "assigned_by")
	delete(result, "date_assigned")
	delete(result, "date_assigned_string")
	result["status"] = int32(licensepb.LicenseStatus_LICENSE_STATUS_REVOKED)
	result["date_modified"] = now.UnixMilli()

	// Update document
	updatedResult, err := r.dbOps.Update(ctx, r.tableName, req.LicenseId, result)
	if err != nil {
		return nil, fmt.Errorf("failed to update license: %w", err)
	}

	// Convert result to protobuf
	resultJSON, err := json.Marshal(updatedResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := protojson.Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensepb.RevokeLicenseAssignmentResponse{
		License: license,
		Success: true,
	}, nil
}

// ReassignLicense reassigns a license to a new assignee
func (r *PostgresLicenseRepository) ReassignLicense(ctx context.Context, req *licensepb.ReassignLicenseRequest) (*licensepb.ReassignLicenseResponse, error) {
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}
	if req.NewAssigneeId == "" {
		return nil, fmt.Errorf("new assignee ID is required")
	}

	// Read existing license
	result, err := r.dbOps.Read(ctx, r.tableName, req.LicenseId)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

	// Update assignment fields
	now := time.Now()
	result["assignee_id"] = req.NewAssigneeId
	result["assignee_type"] = req.NewAssigneeType
	if req.NewAssigneeName != nil {
		result["assignee_name"] = *req.NewAssigneeName
	}
	result["assigned_by"] = req.PerformedBy
	result["date_assigned"] = now.UnixMilli()
	result["date_assigned_string"] = now.Format(time.RFC3339)
	result["date_modified"] = now.UnixMilli()

	// Update document
	updatedResult, err := r.dbOps.Update(ctx, r.tableName, req.LicenseId, result)
	if err != nil {
		return nil, fmt.Errorf("failed to update license: %w", err)
	}

	// Convert result to protobuf
	resultJSON, err := json.Marshal(updatedResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := protojson.Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensepb.ReassignLicenseResponse{
		License: license,
		Success: true,
	}, nil
}

// SuspendLicense suspends a license
func (r *PostgresLicenseRepository) SuspendLicense(ctx context.Context, req *licensepb.SuspendLicenseRequest) (*licensepb.SuspendLicenseResponse, error) {
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	// Read existing license
	result, err := r.dbOps.Read(ctx, r.tableName, req.LicenseId)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

	// Update status
	now := time.Now()
	result["status"] = int32(licensepb.LicenseStatus_LICENSE_STATUS_SUSPENDED)
	result["date_modified"] = now.UnixMilli()

	// Update document
	updatedResult, err := r.dbOps.Update(ctx, r.tableName, req.LicenseId, result)
	if err != nil {
		return nil, fmt.Errorf("failed to update license: %w", err)
	}

	// Convert result to protobuf
	resultJSON, err := json.Marshal(updatedResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := protojson.Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensepb.SuspendLicenseResponse{
		License: license,
		Success: true,
	}, nil
}

// ReactivateLicense reactivates a suspended license
func (r *PostgresLicenseRepository) ReactivateLicense(ctx context.Context, req *licensepb.ReactivateLicenseRequest) (*licensepb.ReactivateLicenseResponse, error) {
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	// Read existing license
	result, err := r.dbOps.Read(ctx, r.tableName, req.LicenseId)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

	// Update status
	now := time.Now()
	result["status"] = int32(licensepb.LicenseStatus_LICENSE_STATUS_ACTIVE)
	result["date_modified"] = now.UnixMilli()

	// Update document
	updatedResult, err := r.dbOps.Update(ctx, r.tableName, req.LicenseId, result)
	if err != nil {
		return nil, fmt.Errorf("failed to update license: %w", err)
	}

	// Convert result to protobuf
	resultJSON, err := json.Marshal(updatedResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := protojson.Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensepb.ReactivateLicenseResponse{
		License: license,
		Success: true,
	}, nil
}

// ValidateLicenseAccess validates if a license grants access
func (r *PostgresLicenseRepository) ValidateLicenseAccess(ctx context.Context, req *licensepb.ValidateLicenseAccessRequest) (*licensepb.ValidateLicenseAccessResponse, error) {
	var result map[string]any
	var err error

	// Find by ID or license key
	if req.LicenseId != "" {
		result, err = r.dbOps.Read(ctx, r.tableName, req.LicenseId)
	} else if req.LicenseKey != nil && *req.LicenseKey != "" {
		// Search by license key - would need to implement a FindByField method
		// For now, use list and filter
		listResult, listErr := r.dbOps.List(ctx, r.tableName, nil)
		if listErr != nil {
			return nil, fmt.Errorf("failed to search licenses: %w", listErr)
		}
		for _, item := range listResult.Data {
			if licenseKey, ok := item["license_key"].(string); ok && licenseKey == *req.LicenseKey {
				result = item
				break
			}
		}
		if result == nil {
			err = fmt.Errorf("license not found")
		}
	} else {
		return nil, fmt.Errorf("license ID or license key is required")
	}

	if err != nil {
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			ValidationMessage: strPtr("License not found"),
			Success:           true,
		}, nil
	}

	// Convert to protobuf
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := protojson.Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	// Check if license is active
	if license.Status != licensepb.LicenseStatus_LICENSE_STATUS_ACTIVE {
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			License:           license,
			ValidationMessage: strPtr(fmt.Sprintf("License is not active, current status: %s", license.Status.String())),
			Success:           true,
		}, nil
	}

	// Check assignee if specified
	if req.AssigneeId != nil && *req.AssigneeId != "" {
		if license.AssigneeId == nil || *license.AssigneeId != *req.AssigneeId {
			return &licensepb.ValidateLicenseAccessResponse{
				IsValid:           false,
				License:           license,
				ValidationMessage: strPtr("License is not assigned to the specified assignee"),
				Success:           true,
			}, nil
		}
	}

	// Check validity dates
	now := time.Now().UnixMilli()
	if license.DateValidFrom != nil && *license.DateValidFrom > now {
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			License:           license,
			ValidationMessage: strPtr("License is not yet valid"),
			Success:           true,
		}, nil
	}
	if license.DateValidUntil != nil && *license.DateValidUntil < now {
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			License:           license,
			ValidationMessage: strPtr("License has expired"),
			Success:           true,
		}, nil
	}

	return &licensepb.ValidateLicenseAccessResponse{
		IsValid:           true,
		License:           license,
		ValidationMessage: strPtr("License is valid"),
		Success:           true,
	}, nil
}

// CreateLicensesFromPlan creates multiple licenses from a plan
func (r *PostgresLicenseRepository) CreateLicensesFromPlan(ctx context.Context, req *licensepb.CreateLicensesFromPlanRequest) (*licensepb.CreateLicensesFromPlanResponse, error) {
	if req.SubscriptionId == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}
	if req.PlanId == "" {
		return nil, fmt.Errorf("plan ID is required")
	}
	if req.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be greater than 0")
	}

	now := time.Now()
	createdLicenses := make([]*licensepb.License, 0, req.Quantity)

	// Determine license type
	licenseType := licensepb.LicenseType_LICENSE_TYPE_USER
	if req.DefaultLicenseType != nil && *req.DefaultLicenseType != "" {
		switch *req.DefaultLicenseType {
		case "device":
			licenseType = licensepb.LicenseType_LICENSE_TYPE_DEVICE
		case "tenant":
			licenseType = licensepb.LicenseType_LICENSE_TYPE_TENANT
		case "floating":
			licenseType = licensepb.LicenseType_LICENSE_TYPE_FLOATING
		}
	}

	for i := int32(0); i < req.Quantity; i++ {
		seqNum := i + 1
		subscriptionPrefix := req.SubscriptionId
		if len(subscriptionPrefix) > 8 {
			subscriptionPrefix = subscriptionPrefix[:8]
		}
		licenseKey := fmt.Sprintf("LIC-%s-%04d", subscriptionPrefix, seqNum)

		data := map[string]any{
			"subscription_id":      req.SubscriptionId,
			"plan_id":              req.PlanId,
			"license_key":          licenseKey,
			"license_type":         int32(licenseType),
			"status":               int32(licensepb.LicenseStatus_LICENSE_STATUS_PENDING),
			"sequence_number":      seqNum,
			"date_created":         now.UnixMilli(),
			"date_modified":        now.UnixMilli(),
			"active":               true,
		}

		// Create document
		result, err := r.dbOps.Create(ctx, r.tableName, data)
		if err != nil {
			return nil, fmt.Errorf("failed to create license %d: %w", i+1, err)
		}

		// Convert to protobuf
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal license %d to JSON: %w", i+1, err)
		}

		license := &licensepb.License{}
		if err := protojson.Unmarshal(resultJSON, license); err != nil {
			return nil, fmt.Errorf("failed to unmarshal license %d to protobuf: %w", i+1, err)
		}

		createdLicenses = append(createdLicenses, license)
	}

	return &licensepb.CreateLicensesFromPlanResponse{
		Licenses:     createdLicenses,
		CreatedCount: int32(len(createdLicenses)),
		Success:      true,
	}, nil
}

// strPtr returns a pointer to a string (helper function)
func strPtr(s string) *string {
	return &s
}

// NewLicenseRepository creates a new PostgreSQL license repository (old-style constructor)
func NewLicenseRepository(db *sql.DB, tableName string) licensepb.LicenseDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresLicenseRepository(dbOps, tableName)
}
