//go:build postgres

package payment

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
	paymentprofilepb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_profile"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "payment_profile", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres payment_profile repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresPaymentProfileRepository(dbOps, tableName), nil
	})
}

// PostgresPaymentProfileRepository implements payment profile CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_payment_profile_active ON payment_profile(active) - Filter active records
//   - CREATE INDEX idx_payment_profile_date_created ON payment_profile(date_created DESC) - Default sorting
//   - CREATE INDEX idx_payment_profile_payment_method_profile_id ON payment_profile_payment_method(payment_profile_id) - Junction table FK
//   - CREATE INDEX idx_payment_profile_payment_method_method_id ON payment_profile_payment_method(payment_method_id) - Junction table FK
//   - CREATE INDEX idx_payment_profile_payment_method_active ON payment_profile_payment_method(active) - Filter active junction records
//   - CREATE INDEX idx_payment_method_active ON payment_method(active) - Filter active payment methods
//
// TODO: Add comprehensive tests for GetPaymentProfileListPageData:
//   - Test with no search query (list all active payment profiles)
//   - Test pagination (page 1, page 2, page size variations)
//   - Test sorting (by different fields, ASC and DESC)
//   - Test with no matching results
//   - Test with inactive payment profiles (should be filtered out)
//   - Test with payment profiles having multiple payment methods
//   - Test with payment profiles having no payment methods
//   - Test with inactive payment methods (should be filtered out via junction table)
//
// TODO: Add comprehensive tests for GetPaymentProfileItemPageData:
//   - Test with valid payment profile ID (with associated payment methods)
//   - Test with valid payment profile ID (without associated payment methods)
//   - Test with non-existent payment profile ID
//   - Test with inactive payment profile (should return not found)
//   - Test with payment profile having inactive payment methods (should be filtered out)
//   - Test timestamp parsing for date_created and date_modified
type PostgresPaymentProfileRepository struct {
	paymentprofilepb.UnimplementedPaymentProfileDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresPaymentProfileRepository creates a new PostgreSQL payment profile repository
func NewPostgresPaymentProfileRepository(dbOps interfaces.DatabaseOperation, tableName string) paymentprofilepb.PaymentProfileDomainServiceServer {
	if tableName == "" {
		tableName = "payment_profile" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresPaymentProfileRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePaymentProfile creates a new payment profile using common PostgreSQL operations
func (r *PostgresPaymentProfileRepository) CreatePaymentProfile(ctx context.Context, req *paymentprofilepb.CreatePaymentProfileRequest) (*paymentprofilepb.CreatePaymentProfileResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("payment profile data is required")
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
		return nil, fmt.Errorf("failed to create payment profile: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	paymentProfile := &paymentprofilepb.PaymentProfile{}
	if err := protojson.Unmarshal(resultJSON, paymentProfile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &paymentprofilepb.CreatePaymentProfileResponse{
		Data: []*paymentprofilepb.PaymentProfile{paymentProfile},
	}, nil
}

// ReadPaymentProfile retrieves a payment profile using common PostgreSQL operations
func (r *PostgresPaymentProfileRepository) ReadPaymentProfile(ctx context.Context, req *paymentprofilepb.ReadPaymentProfileRequest) (*paymentprofilepb.ReadPaymentProfileResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment profile ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read payment profile: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	paymentProfile := &paymentprofilepb.PaymentProfile{}
	if err := protojson.Unmarshal(resultJSON, paymentProfile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &paymentprofilepb.ReadPaymentProfileResponse{
		Data: []*paymentprofilepb.PaymentProfile{paymentProfile},
	}, nil
}

// UpdatePaymentProfile updates a payment profile using common PostgreSQL operations
func (r *PostgresPaymentProfileRepository) UpdatePaymentProfile(ctx context.Context, req *paymentprofilepb.UpdatePaymentProfileRequest) (*paymentprofilepb.UpdatePaymentProfileResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment profile ID is required")
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
		return nil, fmt.Errorf("failed to update payment profile: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	paymentProfile := &paymentprofilepb.PaymentProfile{}
	if err := protojson.Unmarshal(resultJSON, paymentProfile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &paymentprofilepb.UpdatePaymentProfileResponse{
		Data: []*paymentprofilepb.PaymentProfile{paymentProfile},
	}, nil
}

// DeletePaymentProfile deletes a payment profile using common PostgreSQL operations
func (r *PostgresPaymentProfileRepository) DeletePaymentProfile(ctx context.Context, req *paymentprofilepb.DeletePaymentProfileRequest) (*paymentprofilepb.DeletePaymentProfileResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment profile ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete payment profile: %w", err)
	}

	return &paymentprofilepb.DeletePaymentProfileResponse{
		Success: true,
	}, nil
}

// ListPaymentProfiles lists payment profiles using common PostgreSQL operations
func (r *PostgresPaymentProfileRepository) ListPaymentProfiles(ctx context.Context, req *paymentprofilepb.ListPaymentProfilesRequest) (*paymentprofilepb.ListPaymentProfilesResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list payment profiles: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var paymentProfiles []*paymentprofilepb.PaymentProfile
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		paymentProfile := &paymentprofilepb.PaymentProfile{}
		if err := protojson.Unmarshal(resultJSON, paymentProfile); err != nil {
			// Log error and continue with next item
			continue
		}
		paymentProfiles = append(paymentProfiles, paymentProfile)
	}

	return &paymentprofilepb.ListPaymentProfilesResponse{
		Data: paymentProfiles,
	}, nil
}

// GetPaymentProfileListPageData retrieves payment profiles with advanced filtering, sorting, searching, and pagination using CTE
func (r *PostgresPaymentProfileRepository) GetPaymentProfileListPageData(
	ctx context.Context,
	req *paymentprofilepb.GetPaymentProfileListPageDataRequest,
) (*paymentprofilepb.GetPaymentProfileListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get payment profile list page data request is required")
	}

	// Default pagination values
	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		// Handle offset pagination
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	// Default sort
	sortField := "date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == 1 { // ASC = 1
			sortOrder = "ASC"
		}
	}

	// CTE Query - Single round-trip with enriched payment methods data
	// Performance Notes:
	// - INDEX RECOMMENDATION: Create index on payment_profile_payment_method.payment_profile_id (foreign key)
	// - INDEX RECOMMENDATION: Create index on payment_profile_payment_method.payment_method_id (foreign key)
	// - INDEX RECOMMENDATION: Create index on payment_profile_payment_method.active for filtering active records
	// - INDEX RECOMMENDATION: Create index on payment_method.active for filtering active records
	// - INDEX RECOMMENDATION: Create index on payment_profile.active for filtering active records
	// - INDEX RECOMMENDATION: Create index on payment_profile.date_created for default sorting
	query := `
		WITH payment_methods_agg AS (
			SELECT
				pppm.payment_profile_id,
				jsonb_agg(
					jsonb_build_object(
						'id', pm.id,
						'name', pm.name,
						'active', pm.active,
						'date_created', pm.date_created,
						'date_created_string', pm.date_created_string,
						'date_modified', pm.date_modified,
						'date_modified_string', pm.date_modified_string,
						'provider_name', pm.provider_name
					) ORDER BY pm.name
				) as payment_methods
			FROM payment_profile_payment_method pppm
			JOIN payment_method pm ON pppm.payment_method_id = pm.id
			WHERE pppm.active = true AND pm.active = true
			GROUP BY pppm.payment_profile_id
		),
		enriched AS (
			SELECT
				pp.id,
				pp.client_id,
				pp.payment_method_id,
				pp.active,
				pp.date_created,
				pp.date_created_string,
				pp.date_modified,
				pp.date_modified_string,
				COALESCE(pma.payment_methods, '[]'::jsonb) as payment_methods
			FROM payment_profile pp
			LEFT JOIN payment_methods_agg pma ON pp.id = pma.payment_profile_id
			WHERE pp.active = true
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT $1 OFFSET $2;
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query payment profile list page data: %w", err)
	}
	defer rows.Close()

	var paymentProfiles []*paymentprofilepb.PaymentProfile
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			clientId           string
			paymentMethodId    string
			active             bool
			dateCreated        *string
			dateCreatedString  *string
			dateModified       *string
			dateModifiedString *string
			paymentMethodsJSON []byte
			total              int64
		)

		err := rows.Scan(
			&id,
			&clientId,
			&paymentMethodId,
			&active,
			&dateCreated,
			&dateCreatedString,
			&dateModified,
			&dateModifiedString,
			&paymentMethodsJSON,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment profile row: %w", err)
		}

		totalCount = total

		paymentProfile := &paymentprofilepb.PaymentProfile{
			Id:              id,
			ClientId:        clientId,
			PaymentMethodId: paymentMethodId,
			Active:          active,
		}

		// Handle nullable timestamp fields
		if dateCreatedString != nil {
			paymentProfile.DateCreatedString = dateCreatedString
		}
		if dateModifiedString != nil {
			paymentProfile.DateModifiedString = dateModifiedString
		}

		// Parse timestamps if provided
		if dateCreated != nil && *dateCreated != "" {
			if ts, err := parsePaymentProfileTimestamp(*dateCreated); err == nil {
				paymentProfile.DateCreated = &ts
			}
		}
		if dateModified != nil && *dateModified != "" {
			if ts, err := parsePaymentProfileTimestamp(*dateModified); err == nil {
				paymentProfile.DateModified = &ts
			}
		}

		// Parse payment methods JSON array
		// Note: In the full implementation, you would parse this into repeated PaymentMethod fields
		// For now, storing the raw JSON for reference

		paymentProfiles = append(paymentProfiles, paymentProfile)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating payment profile rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &paymentprofilepb.GetPaymentProfileListPageDataResponse{
		PaymentProfileList: paymentProfiles,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
		Success: true,
	}, nil
}

// GetPaymentProfileItemPageData retrieves a single payment profile with enhanced item page data using CTE
func (r *PostgresPaymentProfileRepository) GetPaymentProfileItemPageData(
	ctx context.Context,
	req *paymentprofilepb.GetPaymentProfileItemPageDataRequest,
) (*paymentprofilepb.GetPaymentProfileItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get payment profile item page data request is required")
	}
	if req.PaymentProfileId == "" {
		return nil, fmt.Errorf("payment profile ID is required")
	}

	// CTE Query - Single round-trip with enriched payment methods data
	query := `
		WITH payment_methods_agg AS (
			SELECT
				pppm.payment_profile_id,
				jsonb_agg(
					jsonb_build_object(
						'id', pm.id,
						'name', pm.name,
						'active', pm.active,
						'date_created', pm.date_created,
						'date_created_string', pm.date_created_string,
						'date_modified', pm.date_modified,
						'date_modified_string', pm.date_modified_string,
						'provider_name', pm.provider_name
					) ORDER BY pm.name
				) as payment_methods
			FROM payment_profile_payment_method pppm
			JOIN payment_method pm ON pppm.payment_method_id = pm.id
			WHERE pppm.active = true AND pm.active = true
			GROUP BY pppm.payment_profile_id
		),
		enriched AS (
			SELECT
				pp.id,
				pp.client_id,
				pp.payment_method_id,
				pp.active,
				pp.date_created,
				pp.date_created_string,
				pp.date_modified,
				pp.date_modified_string,
				COALESCE(pma.payment_methods, '[]'::jsonb) as payment_methods
			FROM payment_profile pp
			LEFT JOIN payment_methods_agg pma ON pp.id = pma.payment_profile_id
			WHERE pp.id = $1 AND pp.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.PaymentProfileId)

	var (
		id                 string
		clientId           string
		paymentMethodId    string
		active             bool
		dateCreated        *string
		dateCreatedString  *string
		dateModified       *string
		dateModifiedString *string
		paymentMethodsJSON []byte
	)

	err := row.Scan(
		&id,
		&clientId,
		&paymentMethodId,
		&active,
		&dateCreated,
		&dateCreatedString,
		&dateModified,
		&dateModifiedString,
		&paymentMethodsJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &paymentprofilepb.GetPaymentProfileItemPageDataResponse{
				Success: false,
				Error: &commonpb.Error{
					Code:    "NOT_FOUND",
					Message: "payment profile not found",
				},
			}, nil
		}
		return nil, fmt.Errorf("failed to query payment profile item page data: %w", err)
	}

	paymentProfile := &paymentprofilepb.PaymentProfile{
		Id:              id,
		ClientId:        clientId,
		PaymentMethodId: paymentMethodId,
		Active:          active,
	}

	// Handle nullable timestamp fields
	if dateCreatedString != nil {
		paymentProfile.DateCreatedString = dateCreatedString
	}
	if dateModifiedString != nil {
		paymentProfile.DateModifiedString = dateModifiedString
	}

	// Parse timestamps if provided
	if dateCreated != nil && *dateCreated != "" {
		if ts, err := parsePaymentProfileTimestamp(*dateCreated); err == nil {
			paymentProfile.DateCreated = &ts
		}
	}
	if dateModified != nil && *dateModified != "" {
		if ts, err := parsePaymentProfileTimestamp(*dateModified); err == nil {
			paymentProfile.DateModified = &ts
		}
	}

	// Parse payment methods JSON array
	// Note: In the full implementation, you would parse this into repeated PaymentMethod fields
	// For now, storing the raw JSON for reference

	return &paymentprofilepb.GetPaymentProfileItemPageDataResponse{
		PaymentProfile: paymentProfile,
		Success:        true,
	}, nil
}

// parsePaymentProfileTimestamp parses a timestamp string into Unix milliseconds
func parsePaymentProfileTimestamp(timestampStr string) (int64, error) {
	// Try parsing as RFC3339 format first (most common)
	if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
		return t.UnixMilli(), nil
	}

	// Try other common formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.000Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t.UnixMilli(), nil
		}
	}

	return 0, fmt.Errorf("unable to parse timestamp: %s", timestampStr)
}

// NewPaymentProfileRepository creates a new PostgreSQL payment_profile repository (old-style constructor)
func NewPaymentProfileRepository(db *sql.DB, tableName string) paymentprofilepb.PaymentProfileDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresPaymentProfileRepository(dbOps, tableName)
}
