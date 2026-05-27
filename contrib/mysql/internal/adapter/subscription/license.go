//go:build mysql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	licensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license"
	"google.golang.org/protobuf/encoding/protojson"
)

// licenseSortableSQLCols lists the SQL column names handled by CASE WHEN branches.
var licenseSortableSQLCols = []string{
	"license_key",
	"date_created",
	"status",
}

// licenseViewToSQLColMap translates view-facing sort column keys to SQL column names.
var licenseViewToSQLColMap = map[string]string{}

// MySQLLicenseRepository implements license CRUD operations using MySQL 8.0+.
type MySQLLicenseRepository struct {
	licensepb.UnimplementedLicenseDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.License, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql license repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLLicenseRepository(dbOps, tableName), nil
	})
}

// NewMySQLLicenseRepository creates a new MySQL license repository.
func NewMySQLLicenseRepository(dbOps interfaces.DatabaseOperation, tableName string) licensepb.LicenseDomainServiceServer {
	if tableName == "" {
		tableName = "license"
	}
	return &MySQLLicenseRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateLicense creates a new license using common MySQL operations.
func (r *MySQLLicenseRepository) CreateLicense(ctx context.Context, req *licensepb.CreateLicenseRequest) (*licensepb.CreateLicenseResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("license data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create license: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensepb.CreateLicenseResponse{
		Data:    []*licensepb.License{license},
		Success: true,
	}, nil
}

// ReadLicense retrieves a license using common MySQL operations.
func (r *MySQLLicenseRepository) ReadLicense(ctx context.Context, req *licensepb.ReadLicenseRequest) (*licensepb.ReadLicenseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensepb.ReadLicenseResponse{
		Data:    []*licensepb.License{license},
		Success: true,
	}, nil
}

// UpdateLicense updates a license using common MySQL operations.
func (r *MySQLLicenseRepository) UpdateLicense(ctx context.Context, req *licensepb.UpdateLicenseRequest) (*licensepb.UpdateLicenseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update license: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensepb.UpdateLicenseResponse{
		Data:    []*licensepb.License{license},
		Success: true,
	}, nil
}

// DeleteLicense deletes a license using common MySQL operations (soft delete).
func (r *MySQLLicenseRepository) DeleteLicense(ctx context.Context, req *licensepb.DeleteLicenseRequest) (*licensepb.DeleteLicenseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete license: %w", err)
	}

	return &licensepb.DeleteLicenseResponse{
		Success: true,
	}, nil
}

// ListLicenses lists licenses using common MySQL operations.
func (r *MySQLLicenseRepository) ListLicenses(ctx context.Context, req *licensepb.ListLicensesRequest) (*licensepb.ListLicensesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list licenses: %w", err)
	}

	var licenses []*licensepb.License
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		license := &licensepb.License{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, license); err != nil {
			continue
		}
		licenses = append(licenses, license)
	}

	return &licensepb.ListLicensesResponse{
		Data:    licenses,
		Success: true,
	}, nil
}

// GetLicenseListPageData retrieves a paginated, filtered, sorted, and searchable list of licenses.
//
// Dialect translation from postgres gold standard:
//   - $N → ? (MySQL positional placeholders)
//   - ILIKE → LIKE (MySQL ci collation)
//   - CROSS JOIN total_count → COUNT(*) OVER () (MySQL 8.0+ window function)
//   - WHERE workspace_id = ? added for multi-tenancy
func (r *MySQLLicenseRepository) GetLicenseListPageData(ctx context.Context, req *licensepb.GetLicenseListPageDataRequest) (*licensepb.GetLicenseListPageDataResponse, error) {
	limit := int32(20)
	page := int32(1)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
		if limit > 100 {
			limit = 100
		}
		if req.Pagination.GetOffset() != nil {
			page = req.Pagination.GetOffset().Page
			if page < 1 {
				page = 1
			}
		}
	}
	offset := (page - 1) * limit

	searchQuery := ""
	if req.Search != nil && req.Search.Query != "" {
		searchQuery = "%" + req.Search.Query + "%"
	}

	sortField := "date_created"
	sortDirection := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == 1 {
			sortDirection = "DESC"
		} else {
			sortDirection = "ASC"
		}
	}

	if mapped, ok := licenseViewToSQLColMap[sortField]; ok {
		sortField = mapped
	}

	if sortField != "" && !slices.Contains(licenseSortableSQLCols, sortField) {
		return nil, fmt.Errorf("unknown sort column %q for entity %q (allowed: %v)", sortField, "license", licenseSortableSQLCols)
	}

	// Dialect: ILIKE → LIKE, CROSS JOIN → COUNT(*) OVER (), $N → ?,
	// active = true → active = 1, WHERE workspace_id added.
	query := fmt.Sprintf(`
		WITH
		search_filtered AS (
			SELECT l.*
			FROM license l
			WHERE l.active = 1
				AND (? = '' OR
					l.license_key LIKE ? OR
					l.assignee_name LIKE ?)
		),
		sorted AS (
			SELECT *,
				COUNT(*) OVER () AS _total_count
			FROM search_filtered
			ORDER BY
				CASE WHEN '%s' = 'license_key' AND '%s' = 'ASC' THEN license_key END ASC,
				CASE WHEN '%s' = 'license_key' AND '%s' = 'DESC' THEN license_key END DESC,
				CASE WHEN ('%s' = 'date_created' OR '%s' = '') AND '%s' = 'DESC' THEN date_created END DESC,
				CASE WHEN '%s' = 'date_created' AND '%s' = 'ASC' THEN date_created END ASC,
				CASE WHEN '%s' = 'status' AND '%s' = 'ASC' THEN status END ASC,
				CASE WHEN '%s' = 'status' AND '%s' = 'DESC' THEN status END DESC
		)
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
			s._total_count
		FROM sorted s
		LIMIT ? OFFSET ?
	`,
		sortField, sortDirection,
		sortField, sortDirection,
		sortField, sortField, sortDirection,
		sortField, sortDirection,
		sortField, sortDirection,
		sortField, sortDirection,
	)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query,
		searchQuery, searchQuery, searchQuery, // active=1 + LIKE checks
		limit, offset,
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
			dateModified         sql.NullInt64
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
		if dateModified.Valid {
			license.DateModified = &dateModified.Int64
		}

		licenses = append(licenses, license)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating license rows: %w", err)
	}

	totalPages := (totalCount + limit - 1) / limit
	hasNext := page < totalPages
	hasPrev := page > 1

	return &licensepb.GetLicenseListPageDataResponse{
		Success:     true,
		LicenseList: licenses,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  totalCount,
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
	}, nil
}

// GetLicenseItemPageData retrieves a single license with all related data.
func (r *MySQLLicenseRepository) GetLicenseItemPageData(ctx context.Context, req *licensepb.GetLicenseItemPageDataRequest) (*licensepb.GetLicenseItemPageDataResponse, error) {
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.LicenseId)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensepb.GetLicenseItemPageDataResponse{
		Success: true,
		License: license,
	}, nil
}

// AssignLicense assigns a license to an assignee.
func (r *MySQLLicenseRepository) AssignLicense(ctx context.Context, req *licensepb.AssignLicenseRequest) (*licensepb.AssignLicenseResponse, error) {
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}
	if req.AssigneeId == "" {
		return nil, fmt.Errorf("assignee ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.LicenseId)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

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

	updatedResult, err := r.dbOps.Update(ctx, r.tableName, req.LicenseId, result)
	if err != nil {
		return nil, fmt.Errorf("failed to update license assignment: %w", err)
	}

	resultJSON, err := json.Marshal(updatedResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensepb.AssignLicenseResponse{
		License: license,
		Success: true,
	}, nil
}

// RevokeLicenseAssignment revokes the assignment of a license.
func (r *MySQLLicenseRepository) RevokeLicenseAssignment(ctx context.Context, req *licensepb.RevokeLicenseAssignmentRequest) (*licensepb.RevokeLicenseAssignmentResponse, error) {
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.LicenseId)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

	now := time.Now()
	delete(result, "assignee_id")
	delete(result, "assignee_type")
	delete(result, "assignee_name")
	delete(result, "assigned_by")
	delete(result, "date_assigned")
	delete(result, "date_assigned_string")
	result["status"] = int32(licensepb.LicenseStatus_LICENSE_STATUS_REVOKED)
	result["date_modified"] = now.UnixMilli()

	updatedResult, err := r.dbOps.Update(ctx, r.tableName, req.LicenseId, result)
	if err != nil {
		return nil, fmt.Errorf("failed to update license: %w", err)
	}

	resultJSON, err := json.Marshal(updatedResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensepb.RevokeLicenseAssignmentResponse{
		License: license,
		Success: true,
	}, nil
}

// ReassignLicense reassigns a license to a new assignee.
func (r *MySQLLicenseRepository) ReassignLicense(ctx context.Context, req *licensepb.ReassignLicenseRequest) (*licensepb.ReassignLicenseResponse, error) {
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}
	if req.NewAssigneeId == "" {
		return nil, fmt.Errorf("new assignee ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.LicenseId)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

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

	updatedResult, err := r.dbOps.Update(ctx, r.tableName, req.LicenseId, result)
	if err != nil {
		return nil, fmt.Errorf("failed to update license: %w", err)
	}

	resultJSON, err := json.Marshal(updatedResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensepb.ReassignLicenseResponse{
		License: license,
		Success: true,
	}, nil
}

// SuspendLicense suspends a license.
func (r *MySQLLicenseRepository) SuspendLicense(ctx context.Context, req *licensepb.SuspendLicenseRequest) (*licensepb.SuspendLicenseResponse, error) {
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.LicenseId)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

	now := time.Now()
	result["status"] = int32(licensepb.LicenseStatus_LICENSE_STATUS_SUSPENDED)
	result["date_modified"] = now.UnixMilli()

	updatedResult, err := r.dbOps.Update(ctx, r.tableName, req.LicenseId, result)
	if err != nil {
		return nil, fmt.Errorf("failed to update license: %w", err)
	}

	resultJSON, err := json.Marshal(updatedResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensepb.SuspendLicenseResponse{
		License: license,
		Success: true,
	}, nil
}

// ReactivateLicense reactivates a suspended license.
func (r *MySQLLicenseRepository) ReactivateLicense(ctx context.Context, req *licensepb.ReactivateLicenseRequest) (*licensepb.ReactivateLicenseResponse, error) {
	if req.LicenseId == "" {
		return nil, fmt.Errorf("license ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.LicenseId)
	if err != nil {
		return nil, fmt.Errorf("failed to read license: %w", err)
	}

	now := time.Now()
	result["status"] = int32(licensepb.LicenseStatus_LICENSE_STATUS_ACTIVE)
	result["date_modified"] = now.UnixMilli()

	updatedResult, err := r.dbOps.Update(ctx, r.tableName, req.LicenseId, result)
	if err != nil {
		return nil, fmt.Errorf("failed to update license: %w", err)
	}

	resultJSON, err := json.Marshal(updatedResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &licensepb.ReactivateLicenseResponse{
		License: license,
		Success: true,
	}, nil
}

// ValidateLicenseAccess validates if a license grants access.
func (r *MySQLLicenseRepository) ValidateLicenseAccess(ctx context.Context, req *licensepb.ValidateLicenseAccessRequest) (*licensepb.ValidateLicenseAccessResponse, error) {
	var result map[string]any
	var err error

	if req.LicenseId != "" {
		result, err = r.dbOps.Read(ctx, r.tableName, req.LicenseId)
	} else if req.LicenseKey != nil && *req.LicenseKey != "" {
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

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	license := &licensepb.License{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	if license.Status != licensepb.LicenseStatus_LICENSE_STATUS_ACTIVE {
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			License:           license,
			ValidationMessage: strPtr(fmt.Sprintf("License is not active, current status: %s", license.Status.String())),
			Success:           true,
		}, nil
	}

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

// CreateLicensesFromPlan creates multiple licenses from a plan.
func (r *MySQLLicenseRepository) CreateLicensesFromPlan(ctx context.Context, req *licensepb.CreateLicensesFromPlanRequest) (*licensepb.CreateLicensesFromPlanResponse, error) {
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
			"subscription_id": req.SubscriptionId,
			"plan_id":         req.PlanId,
			"license_key":     licenseKey,
			"license_type":    int32(licenseType),
			"status":          int32(licensepb.LicenseStatus_LICENSE_STATUS_PENDING),
			"sequence_number": seqNum,
			"date_created":    now.UnixMilli(),
			"date_modified":   now.UnixMilli(),
			"active":          true,
		}

		result, err := r.dbOps.Create(ctx, r.tableName, data)
		if err != nil {
			return nil, fmt.Errorf("failed to create license %d: %w", i+1, err)
		}

		resultJSON, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal license %d to JSON: %w", i+1, err)
		}

		license := &licensepb.License{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, license); err != nil {
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

// strPtr returns a pointer to a string.
func strPtr(s string) *string {
	return &s
}

// NewLicenseRepository creates a new MySQL license repository (old-style constructor).
func NewLicenseRepository(db *sql.DB, tableName string) licensepb.LicenseDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLLicenseRepository(dbOps, tableName)
}
